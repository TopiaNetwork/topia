package consensus

import (
	"github.com/TopiaNetwork/topia/codec"
	tptypes "github.com/TopiaNetwork/topia/common/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

type ConsensusHandler interface {
	VerifyBlock(block *tptypes.Block) error

	ProcessPropose(msg *ProposeMessage) error

	ProcessVote(msg *VoteMessage) error
}

type consensusHandler struct {
	log            tplog.Logger
	roundCh        chan *RoundInfo
	proposeMsgChan chan *ProposeMessage
	voteCollector  *consensusVoteCollector
	deliver        *messageDeliver
	csState        ConsensusStore
	marshaler      codec.Marshaler
}

func NewConsensusHandler(log tplog.Logger, roundCh chan *RoundInfo, proposeMsgChan chan *ProposeMessage, threshold uint64, csState ConsensusStore, marshaler codec.Marshaler) *consensusHandler {
	return &consensusHandler{
		log:            log,
		proposeMsgChan: proposeMsgChan,
		voteCollector:  newConsensusVoteCollector(log),
		csState:        csState,
		marshaler:      marshaler,
	}
}

func (handler *consensusHandler) VerifyBlock(block *tptypes.Block) error {
	panic("implement me")
}

func (handler *consensusHandler) ProcessPropose(msg *ProposeMessage) error {
	err := handler.csState.Commit()
	if err != nil {
		handler.log.Errorf("Can't commit block height =%d, err=%v", err)
		return err
	}

	handler.proposeMsgChan <- msg
	return nil
}

func (handler *consensusHandler) ProcessVote(msg *VoteMessage) error {
	aggSign, err := handler.voteCollector.tryAggregateSignAndAddVote(msg)
	if err != nil {
		handler.log.Errorf("Try to aggregate sign and add vote faild: err=%v", err)
		return err
	}

	var block tptypes.Block
	err = handler.marshaler.Unmarshal(msg.Block, &block)
	if err != nil {
		handler.log.Errorf("Unmarshal block failed: %v", err)
		return err
	}

	if aggSign != nil {
		err := handler.csState.Commit()
		if err != nil {
			handler.log.Errorf("Can't commit block height =%d, err=%v", block.Head.Height, err)
			return err
		}

		handler.csState.ClearBlockMiddleResult(msg.Round)
		handler.voteCollector.reset()

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
	}

	return nil
}
