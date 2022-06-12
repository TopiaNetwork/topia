package consensus

import (
	"bytes"
	"context"
	"fmt"
	"github.com/TopiaNetwork/topia/codec"
	lru "github.com/hashicorp/golang-lru"
	"math/big"
	"strconv"
	"sync"
	"time"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/state"
)

type consensusValidator struct {
	log                   tplog.Logger
	nodeID                string
	proposeMsgChan        chan *ProposeMessage
	bestProposeMsgChan    chan *BestProposeMessage
	ledger                ledger.Ledger
	deliver               messageDeliverI
	marshaler             codec.Marshaler
	syncPropMsgCached     sync.RWMutex
	propMsgCached         *ProposeMessage
	syncBestPropMsgCached sync.RWMutex
	bestPropMsgCached     *BestProposeMessage
	propMsgVoted          *lru.Cache
}

func newConsensusValidator(log tplog.Logger, nodeID string, proposeMsgChan chan *ProposeMessage, bestProposeMsgChan chan *BestProposeMessage, ledger ledger.Ledger, deliver *messageDeliver, marshaler codec.Marshaler) *consensusValidator {
	propMsgVoted, _ := lru.New(5)
	return &consensusValidator{
		log:                tplog.CreateSubModuleLogger("validator", log),
		nodeID:             nodeID,
		proposeMsgChan:     proposeMsgChan,
		bestProposeMsgChan: bestProposeMsgChan,
		ledger:             ledger,
		deliver:            deliver,
		marshaler:          marshaler,
		propMsgVoted:       propMsgVoted,
	}
}

func (v *consensusValidator) updateLogger(log tplog.Logger) {
	v.log = log
}

