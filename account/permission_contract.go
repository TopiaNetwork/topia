package account

import (
	"encoding/json"
	"reflect"

	tpcmm "github.com/TopiaNetwork/topia/common"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
)

type PermissionContractMethod struct {
	GasLimit   uint64
	MethodsMap map[string][]string //Contract Address->Methods
}

func NewPermissionContractMethod(gasLimit uint64) Permission {
	return &PermissionContractMethod{
		GasLimit:   gasLimit,
		MethodsMap: make(map[string][]string),
	}
}

func (pm *PermissionContractMethod) IsRoot() bool {
	return false
}

func (pm *PermissionContractMethod) AddMethod(contractAddr tpcrtypes.Address, method string) {
	if method == "" {
		return
	}

	addrStr, _ := contractAddr.HexString()
	if methods, ok := pm.MethodsMap[addrStr]; ok {
		if !tpcmm.IsContainString(method, methods) {
			methods = append(methods, method)
		}
	}
}

func (pm *PermissionContractMethod) RemoveMethod(contractAddr tpcrtypes.Address, method string) {
	addrStr, _ := contractAddr.HexString()
	if methods, ok := pm.MethodsMap[addrStr]; ok {
		tpcmm.RemoveIfExistString(method, methods)
	}
}

func (pm *PermissionContractMethod) HasMethodPerm(contractAddr tpcrtypes.Address, method string) bool {
	addrStr, _ := contractAddr.HexString()
	if methods, ok := pm.MethodsMap[addrStr]; ok {
		if tpcmm.IsContainString(method, methods) {
			return true
		}
	}

	return false
}

func (pm *PermissionContractMethod) SamePerm(other Permission) bool {
	perm, ok := other.(*PermissionContractMethod)
	if !ok {
		return false
	}

	return reflect.DeepEqual(pm.MethodsMap, perm.MethodsMap)
}

func (pm *PermissionContractMethod) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		GasLimit   uint64
		MethodsMap map[string][]string
	}{
		GasLimit:   pm.GasLimit,
		MethodsMap: pm.MethodsMap,
	})
}

func (pm *PermissionContractMethod) UnmarshalJSON(data []byte) error {
	var pmObj struct {
		GasLimit   uint64
		MethodsMap map[string][]string
	}
	err := json.Unmarshal(data, &pmObj)
	if err != nil {
		return err
	}

	pm.GasLimit = pmObj.GasLimit
	pm.MethodsMap = pmObj.MethodsMap

	return nil
}
