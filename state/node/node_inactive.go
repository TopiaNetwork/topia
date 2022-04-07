package node

import (
	"encoding/binary"
	"encoding/json"

	"github.com/TopiaNetwork/topia/chain"
	tplgss "github.com/TopiaNetwork/topia/ledger/state"
)

const StateStore_Name_NodeInactive = "nodeinactive"

const (
	TotalInactiveNodeIDs_Key    = "totalinnodeids"
	TotalInactiveNodeWeight_Key = "totalinweight"
)

type NodeInactiveState interface {
	GetNodeInactiveStateRoot() ([]byte, error)

	IsExistInactiveNode(nodeID string) bool

	GetInactiveNodeIDs() ([]string, error)

	GetInactiveNode(nodeID string) (*chain.NodeInfo, error)

	GetInactiveNodesTotalWeight() (uint64, error)

	AddInactiveNode(nodeInfo *chain.NodeInfo) error

	updateInactiveNodeWeight(nodeID string, weight uint64) error

	updateInactiveNodeDKGPartPubKey(nodeID string, pubKey string) error

	removeInactiveNode(nodeID string) error
}

type nodeInactiveState struct {
	tplgss.StateStore
}

func NewNodeInactiveState(stateStore tplgss.StateStore) NodeInactiveState {
	stateStore.AddNamedStateStore(StateStore_Name_NodeInactive)
	return &nodeInactiveState{
		StateStore: stateStore,
	}
}

func (ns *nodeInactiveState) GetNodeInactiveStateRoot() ([]byte, error) {
	return ns.Root(StateStore_Name_NodeInactive)
}

func (ns *nodeInactiveState) IsExistInactiveNode(nodeID string) bool {
	return isNodeExist(ns.StateStore, StateStore_Name_NodeInactive, nodeID)
}

func (ns *nodeInactiveState) GetInactiveNodeIDs() ([]string, error) {
	totolAEIdsBytes, _, err := ns.GetState(StateStore_Name_NodeInactive, []byte(TotalInactiveNodeIDs_Key))
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

func (ns *nodeInactiveState) GetInactiveNode(nodeID string) (*chain.NodeInfo, error) {
	return getNode(ns.StateStore, StateStore_Name_NodeInactive, nodeID)
}

func (ns *nodeInactiveState) GetInactiveNodesTotalWeight() (uint64, error) {
	totalAEWeightBytes, _, err := ns.GetState(StateStore_Name_NodeInactive, []byte(TotalInactiveNodeWeight_Key))
	if err != nil {
		return 0, err
	}

	if totalAEWeightBytes == nil {
		return 0, nil
	}

	return binary.BigEndian.Uint64(totalAEWeightBytes), nil
}

func (ns *nodeInactiveState) AddInactiveNode(nodeInfo *chain.NodeInfo) error {
	return addNode(ns.StateStore, StateStore_Name_NodeInactive, TotalInactiveNodeIDs_Key, TotalInactiveNodeWeight_Key, nodeInfo)
}

func (ns *nodeInactiveState) updateInactiveNodeWeight(nodeID string, weight uint64) error {
	return uppdateWeight(ns.StateStore, StateStore_Name_NodeInactive, nodeID, TotalInactiveNodeWeight_Key, weight)
}

func (ns *nodeInactiveState) updateInactiveNodeDKGPartPubKey(nodeID string, pubKey string) error {
	return uppdateDKGPartPubKey(ns.StateStore, StateStore_Name_NodeInactive, nodeID, pubKey)
}

func (ns *nodeInactiveState) removeInactiveNode(nodeID string) error {
	nodeInfo, err := ns.GetInactiveNode(nodeID)
	if err != nil {
		return nil
	}

	if nodeInfo == nil {
		return nil
	}

	return removeNode(ns.StateStore, StateStore_Name_NodeInactive, TotalInactiveNodeIDs_Key, TotalInactiveNodeWeight_Key, nodeID, nodeInfo.Weight)
}
