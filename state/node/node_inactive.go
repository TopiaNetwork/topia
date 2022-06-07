package node

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/TopiaNetwork/topia/common"

	tplgss "github.com/TopiaNetwork/topia/ledger/state"
)

const StateStore_Name_NodeInactive = "nodeinactive"

const (
	TotalInactiveNodeIDs_Key    = "totalinnodeids"
	TotalInactiveNodeWeight_Key = "totalinweight"
)

type NodeInactiveState interface {
	GetNodeInactiveRoot() ([]byte, error)

	IsExistInactiveNode(nodeID string) bool

	GetInactiveNodeIDs() ([]string, error)

	GetInactiveNode(nodeID string) (*common.NodeInfo, error)

	GetInactiveNodesTotalWeight() (uint64, error)

	GetAllInactiveNodes() ([]*common.NodeInfo, error)

	AddInactiveNode(nodeInfo *common.NodeInfo) error

	updateInactiveNodeWeight(nodeID string, weight uint64) error

	updateInactiveNodeDKGPartPubKey(nodeID string, pubKey string) error

	removeInactiveNode(nodeID string) error
}

type nodeInactiveState struct {
	tplgss.StateStore
}

func NewNodeInactiveState(stateStore tplgss.StateStore, cacheSize int) NodeInactiveState {
	stateStore.AddNamedStateStore(StateStore_Name_NodeInactive, cacheSize)
	return &nodeInactiveState{
		StateStore: stateStore,
	}
}

func (ns *nodeInactiveState) GetNodeInactiveRoot() ([]byte, error) {
	return ns.Root(StateStore_Name_NodeInactive)
}

func (ns *nodeInactiveState) IsExistInactiveNode(nodeID string) bool {
	return isNodeExist(ns.StateStore, StateStore_Name_NodeInactive, nodeID)
}

func (ns *nodeInactiveState) GetInactiveNodeIDs() ([]string, error) {
	totolAEIdsBytes, err := ns.GetStateData(StateStore_Name_NodeInactive, []byte(TotalInactiveNodeIDs_Key))
	if err != nil {
		return nil, err
	}

	if totolAEIdsBytes == nil {
		return nil, nil
	}

	var nodeAEIDs []string
	err = json.Unmarshal(totolAEIdsBytes, &nodeAEIDs)
	if err != nil {
		return nil, err
	}

	return nodeAEIDs, nil
}

func (ns *nodeInactiveState) GetInactiveNode(nodeID string) (*common.NodeInfo, error) {
	return getNode(ns.StateStore, StateStore_Name_NodeInactive, nodeID)
}

func (ns *nodeInactiveState) GetInactiveNodesTotalWeight() (uint64, error) {
	totalAEWeightBytes, err := ns.GetStateData(StateStore_Name_NodeInactive, []byte(TotalInactiveNodeWeight_Key))
	if err != nil {
		return 0, err
	}

	if totalAEWeightBytes == nil {
		return 0, nil
	}

	return binary.BigEndian.Uint64(totalAEWeightBytes), nil
}

func (ns *nodeInactiveState) GetAllInactiveNodes() ([]*common.NodeInfo, error) {
	keys, vals, err := ns.GetAllStateData(StateStore_Name_NodeInactive)
	if err != nil {
		return nil, err
	}

	if len(keys) != len(vals) {
		return nil, fmt.Errorf("Invalid keys' len %d and vals' len %d", len(keys), len(vals))
	}

	var nodes []*common.NodeInfo
	for i, val := range vals {
		if string(keys[i]) == TotalInactiveNodeIDs_Key || string(keys[i]) == TotalInactiveNodeWeight_Key {
			continue
		}

		var nodeInfo common.NodeInfo
		err = json.Unmarshal(val, &nodeInfo)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, &nodeInfo)
	}

	return nodes, nil
}

func (ns *nodeInactiveState) AddInactiveNode(nodeInfo *common.NodeInfo) error {
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
