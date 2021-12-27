package consensus

import (
	tptypes "github.com/TopiaNetwork/topia/common/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

type ConsensusHandler interface {
	VerifyBlock(block *tptypes.Block) error

	ProcessPropose(msg *ProposeMessage) error

	ProcessVote(msg *VoteMessage) error
}

type consensusHandler struct {
	log tplog.Logger
}

func NewConsensusHandler(log tplog.Logger) *consensusHandler {
	return &consensusHandler{
		log: log,
	}
}

func (handler *consensusHandler) VerifyBlock(block *tptypes.Block) error {
	panic("implement me")
}

func (handler *consensusHandler) ProcessPropose(msg *ProposeMessage) error {
	panic("implement me")
}

func (handler *consensusHandler) ProcessVote(msg *VoteMessage) error {
	panic("implement me")
}

