package consensus

import (
	"context"
	"fmt"
	"github.com/AsynkronIT/protoactor-go/actor"
	tptypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/execution"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/state"
)

type ConsensusHandler interface {
	VerifyBlock(block *tptypes.Block) error

	ProcessPreparePackedMsgExe(msg *PreparePackedMessageExe) error

	ProcessPreparePackedMsgProp(msg *PreparePackedMessageProp) error

	ProcessPropose(msg *ProposeMessage) error

	ProcesExeResultValidateReq(actorCtx actor.Context, msg *ExeResultValidateReqMessage) error

	ProcessVote(msg *VoteMessage) error

	ProcessCommit(msg *CommitMessage) error

	ProcessDKGPartPubKey(msg *DKGPartPubKeyMessage) error

	ProcessDKGDeal(msg *DKGDealMessage) error

	ProcessDKGDealResp(msg *DKGDealRespMessage) error

	updateDKGBls(dkgBls DKGBls)
}

type consensusHandler struct {
	log                     tplog.Logger
	roundCh                 chan *RoundInfo
	preprePackedMsgExeChan  chan *PreparePackedMessageExe
	preprePackedMsgPropChan chan *PreparePackedMessageProp
	proposeMsgChan          chan *ProposeMessage
	partPubKey              chan *DKGPartPubKeyMessage
	dealMsgCh               chan *DKGDealMessage
	dealRespMsgCh           chan *DKGDealRespMessage
	voteCollector           *consensusVoteCollector
	ledger                  ledger.Ledger
	marshaler               codec.Marshaler
	deliver                 messageDeliverI
	exeScheduler            execution.ExecutionScheduler
}

func NewConsensusHandler(log tplog.Logger,
	roundCh chan *RoundInfo,
	preprePackedMsgExeChan chan *PreparePackedMessageExe,
	preprePackedMsgPropChan chan *PreparePackedMessageProp,
	proposeMsgChan chan *ProposeMessage,
	partPubKey chan *DKGPartPubKeyMessage,
	dealMsgCh chan *DKGDealMessage,
	dealRespMsgCh chan *DKGDealRespMessage,
	ledger ledger.Ledger,
	marshaler codec.Marshaler,
	deliver messageDeliverI,
	exeScheduler execution.ExecutionScheduler) *consensusHandler {
	return &consensusHandler{
		log:                     log,
		roundCh:                 roundCh,
		preprePackedMsgExeChan:  preprePackedMsgExeChan,
		preprePackedMsgPropChan: preprePackedMsgPropChan,
		proposeMsgChan:          proposeMsgChan,
		partPubKey:              partPubKey,
		dealMsgCh:               dealMsgCh,
		dealRespMsgCh:           dealRespMsgCh,
		voteCollector:           newConsensusVoteCollector(log),
		ledger:                  ledger,
		marshaler:               marshaler,
		deliver:                 deliver,
		exeScheduler:            exeScheduler,
	}
}

func (handler *consensusHandler) VerifyBlock(block *tptypes.Block) error {
	panic("implement me")
}

func (handler *consensusHandler) ProcessPreparePackedMsgExe(msg *PreparePackedMessageExe) error {
	csStateRN := state.CreateCompositionStateReadonly(handler.log, handler.ledger)
	defer csStateRN.Stop()

	id := handler.deliver.deliverNetwork().ID()

	activeExeIds, err := csStateRN.GetActiveExecutorIDs()
	if err != nil {
		handler.log.Errorf("Can't get active executor ids: %v", err)
		return err
	}

	if tpcmm.IsContainString(id, activeExeIds) {
		handler.preprePackedMsgExeChan <- msg
	} else {
		err = fmt.Errorf("Node %s not active executors, so will discard received prepare packed msg exe", id)
		handler.log.Errorf("%v", err)
	}

	return nil
}

func (handler *consensusHandler) ProcessPreparePackedMsgProp(msg *PreparePackedMessageProp) error {
	csStateRN := state.CreateCompositionStateReadonly(handler.log, handler.ledger)
	defer csStateRN.Stop()

	id := handler.deliver.deliverNetwork().ID()

	activeeProposeIds, err := csStateRN.GetActiveProposerIDs()
	if err != nil {
		handler.log.Errorf("Can't get active proposer ids: %v", err)
		return err
	}

	if tpcmm.IsContainString(id, activeeProposeIds) {
		handler.preprePackedMsgPropChan <- msg
	} else {
		err = fmt.Errorf("Node %s not active proposers, so will discard received prepare packed msg prop", id)
		handler.log.Errorf("%v", err)
	}

	return nil
}

