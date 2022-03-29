package mock

import (
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/consensus"
	tplog "github.com/TopiaNetwork/topia/log"
)

type MockConsensusHandler struct {
	log tplog.Logger
}

func (handler *MockConsensusHandler) VerifyBlock(block *tpchaintypes.Block) error {
	handler.log.Info("VerifyBlock")
	return nil
}

func (handler *MockConsensusHandler) ProcessPropose(msg *consensus.ProposeMessage) error {
	handler.log.Info("ProcessPropose")
	return nil
}

func (handler *MockConsensusHandler) ProcessVote(msg *consensus.VoteMessage) error {
	handler.log.Info("ProcessVote")
	return nil
}
