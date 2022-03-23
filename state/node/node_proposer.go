package node

import tplgss "github.com/TopiaNetwork/topia/ledger/state"

type NodeProposerState interface {
	GetActiveProposerIDs() ([]string, error)

	GetActiveProposersTotalWeight() (uint64, error)
}

type nodeProposerState struct {
	tplgss.StateStore
}

func NewNodeProposerState(stateStore tplgss.StateStore) NodeProposerState {
	stateStore.AddNamedStateStore("proposer")
	return &nodeProposerState{
		StateStore: stateStore,
	}
}

func (ns *nodeProposerState) GetActiveProposerIDs() ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func (ns *nodeProposerState) GetActiveProposersTotalWeight() (u uint64, err error) {
	//TODO implement me
	panic("implement me")
}
