package consensus

import (
	"context"
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/execution"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
)

type ConsensusHandler interface {
	VerifyBlock(block *tpchaintypes.Block) error

	ProcessPreparePackedMsgExe(msg *PreparePackedMessageExe) error

	ProcessPreparePackedMsgExeIndication(msg *PreparePackedMessageExeIndication) error

	ProcessPreparePackedMsgProp(msg *PreparePackedMessageProp) error

	ProcessPropose(msg *ProposeMessage) error

	ProcesExeResultValidateReq(actorCtx actor.Context, msg *ExeResultValidateReqMessage) error

	ProcessBestPropose(msg *BestProposeMessage) error

	ProcessVote(msg *VoteMessage) error

	ProcessCommit(msg *CommitMessage) error

	ProcessBlockAdded(block *tpchaintypes.Block) error

	ProcessEpochNew(epoch *tpcmm.EpochInfo) error

	ProcessDKGPartPubKey(msg *DKGPartPubKeyMessage) error

	ProcessDKGDeal(msg *DKGDealMessage) error

	ProcessDKGDealResp(msg *DKGDealRespMessage) error
}

type consensusHandler struct {
	log                         tplog.Logger
	epochNew                    chan *tpcmm.EpochInfo
	preprePackedMsgExeChan      chan *PreparePackedMessageExe
	preprePackedMsgExeIndicChan chan *PreparePackedMessageExeIndication
	preprePackedMsgPropChan     chan *PreparePackedMessageProp
	proposeMsgChan              chan *ProposeMessage
	bestProposeMsgChan          chan *BestProposeMessage
	voteMsgChan                 chan *VoteMessage
	commitMsgChan               chan *CommitMessage
	blockAddedEpochCh           chan *tpchaintypes.Block
	blockAddedProposerCh        chan *tpchaintypes.Block
	partPubKey                  chan *DKGPartPubKeyMessage
	dealMsgCh                   chan *DKGDealMessage
	dealRespMsgCh               chan *DKGDealRespMessage
	ledger                      ledger.Ledger
	marshaler                   codec.Marshaler
	deliver                     messageDeliverI
	exeScheduler                execution.ExecutionScheduler
	epochService                EpochService
}

func NewConsensusHandler(log tplog.Logger,
	epochNew chan *tpcmm.EpochInfo,
	preprePackedMsgExeChan chan *PreparePackedMessageExe,
	preprePackedMsgExeIndicChan chan *PreparePackedMessageExeIndication,
	preprePackedMsgPropChan chan *PreparePackedMessageProp,
	proposeMsgChan chan *ProposeMessage,
	bestProposeMsgChan chan *BestProposeMessage,
	voteMsgChan chan *VoteMessage,
	commitMsgChan chan *CommitMessage,
	blockAddedEpochCh chan *tpchaintypes.Block,
	blockAddedProposerCh chan *tpchaintypes.Block,
	partPubKey chan *DKGPartPubKeyMessage,
	dealMsgCh chan *DKGDealMessage,
	dealRespMsgCh chan *DKGDealRespMessage,
	ledger ledger.Ledger,
	marshaler codec.Marshaler,
	deliver messageDeliverI,
	exeScheduler execution.ExecutionScheduler,
	epochService EpochService) *consensusHandler {
	return &consensusHandler{
		log:                         log,
		epochNew:                    epochNew,
		preprePackedMsgExeChan:      preprePackedMsgExeChan,
		preprePackedMsgExeIndicChan: preprePackedMsgExeIndicChan,
		preprePackedMsgPropChan:     preprePackedMsgPropChan,
		proposeMsgChan:              proposeMsgChan,
		bestProposeMsgChan:          bestProposeMsgChan,
		voteMsgChan:                 voteMsgChan,
		commitMsgChan:               commitMsgChan,
		blockAddedEpochCh:           blockAddedEpochCh,
		blockAddedProposerCh:        blockAddedProposerCh,
		partPubKey:                  partPubKey,
		dealMsgCh:                   dealMsgCh,
		dealRespMsgCh:               dealRespMsgCh,
		ledger:                      ledger,
		marshaler:                   marshaler,
		deliver:                     deliver,
		exeScheduler:                exeScheduler,
		epochService:                epochService,
	}
}

func (handler *consensusHandler) VerifyBlock(block *tpchaintypes.Block) error {
	panic("implement me")
}

func (handler *consensusHandler) ProcessPreparePackedMsgExe(msg *PreparePackedMessageExe) error {
	id := handler.deliver.deliverNetwork().ID()

	activeExeIds := handler.epochService.GetActiveExecutorIDs()
	if tpcmm.IsContainString(id, activeExeIds) {
		handler.preprePackedMsgExeChan <- msg
	} else {
		err := fmt.Errorf("Node %s not active executors, so will discard received prepare packed msg exe", id)
		handler.log.Errorf("%v", err)
	}

	return nil
}

func (handler *consensusHandler) ProcessPreparePackedMsgExeIndication(msg *PreparePackedMessageExeIndication) error {
	id := handler.deliver.deliverNetwork().ID()

	activeExeIds := handler.epochService.GetActiveExecutorIDs()
	if tpcmm.IsContainString(id, activeExeIds) {
		handler.preprePackedMsgExeIndicChan <- msg
	} else {
		err := fmt.Errorf("Node %s not active executors, so will discard received prepare packed msg exe indication", id)
		handler.log.Errorf("%v", err)
	}

	return nil
}

func (handler *consensusHandler) ProcessPreparePackedMsgProp(msg *PreparePackedMessageProp) error {
	id := handler.deliver.deliverNetwork().ID()

	activeProposeIds := handler.epochService.GetActiveProposerIDs()
	if tpcmm.IsContainString(id, activeProposeIds) {
		handler.preprePackedMsgPropChan <- msg
	} else {
		err := fmt.Errorf("Node %s not active proposers, so will discard received prepare packed msg prop", id)
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

	validateResp := &ExeResultValidateRespMessage{
		ChainID:      msg.ChainID,
		Version:      msg.Version,
		Epoch:        msg.Epoch,
		Round:        msg.Round,
		StateVersion: msg.StateVersion,
	}

	if err != nil {
		handler.log.Errorf("Can't get tx proofs and tx result proof of ExeResultValidateReqMessage: state version %d,  err %v, self node %s", msg.StateVersion, err, handler.deliver.deliverNetwork().ID())
	} else {
		validateResp.TxProofs = txProofs
		validateResp.TxResultProofs = txRSProofs
	}

	return handler.deliver.deliverResultValidateRespMessage(actorCtx, validateResp, err)
}

func (handler *consensusHandler) ProcessBestPropose(msg *BestProposeMessage) error {
	handler.bestProposeMsgChan <- msg

	return nil
}

func (handler *consensusHandler) ProcessVote(msg *VoteMessage) error {
	handler.voteMsgChan <- msg

	return nil
}

func (handler *consensusHandler) ProcessCommit(msg *CommitMessage) error {
	handler.commitMsgChan <- msg

	return nil
}

func (handler *consensusHandler) ProcessBlockAdded(block *tpchaintypes.Block) error {
	handler.blockAddedEpochCh <- block
	handler.blockAddedProposerCh <- block

	return nil
}

func (handler *consensusHandler) ProcessEpochNew(epoch *tpcmm.EpochInfo) error {
	handler.epochNew <- epoch

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
