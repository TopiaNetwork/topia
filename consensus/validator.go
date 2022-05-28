package consensus

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/state"
)

type consensusValidator struct {
	log               tplog.Logger
	nodeID            string
	proposeMsgChan    chan *ProposeMessage
	ledger            ledger.Ledger
	deliver           messageDeliverI
	syncPropMsgCached sync.RWMutex
	propMsgCached     *ProposeMessage
}

func newConsensusValidator(log tplog.Logger, nodeID string, proposeMsgChan chan *ProposeMessage, ledger ledger.Ledger, deliver *messageDeliver) *consensusValidator {
	return &consensusValidator{
		log:            tplog.CreateSubModuleLogger("validator", log),
		nodeID:         nodeID,
		proposeMsgChan: proposeMsgChan,
		ledger:         ledger,
		deliver:        deliver,
	}
}

func (v *consensusValidator) updateLogger(log tplog.Logger) {
	v.log = log
}

func (v *consensusValidator) judgeLocalMaxPriBestForProposer(maxPri []byte, latestBlock *tpchaintypes.Block) (bool, error) {
	v.syncPropMsgCached.RLock()
	defer v.syncPropMsgCached.RUnlock()

	if v.propMsgCached != nil {
		propMsg := v.propMsgCached

		bhCached, err := propMsg.BlockHeadInfo()
		if err != nil {
			v.log.Errorf("Can't get cached propose msg bock head info: %v", err)
			return false, err
		}
		if bytes.Equal(latestBlock.Head.ChainID, propMsg.ChainID) && bhCached.Height == (latestBlock.Head.Height+1) {
			cachedMaxPri := bhCached.MaxPri

			if new(big.Int).SetBytes(maxPri).Cmp(new(big.Int).SetBytes(cachedMaxPri)) <= 0 {
				err = fmt.Errorf("Cached propose msg bock max pri bigger,cached info: chainID=%s, height=%d, state version %d", propMsg.ChainID, bhCached.Height, propMsg.StateVersion)
				v.log.Errorf("%v", err)
				return false, err
			}
		}
	}

	return true, nil
}

func (v *consensusValidator) collectProposeMsgTimerStart(ctx context.Context) {
	go func(ctxSub context.Context) {
		v.log.Infof("Begin collectProposeMsgTimerStart, self node %s", v.nodeID)
		timer := time.NewTimer(2000 * time.Millisecond)
		defer timer.Stop()

		select {
		case <-timer.C:
			v.log.Infof("CollectProposeMsgTimerStart timeout, self node %s", v.nodeID)
			v.syncPropMsgCached.RLock()
			defer v.syncPropMsgCached.RUnlock()

			propMsg := v.propMsgCached

			bh, err := propMsg.BlockHeadInfo()
			if err != nil {
				v.log.Errorf("Can't get propose block head info in collectProposeMsgTimerStart: %v", err)
				return
			}

			if string(bh.Proposer) == v.nodeID {
				v.log.Warnf("Self proposed block and ignore: self node %s", v.nodeID)
				return
			}

			voteMsg, err := v.produceVoteMsg(propMsg)
			if err != nil {
				v.log.Errorf("Can't produce vote msg: err=%v", err)
				return
			}

			waitCount := 1
			for !v.deliver.isReady() && waitCount <= 10 {
				v.log.Warnf("Message deliver not ready now for delivering vote message, wait 50ms, no. %d", waitCount)
				time.Sleep(50 * time.Millisecond)
				waitCount++
			}
			if waitCount > 10 {
				err = fmt.Errorf("Finally nil dkgBls and can't deliver vote message, self node %s", v.nodeID)
				v.log.Errorf("%v", err)
				return
			}

			v.log.Infof("Message deliver ready, state version %d self node %s", propMsg.StateVersion, v.nodeID)

			if err = v.deliver.deliverVoteMessage(ctx, voteMsg, string(bh.Proposer)); err != nil {
				v.log.Errorf("Consensus deliver vote message err: %v", err)
				return
			}

			v.log.Infof("Deliver vote message, state version %d, to proposer %s, self node %s", propMsg.StateVersion, string(bh.Proposer), v.nodeID)
		case <-ctx.Done():
			v.log.Infof("collectProposeMsgTimerStart end: self node %s", v.nodeID)
			return
		}
	}(ctx)
}

func (v *consensusValidator) existProposeMsg(chainID tpchaintypes.ChainID, round uint64, stateVersion uint64, proposer string) bool {
	v.syncPropMsgCached.Lock()
	defer v.syncPropMsgCached.Unlock()

	if v.propMsgCached == nil {
		return false
	}

	bhCached, err := v.propMsgCached.BlockHeadInfo()
	if err != nil {
		v.log.Errorf("Can't get cached propose msg bock head info: %v", err)
		return false
	}
	if chainID == tpchaintypes.ChainID(v.propMsgCached.ChainID) &&
		round == v.propMsgCached.Round &&
		stateVersion == v.propMsgCached.StateVersion &&
		proposer == string(bhCached.Proposer) {

		return true
	}

	return false
}

