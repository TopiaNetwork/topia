package domain

import (
	"encoding/json"
	"errors"
	"fmt"

	tpcmm "github.com/TopiaNetwork/topia/common"
	tplgss "github.com/TopiaNetwork/topia/ledger/state"
)

const StateStore_Name_NodeDomain = "nodedomain"

type NodeDomainState interface {
	GetNodeDomainRoot() ([]byte, error)

	IsNodeDomainExist(nodeID string) bool

	GetNodeDomain(id string) (*tpcmm.NodeDomainInfo, error)

	GetAllActiveNodeDomains(height uint64) ([]*tpcmm.NodeDomainInfo, error)

	AddNodeDomain(domain *tpcmm.NodeDomainInfo) error
}

type nodeDomainState struct {
	tplgss.StateStore
	NodeExecuteDomainState
	NodeConsensusDomainState
}

type nodeDomainStateMetaInfo struct {
	DomainType tpcmm.DomainType
}

func NewNodeDomainState(stateStore tplgss.StateStore, exeDomainState NodeExecuteDomainState, csDomainState NodeConsensusDomainState, cacheSize int) NodeDomainState {
	stateStore.AddNamedStateStore(StateStore_Name_NodeDomain, cacheSize)
	return &nodeDomainState{
		StateStore:               stateStore,
		NodeExecuteDomainState:   exeDomainState,
		NodeConsensusDomainState: csDomainState,
	}
}

func (ns *nodeDomainState) GetNodeDomainRoot() ([]byte, error) {
	return ns.Root(StateStore_Name_NodeDomain)
}

func (ns *nodeDomainState) IsNodeDomainExist(id string) bool {
	return isNodeDomainExist(ns.StateStore, StateStore_Name_NodeDomain, id)
}

func (ns *nodeDomainState) GetNodeDomain(id string) (*tpcmm.NodeDomainInfo, error) {
	domainMetaInfoBytes, err := ns.GetStateData(StateStore_Name_NodeDomain, []byte(id))
	if err != nil {
		return nil, err
	}

	if domainMetaInfoBytes == nil { //means that there is no node domain info
		return nil, nil
	}

	var domainMetaInfo nodeDomainStateMetaInfo
	err = json.Unmarshal(domainMetaInfoBytes, &domainMetaInfo)
	if err != nil {
		return nil, err
	}

	switch domainMetaInfo.DomainType {
	case tpcmm.DomainType_Execute:
		return ns.getNodeExecuteDomain(id)
	case tpcmm.DomainType_Consensus:
		return ns.getNodeConsensusDomain(id)
	default:
		return nil, fmt.Errorf("Invalid domain type from %s", id)
	}
}

func (ns *nodeDomainState) GetAllActiveNodeDomains(height uint64) ([]*tpcmm.NodeDomainInfo, error) {
	var activeNodeDomains []*tpcmm.NodeDomainInfo

	exeActiveNodeDomains, err := ns.GetAllActiveNodeExecuteDomains(height)
	if err != nil {
		return nil, err
	}
	activeNodeDomains = append(activeNodeDomains, exeActiveNodeDomains...)

	csActiveNodeDomains, err := ns.GetAllActiveNodeConsensusDomains(height)
	if err != nil {
		return nil, err
	}
	activeNodeDomains = append(activeNodeDomains, csActiveNodeDomains...)

	return activeNodeDomains, nil
}

func (ns *nodeDomainState) AddNodeDomain(domainInfo *tpcmm.NodeDomainInfo) error {
	if domainInfo == nil {
		return errors.New("Nil domain info input")
	}

	var err error
	switch domainInfo.Type {
	case tpcmm.DomainType_Execute:
		err = ns.addNodeExecuteDomain(domainInfo)
	case tpcmm.DomainType_Consensus:
		err = ns.addNodeConsensusDomain(domainInfo)
	default:
		return fmt.Errorf("Invalid domain type from %s", domainInfo.ID)
	}
	if err != nil {
		return err
	}

	domainMetaInfo := nodeDomainStateMetaInfo{
		DomainType: domainInfo.Type,
	}
	domainMetaInfoBytes, err := json.Marshal(&domainMetaInfo)
	if err != nil {
		return err
	}

	return ns.Put(StateStore_Name_NodeDomain, []byte(domainInfo.ID), domainMetaInfoBytes)
}
