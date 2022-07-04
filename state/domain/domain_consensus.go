package domain

import (
	"encoding/json"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tplgss "github.com/TopiaNetwork/topia/ledger/state"
)

const StateStore_Name_NodeConsensusDomain = "nodeconsensusdomain"

type NodeConsensusDomainState interface {
	GetNodeConsensusDomainRoot() ([]byte, error)

	IsExistNodeConsensusDomain(id string) bool

	getNodeConsensusDomain(id string) (*tpcmm.NodeDomainInfo, error)

	GetAllActiveNodeConsensusDomains(height uint64) ([]*tpcmm.NodeDomainInfo, error)

	addNodeConsensusDomain(domainInfo *tpcmm.NodeDomainInfo) error
}

type nodeConsensusDomainState struct {
	tplgss.StateStore
}

func NewNodeConsensusDomainState(stateStore tplgss.StateStore, cacheSize int) NodeConsensusDomainState {
	stateStore.AddNamedStateStore(StateStore_Name_NodeConsensusDomain, cacheSize)
	return &nodeConsensusDomainState{
		StateStore: stateStore,
	}
}

func (ns *nodeConsensusDomainState) GetNodeConsensusDomainRoot() ([]byte, error) {
	return ns.Root(StateStore_Name_NodeConsensusDomain)
}

func (ns *nodeConsensusDomainState) IsExistNodeConsensusDomain(id string) bool {
	return isNodeDomainExist(ns.StateStore, StateStore_Name_NodeConsensusDomain, id)
}

func (ns *nodeConsensusDomainState) getNodeConsensusDomain(id string) (*tpcmm.NodeDomainInfo, error) {
	return getNodeDomain(ns.StateStore, StateStore_Name_NodeConsensusDomain, id)
}

func (ns *nodeConsensusDomainState) GetAllActiveNodeConsensusDomains(height uint64) ([]*tpcmm.NodeDomainInfo, error) {
	var domains []*tpcmm.NodeDomainInfo

	err := ns.IterateAllStateDataCB(StateStore_Name_NodeConsensusDomain, func(key []byte, val []byte) {
		var domainInfo tpcmm.NodeDomainInfo
		err := json.Unmarshal(val, &domainInfo)
		if err != nil {
			return
		}
		if domainInfo.ValidHeightStart > height && height < domainInfo.ValidHeightEnd && (domainInfo.ValidHeightEnd-tpcmm.EpochSpan) >= 0 {
			domains = append(domains, &domainInfo)
		}
	})
	if err != nil {
		return nil, err
	}

	return domains, nil
}

func (ns *nodeConsensusDomainState) addNodeConsensusDomain(domainInfo *tpcmm.NodeDomainInfo) error {
	return addNodeDomain(ns.StateStore, StateStore_Name_NodeConsensusDomain, domainInfo)
}
