package account

import tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"

type PermissionRoot struct {
}

func NewPermissionRoot() Permission {
	return &PermissionRoot{}
}

func (pr *PermissionRoot) IsRoot() bool {
	return true
}

func (pr *PermissionRoot) AddMethod(contractAddr tpcrtypes.Address, method string) {

}

func (pr *PermissionRoot) RemoveMethod(contractAddr tpcrtypes.Address, method string) {

}

func (pr *PermissionRoot) HasMethodPerm(contractAddr tpcrtypes.Address, method string) bool {
	return true
}

func (pr *PermissionRoot) SamePerm(other Permission) bool {
	return other.IsRoot()
}
