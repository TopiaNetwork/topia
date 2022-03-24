package chain

import (
	"github.com/TopiaNetwork/topia/chain"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tplgss "github.com/TopiaNetwork/topia/ledger/state"
	tpnet "github.com/TopiaNetwork/topia/network"
)

type ChainState interface {
	ChainID() chain.ChainID

	NetworkType() tpnet.NetworkType

	GetChainRoot() ([]byte, error)

	GetAllConsensusNodes() ([]string, error)

	GetChainTotalWeight() (uint64, error)

	GetNodeWeight(nodeID string) (uint64, error)

	GetLatestBlock() (*tpchaintypes.Block, error)
}

type chainState struct {
	tplgss.StateStore
}

func NewChainStore(stateStore tplgss.StateStore) ChainState {
	stateStore.AddNamedStateStore("chain")
	return &chainState{
		StateStore: stateStore,
	}
}

func (cs *chainState) ChainID() chain.ChainID {
	return "TestNet"
}

func (cs *chainState) NetworkType() tpnet.NetworkType {
	//TODO implement me
	panic("implement me")
}

func (cs *chainState) GetChainRoot() ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (cs *chainState) GetAllConsensusNodes() ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func (cs *chainState) GetChainTotalWeight() (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (cs *chainState) GetNodeWeight(nodeID string) (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (cs *chainState) GetLatestBlock() (*tpchaintypes.Block, error) {
	//TODO implement me
	panic("implement me")
}