func (v *consensusValidator) validateAndCollectProposeMsg(ctx context.Context, maxPri []byte, propProposer string, propMsg *ProposeMessage) (bool, bool) {
	canCollectStart := false

	exeRSValidate := newExecutionResultValidate(v.log, v.nodeID, v.deliver)
	executor, ok, err := exeRSValidate.Validate(ctx, propMsg)
	if !ok {
		v.log.Errorf("Propose block validate err by executor %s: %v", err, executor)
		return false, canCollectStart
	}

	v.syncPropMsgCached.Lock()
	defer v.syncPropMsgCached.Unlock()

	if v.propMsgCached != nil &&
		bytes.Compare(propMsg.ChainID, v.propMsgCached.ChainID) == 0 &&
		propMsg.Round == v.propMsgCached.Round &&
		propMsg.StateVersion == v.propMsgCached.StateVersion {
		bhCached, err := v.propMsgCached.BlockHeadInfo()
		if err != nil {
			v.log.Errorf("Can't get cached propose msg bock head info: %v", err)
			return false, canCollectStart
		}
		if string(bhCached.Proposer) == propProposer {
			v.log.Errorf("Same propose msg has existed from same proposer, so will discard: proposer %s, self node %s", propProposer, v.nodeID)
			return false, canCollectStart
		}

		cachedMaxPri := bhCached.MaxPri
		if new(big.Int).SetBytes(cachedMaxPri).Cmp(new(big.Int).SetBytes(maxPri)) < 0 {
			v.propMsgCached = propMsg
			if string(bhCached.Proposer) == v.nodeID {
				canCollectStart = true
			}
			v.log.Infof("New prop pri bigger than cache: cache proposer %s, new prop proposer %s, self node %s", string(bhCached.Proposer), propProposer, v.nodeID)
		} else {
			v.log.Errorf("Have received bigger pri propose block and can't process forward: received proposer %s, new prop proposer %s, self node %s", string(bhCached.Proposer), propProposer, v.nodeID)
			return false, canCollectStart
		}
	} else { //new propose message
		v.propMsgCached = propMsg
		canCollectStart = true
	}

	v.log.Infof("Validate sucessfully: cached state version %d, prop state version %d, canCollectStart %v, self node %s", v.propMsgCached.StateVersion, propMsg.StateVersion, canCollectStart, v.nodeID)

	return true, canCollectStart
}

func (v *consensusValidator) start(ctx context.Context) {
	go func() {
		for {
			select {
			case propMsg := <-v.proposeMsgChan:
				err := func() error {
					csStateRN := state.CreateCompositionStateReadonly(v.log, v.ledger)
					defer csStateRN.Stop()

					latestBlock, err := csStateRN.GetLatestBlock()
					if err != nil {
						v.log.Errorf("Can't get latest block head info: %v, self node %s", err, v.nodeID)
						return err
					}
					if propMsg.StateVersion <= latestBlock.Head.Height {
						err = fmt.Errorf("Received delayed propose message, state version %d, latest height %d,  self node %s", propMsg.StateVersion, latestBlock.Head.Height, v.nodeID)
						v.log.Warnf("%v", err)
						return err
					}

					bh, err := propMsg.BlockHeadInfo()
					if err != nil {
						v.log.Errorf("Can't get propose block head info: %v", err)
						return err
					}

					v.log.Infof("Received propose message, state version %d, latest height %d, epoch %d, proposer %s, self node %s", propMsg.StateVersion, latestBlock.Head.Height, propMsg.Epoch, string(bh.Proposer), v.nodeID)

					canCollectStart := false
					ok := false
					if ok, canCollectStart = v.validateAndCollectProposeMsg(ctx, bh.MaxPri, string(bh.Proposer), propMsg); !ok {
						err = fmt.Errorf("Can't vote received propose msg: state version %d, self node %s", propMsg.StateVersion, v.nodeID)
						v.log.Infof("%s", err.Error())
						return err
					}

					if canCollectStart {
						v.collectProposeMsgTimerStart(ctx)
					}

					return nil
				}()

				if err != nil {
					continue
				}
			case <-ctx.Done():
				v.log.Info("Consensus validator exit")
				return
			}
		}
	}()
}

func (v *consensusValidator) produceVoteMsg(msg *ProposeMessage) (*VoteMessage, error) {
	return &VoteMessage{
		ChainID:      msg.ChainID,
		Version:      CONSENSUS_VER,
		Epoch:        msg.Epoch,
		Round:        msg.Round,
		StateVersion: msg.StateVersion,
		BlockHead:    msg.BlockHead,
	}, nil
}
