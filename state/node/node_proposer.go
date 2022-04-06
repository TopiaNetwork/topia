package node

import (
	"encoding/binary"
	"encoding/json"

	"github.com/TopiaNetwork/topia/chain"
	tplgss "github.com/TopiaNetwork/topia/ledger/state"
)

const StateStore_Name_Prop = "proposer"

const (
	TotalActiveProposerNodeIDs_Key = "totalapnodeids"
	TotalActiveProposerWeight_Key  = "totalapweight"
)

type NodeProposerState interface {
	GetNodeProposerStateRoot() ([]byte, error)

	IsExistActiveProposer(nodeID string) bool

	GetActiveProposerIDs() ([]string, error)

	GetActiveProposer(nodeID string) (*chain.NodeInfo, error)

	GetActiveProposersTotalWeight() (uint64, error)

	AddActiveProposer(nodeInfo *chain.NodeInfo) error

	UpdateActiveProposerWeight(nodeID string, weight uint64) error

	RemoveActiveProposer(nodeID string) error
}

type nodeProposerState struct {
	tplgss.StateStore
}

func NewNodeProposerState(stateStore tplgss.StateStore) NodeProposerState {
	stateStore.AddNamedStateStore(StateStore_Name_Prop)
	return &nodeProposerState{
		StateStore: stateStore,
	}
}

func (ns *nodeProposerState) GetNodeProposerStateRoot() ([]byte, error) {
	return ns.Root(StateStore_Name_Prop)
}

func (ns *nodeProposerState) IsExistActiveProposer(nodeID string) bool {
	return isNodeExist(ns.StateStore, StateStore_Name_Prop, nodeID)
}

func (ns *nodeProposerState) GetActiveProposerIDs() ([]string, error) {
	totolAEIdsBytes, _, err := ns.GetState(StateStore_Name_Prop, []byte(TotalActiveProposerNodeIDs_Key))
	if err != nil {
		return nil, err
	}

	var nodeAEIDs []string
	err = json.Unmarshal(totolAEIdsBytes, &nodeAEIDs)
	if err != nil {
		return nil, err
	}

	return nodeAEIDs, nil
}

func (ns *nodeProposerState) GetActiveProposer(nodeID string) (*chain.NodeInfo, error) {
	return getNode(ns.StateStore, StateStore_Name_Prop, nodeID)
}

func (ns *nodeProposerState) GetActiveProposersTotalWeight() (uint64, error) {
	totalAEWeightBytes, _, err := ns.GetState(StateStore_Name_Prop, []byte(TotalActiveProposerWeight_Key))
	if err != nil {
		return 0, err
	}

	if totalAEWeightBytes == nil {
		return 0, nil
	}

	return binary.BigEndian.Uint64(totalAEWeightBytes), nil
}

func (ns *nodeProposerState) AddActiveProposer(nodeInfo *chain.NodeInfo) error {
	return addNode(ns.StateStore, StateStore_Name_Prop, TotalActiveProposerNodeIDs_Key, TotalActiveProposerWeight_Key, nodeInfo)
}

func (ns *nodeProposerState) UpdateActiveProposerWeight(nodeID string, weight uint64) error {
	return uppdateWeight(ns.StateStore, StateStore_Name_Prop, nodeID, TotalActiveProposerWeight_Key, weight)
}

func (ns *nodeProposerState) RemoveActiveProposer(nodeID string) error {
	nodeInfo, err := ns.GetActiveProposer(nodeID)
	if err != nil {
		return nil
	}

	if nodeInfo == nil {
		return nil
	}

	return removeNode(ns.StateStore, StateStore_Name_Prop, TotalActiveProposerNodeIDs_Key, TotalActiveProposerWeight_Key, nodeID, nodeInfo.Weight)
}
