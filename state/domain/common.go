package domain

import (
	"encoding/json"
	"errors"

	tpcmm "github.com/TopiaNetwork/topia/common"
	tplgss "github.com/TopiaNetwork/topia/ledger/state"
)

func isNodeDomainExist(stateStore tplgss.StateStore, name string, id string) bool {
	isExist, _ := stateStore.Exists(name, []byte(id))

	return isExist
}

func getNodeDomain(stateStore tplgss.StateStore, name string, id string) (*tpcmm.NodeDomainInfo, error) {
	domainInfoBytes, err := stateStore.GetStateData(name, []byte(id))
	if err != nil {
		return nil, err
	}

	if domainInfoBytes == nil {
		return nil, nil
	}

	var domainInfo tpcmm.NodeDomainInfo
	err = json.Unmarshal(domainInfoBytes, &domainInfo)
	if err != nil {
		return nil, err
	}

	return &domainInfo, nil
}

func addNodeDomain(stateStore tplgss.StateStore, name string, domainInfo *tpcmm.NodeDomainInfo) error {
	if domainInfo == nil {
		return errors.New("Nil node domain info")
	}

	domainInfoBytes, err := json.Marshal(domainInfo)
	if err != nil {
		return err
	}

	return stateStore.Put(name, []byte(domainInfo.ID), domainInfoBytes)
}