func (v *consensusValidator) judgeLocalMaxPriBestForProposer(maxPri []byte, latestBlock *tpchaintypes.Block) (bool, error) {
	v.syncPropMsgCached.RLock()
	defer v.syncPropMsgCached.RUnlock()

	propMsg := v.propMsgCached

	if propMsg != nil {
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

func (v *consensusValidator) existProposeMsg(chainID tpchaintypes.ChainID, round uint64, stateVersion uint64, proposer string) bool {
	v.syncPropMsgCached.RLock()
	defer v.syncPropMsgCached.RUnlock()

	propMsg := v.propMsgCached

	if propMsg == nil {
		return false
	}

	if chainID == tpchaintypes.ChainID(propMsg.ChainID) &&
		round == propMsg.Round &&
		stateVersion == propMsg.StateVersion &&
		proposer == string(propMsg.Proposer) {

		return true
	}

	return false
}

func (v *consensusValidator) broadcastBestProposeMsg(ctx context.Context) error {
	v.syncPropMsgCached.RLock()
	defer v.syncPropMsgCached.RUnlock()

	propMsg := v.propMsgCached

	if propMsg == nil {
		return fmt.Errorf("Now propMsgCached nil and can't broadcast: self node %s", v.nodeID)
	}

	propMsgData, err := v.marshaler.Marshal(propMsg)
	if err != nil {
		return err
	}

	bestPropMsg := &BestProposeMessage{
		ChainID:      propMsg.ChainID,
		Version:      CONSENSUS_VER,
		Epoch:        propMsg.Epoch,
		Round:        propMsg.Round,
		StateVersion: propMsg.StateVersion,
		MaxPri:       propMsg.MaxPri,
		Proposer:     propMsg.Proposer,
		PropMsgData:  propMsgData,
	}

	err = v.deliver.deliverBestProposeMessage(ctx, bestPropMsg)
	if err != nil {
		v.log.Infof("Broadcast best propose message fail: state version %d, %v, self node %s", propMsg.StateVersion, err, v.nodeID)
		return err
	}

	v.log.Infof("Broadcast best propose message successfully: state version %d, self node %s", propMsg.StateVersion, v.nodeID)

	return nil
}

func (v *consensusValidator) produceVoteAndDeliver(ctx context.Context, propMsg *ProposeMessage) error {
	haveVoted, propMsgKey := v.haveVoted(propMsg)
	if haveVoted {
		err := fmt.Errorf("Have voted received propose message: propMsgKey %s, self node %s", propMsgKey, v.nodeID)
		v.log.Errorf("%v", err)
		return err
	}

	if string(propMsg.Proposer) == v.nodeID {
		err := fmt.Errorf("Self proposed block and ignore: self node %s", v.nodeID)
		v.log.Warnf("%s", err.Error())
		return err
	}

	voteMsg, err := v.produceVoteMsg(propMsg)
	if err != nil {
		v.log.Errorf("Can't produce vote msg: err=%v", err)
		return err
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
		return err
	}

	v.log.Infof("Message deliver ready, state version %d self node %s", propMsg.StateVersion, v.nodeID)

	if err = v.deliver.deliverVoteMessage(ctx, voteMsg, string(propMsg.Proposer)); err != nil {
		v.log.Errorf("Consensus deliver vote message err: %v", err)
		return err
	}

	v.addVotedPropMsg(propMsgKey)

	v.log.Infof("Deliver vote message, state version %d, to proposer %s, self node %s", propMsg.StateVersion, string(propMsg.Proposer), v.nodeID)

	return nil
}

func (v *consensusValidator) haveVoted(propMsg *ProposeMessage) (bool, string) {
	pmKey := string(propMsg.ChainID) + "@" +
		strconv.FormatUint(uint64(propMsg.Version), 10) + "@" +
		strconv.FormatUint(propMsg.Epoch, 10) + "@" +
		strconv.FormatUint(propMsg.Round, 10) + "@" +
		strconv.FormatUint(propMsg.StateVersion, 10) + "@" +
		string(propMsg.Proposer)
	if ok := v.propMsgVoted.Contains(pmKey); ok {
		return true, pmKey
	}

	return false, pmKey
}

func (v *consensusValidator) addVotedPropMsg(propMsgKey string) {
	v.propMsgVoted.Add(propMsgKey, struct{}{})
}

func (v *consensusValidator) checkValidBestProposeMsg(bestPropMsg *BestProposeMessage) bool {
	v.syncBestPropMsgCached.RLock()
	defer v.syncBestPropMsgCached.RUnlock()

	bestPropMsgCached := v.bestPropMsgCached
	if bestPropMsgCached != nil &&
		bytes.Compare(bestPropMsg.ChainID, bestPropMsgCached.ChainID) == 0 &&
		bestPropMsg.Version == bestPropMsgCached.Version &&
		bestPropMsg.Epoch == bestPropMsgCached.Epoch &&
		bestPropMsg.Round == bestPropMsgCached.Round &&
		bestPropMsg.StateVersion == bestPropMsgCached.StateVersion {
		if bytes.Compare(bestPropMsg.Proposer, bestPropMsgCached.Proposer) == 0 {
			v.log.Errorf("Have received best propose message from proposer %s", string(bestPropMsgCached.Proposer))
			return false
		}

		if new(big.Int).SetBytes(bestPropMsg.MaxPri).Cmp(new(big.Int).SetBytes(bestPropMsgCached.MaxPri)) < 0 {
			v.log.Errorf("Have received bigger pri of the best propose message and will discard new: received proposer %s, new prop proposer %s, self node %s", string(v.bestPropMsgCached.Proposer), string(bestPropMsg.Proposer), v.nodeID)
			return false
		}
	}

	return true
}

func (v *consensusValidator) receiveBestProposeMsgStart(ctx context.Context) {
	go func() {
		for {
			select {
			case bestPropMsg := <-v.bestProposeMsgChan:
				if !v.checkValidBestProposeMsg(bestPropMsg) {
					continue
				}

				v.log.Infof("Received the best propose message, state version %d, proposer %s, self node %s", bestPropMsg.StateVersion, string(bestPropMsg.Proposer), v.nodeID)

				var propMsg ProposeMessage
				err := v.marshaler.Unmarshal(bestPropMsg.PropMsgData, &propMsg)
				if err != nil {
					v.log.Errorf("Can't unmarshal bestPropMsg data: %v", err)
					continue
				}

				err = v.produceVoteAndDeliver(ctx, &propMsg)
				if err != nil {
					v.log.Errorf("Produce vote and deliver err after received the best propose message: state version %d, err %v, self node %s", bestPropMsg.StateVersion, err, v.nodeID)
					continue
				}

				v.syncBestPropMsgCached.Lock()
				v.bestPropMsgCached = bestPropMsg
				v.syncBestPropMsgCached.Unlock()
			case <-ctx.Done():
				v.log.Info("Validator receive best propose message exit")
			}
		}
	}()
}

func (v *consensusValidator) collectProposeMsgTimerStart(ctx context.Context) {
	go func(ctxSub context.Context) {
		v.log.Infof("Begin collectProposeMsgTimerStart, self node %s", v.nodeID)
		timer := time.NewTimer(500 * time.Millisecond)
		defer timer.Stop()

		select {
		case <-timer.C:
			v.log.Infof("CollectProposeMsgTimerStart timeout, self node %s", v.nodeID)
			v.syncPropMsgCached.RLock()
			defer v.syncPropMsgCached.RUnlock()

			propMsg := v.propMsgCached

			err := v.produceVoteAndDeliver(ctxSub, propMsg)
			if err != nil {
				v.log.Errorf("Produce vote and deliver err after received propose message: state version %d, err %v, self node %s", propMsg.StateVersion, err, v.nodeID)
				return
			}

		case <-ctxSub.Done():
			v.log.Infof("CollectProposeMsgTimerStart end: self node %s", v.nodeID)
			return
		}
	}(ctx)
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

		if string(v.propMsgCached.Proposer) == propProposer {
			v.log.Errorf("Same propose msg has existed from same proposer, so will discard: state version %d, proposer %s, self node %s", propMsg.StateVersion, propProposer, v.nodeID)
			return false, canCollectStart
		}

		cachedMaxPri := v.propMsgCached.MaxPri
		if new(big.Int).SetBytes(cachedMaxPri).Cmp(new(big.Int).SetBytes(maxPri)) < 0 {
			v.log.Infof("New prop pri bigger than cache: state version %d, cache proposer %s, new prop proposer %s, self node %s", propMsg.StateVersion, string(v.propMsgCached.Proposer), propProposer, v.nodeID)
			v.propMsgCached = propMsg
			if string(v.propMsgCached.Proposer) == v.nodeID {
				canCollectStart = true
			}
		} else {
			v.log.Errorf("Have received bigger pri propose block and can't process forward: state version %d, received proposer %s, new prop proposer %s, self node %s", propMsg.StateVersion, string(v.propMsgCached.Proposer), propProposer, v.nodeID)
			return false, canCollectStart
		}
	} else { //new propose message
		v.propMsgCached = propMsg
		canCollectStart = true
	}

	v.log.Infof("Validate sucessfully: cached state version %d, prop state version %d, canCollectStart %v, self node %s", v.propMsgCached.StateVersion, propMsg.StateVersion, canCollectStart, v.nodeID)

	return true, canCollectStart
}

func (v *consensusValidator) receiveProposeMsgStart(ctx context.Context) {
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

					v.log.Infof("Received propose message, state version %d, latest height %d, epoch %d, proposer %s, self node %s", propMsg.StateVersion, latestBlock.Head.Height, propMsg.Epoch, string(propMsg.Proposer), v.nodeID)

					canCollectStart := false
					ok := false
					if ok, canCollectStart = v.validateAndCollectProposeMsg(ctx, propMsg.MaxPri, string(propMsg.Proposer), propMsg); !ok {
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

func (v *consensusValidator) start(ctx context.Context) {
	v.receiveProposeMsgStart(ctx)

	v.receiveBestProposeMsgStart(ctx)
}
