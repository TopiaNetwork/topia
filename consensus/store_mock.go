package consensus

import (
	tpcmm "github.com/TopiaNetwork/topia/common"
	tptypes "github.com/TopiaNetwork/topia/common/types"
)

type consensusStoreMock struct{}

func (cs *consensusStoreMock) ChainID() tpcmm.ChainID {
	return "TestNet"
}

func (cs *consensusStoreMock) GetLatestBlock() (*tptypes.Block, error) {
	//TODO implement me
	panic("implement me")
}

func (cs *consensusStoreMock) SaveBlockMiddleResult(round uint64, blockResult *tptypes.BlockResultStoreInfo) error {
	//TODO implement me
	panic("implement me")
}

func (cs *consensusStoreMock) Commit() error {
	//TODO implement me
	panic("implement me")
}

func (cs *consensusStoreMock) ClearBlockMiddleResult(round uint64) error {
	//TODO implement me
	panic("implement me")
}

func (cs *consensusStoreMock) GetAllConsensusNodes() ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func (cs *consensusStoreMock) GetChainTotalWeight() (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (cs *consensusStoreMock) GetNodeWeight(nodeID string) (uint64, error) {
	//TODO implement me
	panic("implement me")
}