func (handler *consensusHandler) ProcessPropose(msg *ProposeMessage) error {
	handler.proposeMsgChan <- msg

	return nil
}

func (handler *consensusHandler) ProcesExeResultValidateReq(actorCtx actor.Context, msg *ExeResultValidateReqMessage) error {
	txProofs, txRSProofs, err := handler.exeScheduler.PackedTxProofForValidity(context.Background(), msg.StateVersion, msg.TxHashs, msg.TxResultHashs)
	if err != nil {
		handler.log.Errorf("Get tx proofs and tx result proof err: %v", err)
		return err
	}

	validateResp := &ExeResultValidateRespMessage{
		ChainID:        msg.ChainID,
		Version:        msg.Version,
		Epoch:          msg.Epoch,
		Round:          msg.Round,
		StateVersion:   msg.StateVersion,
		TxProofs:       txProofs,
		TxResultProofs: txRSProofs,
	}

	return handler.deliver.deliverResultValidateRespMessage(actorCtx, validateResp)
}

func (handler *consensusHandler) produceCommitMsg(msg *VoteMessage, aggSign []byte) (*CommitMessage, error) {
	return &CommitMessage{
		ChainID:      msg.ChainID,
		Version:      msg.Version,
		Epoch:        msg.Epoch,
		Round:        msg.Round,
		StateVersion: msg.StateVersion,
		BlockHead:    msg.BlockHead,
	}, nil
}

func (handler *consensusHandler) ProcessVote(msg *VoteMessage) error {
	aggSign, err := handler.voteCollector.tryAggregateSignAndAddVote(msg)
	if err != nil {
		handler.log.Errorf("Try to aggregate sign and add vote faild: err=%v", err)
		return err
	}

	if aggSign != nil {
		commitMsg, _ := handler.produceCommitMsg(msg, aggSign)
		err := handler.deliver.deliverCommitMessage(context.Background(), commitMsg)
		if err != nil {
			handler.log.Panicf("Can't deliver commit message: %v", err)
		}

		var blockHead tptypes.BlockHead
		err = handler.marshaler.Unmarshal(msg.BlockHead, &blockHead)
		if err != nil {
			handler.log.Errorf("Unmarshal block failed: %v", err)
			return err
		}
		blockHead.VoteAggSignature = aggSign
		/*err = handler.ledger.GetBlockStore().CommitBlock(&block)
		if err != nil {
			handler.log.Errorf("Can't commit block height =%d, err=%v", block.Head.Height, err)
			return err
		}*/

		handler.voteCollector.reset()

		/*
			csProof := ConsensusProof{
				ParentBlockHash: block.Head.ParentBlockHash,
				Height:          block.Head.Height,
				AggSign:         aggSign,
			}

			roundInfo := &RoundInfo{
				Epoch:        block.Head.Epoch,
				LastRoundNum: block.Head.Round,
				CurRoundNum:  block.Head.Round + 1,
				Proof:        &csProof,
			}

			handler.roundCh <- roundInfo
		*/
	}

	return nil
}

func (handler *consensusHandler) ProcessCommit(msg *CommitMessage) error {
	var blockHead tptypes.BlockHead
	err := handler.marshaler.Unmarshal(msg.BlockHead, &blockHead)
	if err != nil {
		handler.log.Errorf("Unmarshal block failed: %v", err)
		return err
	}

	/*
		err = handler.ledger.GetBlockStore().CommitBlock(&block)
		if err != nil {
			handler.log.Errorf("Can't commit block height =%d, err=%v", blockHead.Height, err)
			return err
		}
	*/

	handler.voteCollector.reset()

	return nil
}

func (handler *consensusHandler) ProcessDKGPartPubKey(msg *DKGPartPubKeyMessage) error {
	handler.partPubKey <- msg

	return nil
}

func (handler *consensusHandler) ProcessDKGDeal(msg *DKGDealMessage) error {
	handler.dealMsgCh <- msg

	return nil
}

func (handler *consensusHandler) ProcessDKGDealResp(msg *DKGDealRespMessage) error {
	handler.dealRespMsgCh <- msg

	return nil
}

func (handler *consensusHandler) updateDKGBls(dkgBls DKGBls) {
	handler.voteCollector.updateDKGBls(dkgBls)
}
