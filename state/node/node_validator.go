package node

import tplgss "github.com/TopiaNetwork/topia/ledger/state"

type NodeValidatorState interface {
	GetActiveValidatorIDs() ([]string, error)

	GetActiveValidatorsTotalWeight(uint64, error)
}

type nodeValidatorState struct {
	tplgss.StateStore
}

func NewNodeValidatorState(stateStore tplgss.StateStore) NodeValidatorState {
	stateStore.AddNamedStateStore("validator")
	return &nodeValidatorState{
		StateStore: stateStore,
	}
}

func (ns *nodeValidatorState) GetActiveValidatorIDs() ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func (ns *nodeValidatorState) GetActiveValidatorsTotalWeight(u uint64, err error) {
	//TODO implement me
	panic("implement me")
}
