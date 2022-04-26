package account

import (
	"reflect"
	"sync"

	tpcmm "github.com/TopiaNetwork/topia/common"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
)

type PermissionContractMethod struct {
	GasLimit   uint64
	sync       sync.RWMutex
	methodsMap map[string][]string //Contract Address->Methods
}

func NewPermissionContractMethod(gasLimit uint64) Permission {
	return &PermissionContractMethod{
		methodsMap: make(map[string][]string),
	}
}

func (pr *PermissionContractMethod) IsRoot() bool {
	return false
}

func (pm *PermissionContractMethod) AddMethod(contractAddr tpcrtypes.Address, method string) {
	pm.sync.Lock()
	defer pm.sync.Unlock()

	addrStr, _ := contractAddr.HexString()
	if methods, ok := pm.methodsMap[addrStr]; ok {
		if !tpcmm.IsContainString(method, methods) {
			methods = append(methods, method)
		}
	}
}

func (pm *PermissionContractMethod) RemoveMethod(contractAddr tpcrtypes.Address, method string) {
	pm.sync.Lock()
	defer pm.sync.Unlock()

	addrStr, _ := contractAddr.HexString()
	if methods, ok := pm.methodsMap[addrStr]; ok {
		tpcmm.RemoveIfExistString(method, methods)
	}
}

func (pm *PermissionContractMethod) HasMethodPerm(contractAddr tpcrtypes.Address, method string) bool {
	pm.sync.RLock()
	defer pm.sync.RUnlock()

	addrStr, _ := contractAddr.HexString()
	if methods, ok := pm.methodsMap[addrStr]; ok {
		if tpcmm.IsContainString(method, methods) {
			return true
		}
	}

	return false
}

func (pr *PermissionContractMethod) SamePerm(other Permission) bool {
	perm, ok := other.(*PermissionContractMethod)
	if !ok {
		return false
	}

	return reflect.DeepEqual(pr.methodsMap, perm.methodsMap)
}
