package consensus

import (
	"github.com/TopiaNetwork/topia/chain"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
)

type consensusServantMock struct{}

func (cs *consensusServantMock) ChainID() chain.ChainID {
	return "TestNet"
}

func (cs *consensusServantMock) GetLatestBlock() (*tpchaintypes.Block, error) {
	//TODO implement me
	panic("implement me")
}

func (cs *consensusServantMock) GetAllConsensusNodes() ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func (cs *consensusServantMock) GetActiveExecutorIDs() ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func (cs *consensusServantMock) GetActiveProposerIDs() ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func (cs *consensusServantMock) GetActiveValidatorIDs() ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func (cs *consensusServantMock) GetChainTotalWeight() (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (cs *consensusServantMock) GetActiveExecutorsTotalWeight() (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (cs *consensusServantMock) GetActiveProposersTotalWeight() (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (cs *consensusServantMock) GetActiveValidatorsTotalWeight() (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (cs *consensusServantMock) GetNodeWeight(nodeID string) (uint64, error) {
	//TODO implement me
	panic("implement me")
}
