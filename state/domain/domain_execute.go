package domain

import (
	"encoding/json"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tplgss "github.com/TopiaNetwork/topia/ledger/state"
)

const StateStore_Name_NodeExecuteDomain = "nodeexecutedomain"

type NodeExecuteDomainState interface {
	GetNodeExecuteDomainRoot() ([]byte, error)

	IsExistNodeExecuteDomain(id string) bool

	getNodeExecuteDomain(id string) (*tpcmm.NodeDomainInfo, error)

	GetAllActiveNodeExecuteDomains(height uint64) ([]*tpcmm.NodeDomainInfo, error)

	addNodeExecuteDomain(domainInfo *tpcmm.NodeDomainInfo) error
}

type nodeExecuteDomainState struct {
	tplgss.StateStore
}

func NewNodeExecuteDomainState(stateStore tplgss.StateStore, cacheSize int) NodeExecuteDomainState {
	stateStore.AddNamedStateStore(StateStore_Name_NodeExecuteDomain, cacheSize)
	return &nodeExecuteDomainState{
		StateStore: stateStore,
	}
}

func (ns *nodeExecuteDomainState) GetNodeExecuteDomainRoot() ([]byte, error) {
	return ns.Root(StateStore_Name_NodeExecuteDomain)
}

func (ns *nodeExecuteDomainState) IsExistNodeExecuteDomain(id string) bool {
	return isNodeDomainExist(ns.StateStore, StateStore_Name_NodeExecuteDomain, id)
}

func (ns *nodeExecuteDomainState) getNodeExecuteDomain(id string) (*tpcmm.NodeDomainInfo, error) {
	return getNodeDomain(ns.StateStore, StateStore_Name_NodeExecuteDomain, id)
}

func (ns *nodeExecuteDomainState) GetAllActiveNodeExecuteDomains(height uint64) ([]*tpcmm.NodeDomainInfo, error) {
	var domains []*tpcmm.NodeDomainInfo

	err := ns.IterateAllStateDataCB(StateStore_Name_NodeExecuteDomain, func(key []byte, val []byte) {
		var domainInfo tpcmm.NodeDomainInfo
		err := json.Unmarshal(val, &domainInfo)
		if err != nil {
			return
		}
		if height >= domainInfo.ValidHeightStart && height < domainInfo.ValidHeightEnd && (domainInfo.ValidHeightEnd-tpcmm.EpochSpan) >= 0 {
			domains = append(domains, &domainInfo)
		}
	})
	if err != nil {
		return nil, err
	}

	return domains, nil
}

func (ns *nodeExecuteDomainState) addNodeExecuteDomain(domainInfo *tpcmm.NodeDomainInfo) error {
	return addNodeDomain(ns.StateStore, StateStore_Name_NodeExecuteDomain, domainInfo)
}
