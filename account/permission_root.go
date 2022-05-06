package account

import (
	"encoding/json"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
)

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

func (pr *PermissionRoot) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct{}{})
}

func (pr *PermissionRoot) UnmarshalJSON(data []byte) error {
	var prRTN struct{}
	err := json.Unmarshal(data, &prRTN)
	if err != nil {
		return err
	}

	return nil
}
