package node

import tplgss "github.com/TopiaNetwork/topia/ledger/state"

type NodeExecutorState interface {
	GetNodeExecutorStateRoot() ([]byte, error)

	GetActiveExecutorIDs() ([]string, error)

	GetActiveExecutorsTotalWeight() (uint64, error)
}

type nodeExecutorState struct {
	tplgss.StateStore
}

func NewNodeExecutorState(stateStore tplgss.StateStore) NodeExecutorState {
	stateStore.AddNamedStateStore("executor")
	return &nodeExecutorState{
		StateStore: stateStore,
	}
}

func (ns *nodeExecutorState) GetNodeExecutorStateRoot() ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (ns *nodeExecutorState) GetActiveExecutorIDs() ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func (ns *nodeExecutorState) GetActiveExecutorsTotalWeight() (u uint64, err error) {
	//TODO implement me
	panic("implement me")
}
