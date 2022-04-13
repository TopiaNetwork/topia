package consensus

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"

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
		log:            log,
		nodeID:         nodeID,
		proposeMsgChan: proposeMsgChan,
		ledger:         ledger,
		deliver:        deliver,
	}
}

func (v *consensusValidator) judgeLocalMaxPriBestForProposer(maxPri []byte) (bool, error) {
	v.syncPropMsgCached.RLock()
	defer v.syncPropMsgCached.RUnlock()

	if v.propMsgCached != nil {
		bhCached, err := v.propMsgCached.BlockHeadInfo()
		if err != nil {
			v.log.Errorf("Can't get cached propose msg bock head info: %v", err)
			return false, err
		}
		cachedMaxPri := bhCached.MaxPri

		if new(big.Int).SetBytes(maxPri).Cmp(new(big.Int).SetBytes(cachedMaxPri)) <= 0 {
			err = fmt.Errorf("Cached propose msg bock max pri bigger")
			v.log.Errorf("%v", err)
			return false, err
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
		propMsg.Version == v.propMsgCached.Version &&
		propMsg.Epoch == v.propMsgCached.Epoch &&
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

					bh, err := propMsg.BlockHeadInfo()
					if err != nil {
						v.log.Errorf("Can't get propose block head info: %v", err)
						return err
					}

					v.log.Infof("Validator received new propose message: self nodeID %s, epoch %d, stateVersion %d", v.nodeID, propMsg.Epoch, propMsg.StateVersion)

					if can := v.canProcessForwardProposeMsg(ctx, bh.MaxPri, propMsg); !can {
						err = errors.New("Can't vote received propose msg")
						v.log.Infof("%s", err.Error())
						return err
					}

					voteMsg, err := v.produceVoteMsg(propMsg)
					if err != nil {
						v.log.Errorf("Can't produce vote msg: err=%v", err)
						return err
					}
					if err = v.deliver.deliverVoteMessage(ctx, voteMsg, string(bh.Proposer)); err != nil {
						v.log.Errorf("Consensus deliver vote message err: %v", err)
						return err
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
