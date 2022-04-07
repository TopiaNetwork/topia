package node

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/TopiaNetwork/topia/chain"
	tplgss "github.com/TopiaNetwork/topia/ledger/state"
)

const StateStore_Name_Val = "validator"

const (
	TotalActiveValidatorNodeIDs_Key = "totalavnodeids"
	TotalActiveValidatorWeight_Key  = "totalavweight"
)

type NodeValidatorState interface {
	GetNodeValidatorStateRoot() ([]byte, error)

	IsExistActiveValidator(nodeID string) bool

	GetActiveValidatorIDs() ([]string, error)

	GetActiveValidator(nodeID string) (*chain.NodeInfo, error)

	GetActiveValidatorsTotalWeight() (uint64, error)

	GetAllActiveValidators() ([]*chain.NodeInfo, error)

	AddActiveValidator(nodeInfo *chain.NodeInfo) error

	updateActiveValidatorWeight(nodeID string, weight uint64) error

	updateActiveValidatorDKGPartPubKey(nodeID string, pubKey string) error

	removeActiveValidator(nodeID string) error
}

type nodeValidatorState struct {
	tplgss.StateStore
}

func NewNodeValidatorState(stateStore tplgss.StateStore) NodeValidatorState {
	stateStore.AddNamedStateStore(StateStore_Name_Val)
	return &nodeValidatorState{
		StateStore: stateStore,
	}
}

func (ns *nodeValidatorState) GetNodeValidatorStateRoot() ([]byte, error) {
	return ns.Root(StateStore_Name_Val)
}

func (ns *nodeValidatorState) IsExistActiveValidator(nodeID string) bool {
	return isNodeExist(ns.StateStore, StateStore_Name_Val, nodeID)
}

func (ns *nodeValidatorState) GetActiveValidatorIDs() ([]string, error) {
	totolAEIdsBytes, _, err := ns.GetState(StateStore_Name_Val, []byte(TotalActiveValidatorNodeIDs_Key))
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

func (ns *nodeValidatorState) GetActiveValidator(nodeID string) (*chain.NodeInfo, error) {
	return getNode(ns.StateStore, StateStore_Name_Val, nodeID)
}

func (ns *nodeValidatorState) GetActiveValidatorsTotalWeight() (uint64, error) {
	totalAEWeightBytes, _, err := ns.GetState(StateStore_Name_Val, []byte(TotalActiveValidatorWeight_Key))
	if err != nil {
		return 0, err
	}

	if totalAEWeightBytes == nil {
		return 0, nil
	}

	return binary.BigEndian.Uint64(totalAEWeightBytes), nil
}

func (ns *nodeValidatorState) GetAllActiveValidators() ([]*chain.NodeInfo, error) {
	keys, vals, _, err := ns.GetAllState(StateStore_Name_Val)
	if err != nil {
		return nil, err
	}

	if len(keys) != len(vals) {
		return nil, fmt.Errorf("Invalid keys' len %d and vals' len %d", len(keys), len(vals))
	}

	var nodes []*chain.NodeInfo
	for i, val := range vals {
		if string(keys[i]) == TotalActiveValidatorNodeIDs_Key || string(keys[i]) == TotalActiveValidatorWeight_Key {
			continue
		}

		var nodeInfo chain.NodeInfo
		err = json.Unmarshal(val, &nodeInfo)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, &nodeInfo)
	}

	return nodes, nil
}

func (ns *nodeValidatorState) AddActiveValidator(nodeInfo *chain.NodeInfo) error {
	return addNode(ns.StateStore, StateStore_Name_Val, TotalActiveValidatorNodeIDs_Key, TotalActiveValidatorWeight_Key, nodeInfo)
}

func (ns *nodeValidatorState) updateActiveValidatorWeight(nodeID string, weight uint64) error {
	return uppdateWeight(ns.StateStore, StateStore_Name_Val, nodeID, TotalActiveValidatorWeight_Key, weight)
}

func (ns *nodeValidatorState) updateActiveValidatorDKGPartPubKey(nodeID string, pubKey string) error {
	return uppdateDKGPartPubKey(ns.StateStore, StateStore_Name_Val, nodeID, pubKey)
}

func (ns *nodeValidatorState) removeActiveValidator(nodeID string) error {
	nodeInfo, err := ns.GetActiveValidator(nodeID)
	if err != nil {
		return nil
	}

	if nodeInfo == nil {
		return nil
	}

	return removeNode(ns.StateStore, StateStore_Name_Val, TotalActiveValidatorNodeIDs_Key, TotalActiveValidatorWeight_Key, nodeID, nodeInfo.Weight)
}
