package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TopiaNetwork/topia/chain"
	tplgss "github.com/TopiaNetwork/topia/ledger/state"
)

const StateStore_Name_Node = "node"

type NodeState interface {
	GetNodeStateRoot() ([]byte, error)

	IsNodeExist(nodeID string) bool

	GetAllConsensusNodeIDs() ([]string, error)

	GetNode(nodeID string) (*chain.NodeInfo, error)

	GetTotalWeight() (uint64, error)

	GetNodeWeight(nodeID string) (uint64, error)

	AddNode(nodeInfo *chain.NodeInfo) error

	UpdateWeight(nodeID string, weight uint64) error
}

type nodeStateMetaInfo struct {
	Role  chain.NodeRole
	State chain.NodeState
}

type nodeState struct {
	tplgss.StateStore
	NodeExecutorState
	NodeProposerState
	NodeValidatorState
	NodeInactiveState
}

func NewNodeState(stateStore tplgss.StateStore, inactiveState NodeInactiveState, executorState NodeExecutorState, proposerState NodeProposerState, validatorState NodeValidatorState) NodeState {
	stateStore.AddNamedStateStore(StateStore_Name_Node)
	return &nodeState{
		StateStore:         stateStore,
		NodeInactiveState:  inactiveState,
		NodeExecutorState:  executorState,
		NodeProposerState:  proposerState,
		NodeValidatorState: validatorState,
	}
}

func (ns *nodeState) GetNodeStateRoot() ([]byte, error) {
	return ns.Root(StateStore_Name_Node)
}

func (ns *nodeState) IsNodeExist(nodeID string) bool {
	return isNodeExist(ns.StateStore, StateStore_Name_Node, nodeID)
}

func (ns *nodeState) GetAllConsensusNodeIDs() ([]string, error) {
	var allNodeIDs []string

	allExecutorIDs, err := ns.GetActiveExecutorIDs()
	if err != nil {
		return nil, err
	}
	allNodeIDs = append(allNodeIDs, allExecutorIDs...)

	allProposerIDs, err := ns.GetActiveProposerIDs()
	if err != nil {
		return nil, err
	}
	allNodeIDs = append(allNodeIDs, allProposerIDs...)

	allValidatorIDs, err := ns.GetActiveValidatorIDs()
	if err != nil {
		return nil, err
	}
	allNodeIDs = append(allNodeIDs, allValidatorIDs...)

	allInactiveIDs, err := ns.GetInactiveNodeIDs()
	if err != nil {
		return nil, err
	}
	allNodeIDs = append(allNodeIDs, allInactiveIDs...)

	return allNodeIDs, nil
}

func (ns *nodeState) GetNode(nodeID string) (*chain.NodeInfo, error) {
	nodeMetaInfoBytes, _, err := ns.GetState(StateStore_Name_Node, []byte(nodeID))
	if err != nil {
		return nil, err
	}

	if nodeMetaInfoBytes == nil { //means that there is no node info
		return nil, nil
	}

	var nodeMetaInfo nodeStateMetaInfo
	err = json.Unmarshal(nodeMetaInfoBytes, &nodeMetaInfo)
	if err != nil {
		return nil, err
	}

	switch nodeMetaInfo.State {
	case chain.NodeState_Active:
		{
			switch nodeMetaInfo.Role {
			case chain.NodeRole_Executor:
				return ns.GetActiveExecutor(nodeID)
			case chain.NodeRole_Proposer:
				return ns.GetActiveProposer(nodeID)
			case chain.NodeRole_Validator:
				return ns.GetActiveValidator(nodeID)
			default:
				return nil, fmt.Errorf("Invalid node role from %s", nodeID)
			}
		}
	case chain.NodeState_Inactive:
		return ns.GetInactiveNode(nodeID)
	default:
		return nil, fmt.Errorf("Invalid node state from %s", nodeID)
	}
}

func (ns *nodeState) GetTotalWeight() (uint64, error) {
	activeExecutorsWeight, err := ns.GetActiveExecutorsTotalWeight()
	if err != nil {
		return 0, err
	}

	activeProposerWeight, err := ns.GetActiveProposersTotalWeight()
	if err != nil {
		return 0, err
	}

	activeValidatirWeight, err := ns.GetActiveValidatorsTotalWeight()
	if err != nil {
		return 0, err
	}

	return activeExecutorsWeight + activeProposerWeight + activeValidatirWeight, nil
}

func (ns *nodeState) GetNodeWeight(nodeID string) (uint64, error) {
	nodeInfo, err := ns.GetNode(nodeID)
	if err != nil {
		return 0, err
	}
	if nodeInfo == nil {
		return 0, nil
	}

	return nodeInfo.Weight, nil
}

func (ns *nodeState) AddNode(nodeInfo *chain.NodeInfo) error {
	if nodeInfo == nil {
		return errors.New("Nil node info input")
	}

	var err error
	switch nodeInfo.State {
	case chain.NodeState_Active:
		{
			switch nodeInfo.Role {
			case chain.NodeRole_Executor:
				err = ns.addActiveExecutor(nodeInfo)
			case chain.NodeRole_Proposer:
				err = ns.AddActiveProposer(nodeInfo)
			case chain.NodeRole_Validator:
				err = ns.AddActiveValidator(nodeInfo)
			default:
				return fmt.Errorf("Invalid node role from %s", nodeInfo.NodeID)
			}

			break
		}
	case chain.NodeState_Inactive:
		err = ns.AddInactiveNode(nodeInfo)
		break
	default:
		return fmt.Errorf("Invalid node state from %s", nodeInfo.NodeID)
	}

	if err != nil {
		return err
	}

	nodeMetaInfo := nodeStateMetaInfo{
		Role:  nodeInfo.Role,
		State: nodeInfo.State,
	}
	nodeMetaInfoBytes, err := json.Marshal(&nodeMetaInfo)
	if err != nil {
		return err
	}

	return ns.Put(StateStore_Name_Node, []byte(nodeInfo.NodeID), nodeMetaInfoBytes)
}

func (ns *nodeState) UpdateWeight(nodeID string, weight uint64) error {
	nodeMetaInfoBytes, _, err := ns.GetState(StateStore_Name_Node, []byte(nodeID))
	if err != nil {
		return err
	}

	if nodeMetaInfoBytes == nil { //means that there is no node info
		return nil
	}

	var nodeMetaInfo nodeStateMetaInfo
	err = json.Unmarshal(nodeMetaInfoBytes, &nodeMetaInfo)
	if err != nil {
		return err
	}

	switch nodeMetaInfo.State {
	case chain.NodeState_Active:
		{
			switch nodeMetaInfo.Role {
			case chain.NodeRole_Executor:
				return ns.updateActiveExecutorWeight(nodeID, weight)
			case chain.NodeRole_Proposer:
				return ns.UpdateActiveProposerWeight(nodeID, weight)
			case chain.NodeRole_Validator:
				return ns.UpdateActiveValidatorWeight(nodeID, weight)
			default:
				return fmt.Errorf("Invalid node role from %s", nodeID)
			}
		}
	case chain.NodeState_Inactive:
		return ns.UpdateInactiveNodeWeight(nodeID, weight)
	default:
		return fmt.Errorf("Invalid node state from %s", nodeID)
	}
}
