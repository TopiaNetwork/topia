package node

import tplgss "github.com/TopiaNetwork/topia/ledger/state"

type NodeExecutorState interface {
	GetActiveExecutorIDs() ([]string, error)
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

func (ns *nodeExecutorState) GetActiveExecutorIDs() ([]string, error) {
	//TODO implement me
	panic("implement me")
}
