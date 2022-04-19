package consensus

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"math/big"
	"sync"
	"time"

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

func (v *consensusValidator) judgeLocalMaxPriBestForProposer(maxPri []byte, latestBlock *tpchaintypes.Block) (bool, error) {
	v.syncPropMsgCached.RLock()
	defer v.syncPropMsgCached.RUnlock()

	if v.propMsgCached != nil {
		bhCached, err := v.propMsgCached.BlockHeadInfo()
		if err != nil {
			v.log.Errorf("Can't get cached propose msg bock head info: %v", err)
			return false, err
		}
		if bytes.Equal(latestBlock.Head.ChainID, v.propMsgCached.ChainID) && bhCached.Height >= (latestBlock.Head.Height+1) {
			cachedMaxPri := bhCached.MaxPri

			if new(big.Int).SetBytes(maxPri).Cmp(new(big.Int).SetBytes(cachedMaxPri)) <= 0 {
				err = fmt.Errorf("Cached propose msg bock max pri bigger,cached info: chainID=%s, height=%d, state version %d", v.propMsgCached.ChainID, bhCached.Height, v.propMsgCached.StateVersion)
				v.log.Errorf("%v", err)
				return false, err
			}
		}
	}

	return true, nil
}

func (v *consensusValidator) canProcessForwardProposeMsg(ctx context.Context, maxPri []byte, propMsg *ProposeMessage) bool {
	exeRSValidate := newExecutionResultValidate(v.log, v.nodeID, v.deliver)
	executor, ok, err := exeRSValidate.Validate(ctx, propMsg)
	if !ok {
		v.log.Errorf("Propose block execution err: %v", err)
		return false
	}

	v.log.Debugf("Propose block execution ok and can forward to other validator: proposer=%s, executor=%d", v.nodeID, executor)

	v.syncPropMsgCached.Lock()
	defer v.syncPropMsgCached.Unlock()

	if v.propMsgCached != nil &&
		bytes.Compare(propMsg.ChainID, v.propMsgCached.ChainID) == 0 &&
		propMsg.Round == v.propMsgCached.Round {
		bhCached, err := v.propMsgCached.BlockHeadInfo()
		if err != nil {
			v.log.Errorf("Can't get cached propose msg bock head info: %v", err)
			return false
		}
		cachedMaxPri := bhCached.MaxPri
		if new(big.Int).SetBytes(cachedMaxPri).Cmp(new(big.Int).SetBytes(maxPri)) < 0 {
			v.propMsgCached = propMsg
		} else {
			v.log.Errorf("Received bigger pri poropose block and can't  forward to other validator: local proposer=%s, other proposer =%d", v.nodeID, string(bhCached.Proposer))
			return false
		}
	} else {
		v.propMsgCached = propMsg
	}

	return true
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

					v.log.Infof("Received propose message, state version %d, latest height %d, epoch %d, self node %s", propMsg.StateVersion, latestBlock.Head.Height, propMsg.Epoch, v.nodeID)

					bh, err := propMsg.BlockHeadInfo()
					if err != nil {
						v.log.Errorf("Can't get propose block head info: %v", err)
						return err
					}

					if can := v.canProcessForwardProposeMsg(ctx, bh.MaxPri, propMsg); !can {
						err = fmt.Errorf("Can't vote received propose msg: state version %d, self node %s", propMsg.StateVersion, v.nodeID)
						v.log.Infof("%s", err.Error())
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
						err := errors.New("Finally nil dkgBls and can't deliver vote message")
						v.log.Errorf("%v", err)
						return err
					}

					v.log.Infof("Message deliver ready, state version %d self node %s", propMsg.StateVersion, v.nodeID)

					if err = v.deliver.deliverVoteMessage(ctx, voteMsg, string(bh.Proposer)); err != nil {
						v.log.Errorf("Consensus deliver vote message err: %v", err)
						return err
					}

					v.log.Infof("Deliver vote message, state version %d self node %s", propMsg.StateVersion, v.nodeID)

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
