package node

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/TopiaNetwork/topia/common"

	tplgss "github.com/TopiaNetwork/topia/ledger/state"
)

const StateStore_Name_Exe = "executor"

const (
	TotalActiveExecutorNodeIDs_Key = "totalaenodeids"
	TotalActiveExecutorWeight_Key  = "totalaeweight"
)

type NodeExecutorState interface {
	GetNodeExecutorStateRoot() ([]byte, error)

	IsExistActiveExecutor(nodeID string) bool

	GetActiveExecutorIDs() ([]string, error)

	GetActiveExecutor(nodeID string) (*common.NodeInfo, error)

	GetActiveExecutorsTotalWeight() (uint64, error)

	GetAllActiveExecutors() ([]*common.NodeInfo, error)

	addActiveExecutor(nodeInfo *common.NodeInfo) error

	updateActiveExecutorWeight(nodeID string, weight uint64) error

	updateActiveExecutorDKGPartPubKey(nodeID string, pubKey string) error

	removeActiveExecutor(nodeID string) error
}

type nodeExecutorState struct {
	tplgss.StateStore
}

func NewNodeExecutorState(stateStore tplgss.StateStore) NodeExecutorState {
	stateStore.AddNamedStateStore(StateStore_Name_Exe)
	return &nodeExecutorState{
		StateStore: stateStore,
	}
}

func (ns *nodeExecutorState) GetNodeExecutorStateRoot() ([]byte, error) {
	return ns.Root(StateStore_Name_Exe)
}

func (ns *nodeExecutorState) IsExistActiveExecutor(nodeID string) bool {
	return isNodeExist(ns.StateStore, StateStore_Name_Exe, nodeID)
}

func (ns *nodeExecutorState) GetActiveExecutorIDs() ([]string, error) {
	totolAEIdsBytes, _, err := ns.GetState(StateStore_Name_Exe, []byte(TotalActiveExecutorNodeIDs_Key))
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

func (ns *nodeExecutorState) GetActiveExecutor(nodeID string) (*common.NodeInfo, error) {
	return getNode(ns.StateStore, StateStore_Name_Exe, nodeID)
}

func (ns *nodeExecutorState) GetActiveExecutorsTotalWeight() (uint64, error) {
	totalAEWeightBytes, _, err := ns.GetState(StateStore_Name_Exe, []byte(TotalActiveExecutorWeight_Key))
	if err != nil {
		return 0, err
	}

	if totalAEWeightBytes == nil {
		return 0, nil
	}

	return binary.BigEndian.Uint64(totalAEWeightBytes), nil
}

func (ns *nodeExecutorState) GetAllActiveExecutors() ([]*common.NodeInfo, error) {
	keys, vals, _, err := ns.GetAllState(StateStore_Name_Exe)
	if err != nil {
		return nil, err
	}

	if len(keys) != len(vals) {
		return nil, fmt.Errorf("Invalid keys' len %d and vals' len %d", len(keys), len(vals))
	}

	var nodes []*common.NodeInfo
	for i, val := range vals {
		if string(keys[i]) == TotalActiveExecutorNodeIDs_Key || string(keys[i]) == TotalActiveExecutorWeight_Key {
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

func (ns *nodeExecutorState) addActiveExecutor(nodeInfo *common.NodeInfo) error {
	return addNode(ns.StateStore, StateStore_Name_Exe, TotalActiveExecutorNodeIDs_Key, TotalActiveExecutorWeight_Key, nodeInfo)
}

func (ns *nodeExecutorState) updateActiveExecutorWeight(nodeID string, weight uint64) error {
	return uppdateWeight(ns.StateStore, StateStore_Name_Exe, nodeID, TotalActiveExecutorWeight_Key, weight)
}

func (ns *nodeExecutorState) updateActiveExecutorDKGPartPubKey(nodeID string, pubKey string) error {
	return uppdateDKGPartPubKey(ns.StateStore, StateStore_Name_Exe, nodeID, pubKey)
}

func (ns *nodeExecutorState) removeActiveExecutor(nodeID string) error {
	nodeInfo, err := ns.GetActiveExecutor(nodeID)
	if err != nil {
		return nil
	}

	if nodeInfo == nil {
		return nil
	}

	return removeNode(ns.StateStore, StateStore_Name_Exe, TotalActiveExecutorNodeIDs_Key, TotalActiveExecutorWeight_Key, nodeID, nodeInfo.Weight)
}
