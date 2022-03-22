package consensus

import (
	"context"
	tptypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
)

type ConsensusHandler interface {
	VerifyBlock(block *tptypes.Block) error

	ProcessPropose(msg *ProposeMessage) error

	ProcessVote(msg *VoteMessage) error

	ProcessCommit(msg *CommitMessage) error

	ProcessDKGPartPubKey(msg *DKGPartPubKeyMessage) error

	ProcessDKGDeal(msg *DKGDealMessage) error

	ProcessDKGDealResp(msg *DKGDealRespMessage) error

	updateDKGBls(dkgBls DKGBls)
}

type consensusHandler struct {
	log            tplog.Logger
	roundCh        chan *RoundInfo
	proposeMsgChan chan *ProposeMessage
	partPubKey     chan *DKGPartPubKeyMessage
	dealMsgCh      chan *DKGDealMessage
	dealRespMsgCh  chan *DKGDealRespMessage
	voteCollector  *consensusVoteCollector
	ledger         ledger.Ledger
	marshaler      codec.Marshaler
	deliver        messageDeliverI
}

func NewConsensusHandler(log tplog.Logger,
	roundCh chan *RoundInfo,
	proposeMsgChan chan *ProposeMessage,
	partPubKey chan *DKGPartPubKeyMessage,
	dealMsgCh chan *DKGDealMessage,
	dealRespMsgCh chan *DKGDealRespMessage,
	ledger ledger.Ledger,
	marshaler codec.Marshaler,
	deliver messageDeliverI) *consensusHandler {
	return &consensusHandler{
		log:            log,
		roundCh:        roundCh,
		proposeMsgChan: proposeMsgChan,
		partPubKey:     partPubKey,
		dealMsgCh:      dealMsgCh,
		dealRespMsgCh:  dealRespMsgCh,
		voteCollector:  newConsensusVoteCollector(log),
		ledger:         ledger,
		marshaler:      marshaler,
		deliver:        deliver,
	}
}

func (handler *consensusHandler) VerifyBlock(block *tptypes.Block) error {
	panic("implement me")
}

func (handler *consensusHandler) ProcessPropose(msg *ProposeMessage) error {
	handler.proposeMsgChan <- msg

	return nil
}

func (handler *consensusHandler) produceCommitMsg(msg *VoteMessage, aggSign []byte) (*CommitMessage, error) {
	return &CommitMessage{
		ChainID: msg.ChainID,
		Version: msg.Version,
		Epoch:   msg.Epoch,
		Round:   msg.Round,
		Proof:   msg.Proof,
		AggSign: aggSign,
		Block:   msg.Block,
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

		var block tptypes.Block
		err = handler.marshaler.Unmarshal(msg.Block, &block)
		if err != nil {
			handler.log.Errorf("Unmarshal block failed: %v", err)
			return err
		}
		block.Head.VoteAggSignature = aggSign
		err = handler.ledger.GetBlockStore().CommitBlock(&block)
		if err != nil {
			handler.log.Errorf("Can't commit block height =%d, err=%v", block.Head.Height, err)
			return err
		}

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
	var block tptypes.Block
	err := handler.marshaler.Unmarshal(msg.Block, &block)
	if err != nil {
		handler.log.Errorf("Unmarshal block failed: %v", err)
		return err
	}

	block.Head.VoteAggSignature = msg.AggSign

	err = handler.ledger.GetBlockStore().CommitBlock(&block)
	if err != nil {
		handler.log.Errorf("Can't commit block height =%d, err=%v", block.Head.Height, err)
		return err
	}

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
