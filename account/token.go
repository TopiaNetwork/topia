package account

import (
	"encoding/json"
	"fmt"
)

type AccountToken struct {
	Nonce      uint64
	permMode   PermissionModel
	Permission Permission
}

func NewAccountToken(perm Permission) *AccountToken {
	permMode := PermissionModel_ContractMethod
	if perm.IsRoot() {
		permMode = PermissionModel_ROOT
	}

	return &AccountToken{
		permMode:   permMode,
		Permission: perm,
	}
}

func (at *AccountToken) MarshalJSON() ([]byte, error) {
	permBytes, _ := at.Permission.MarshalJSON()
	return json.Marshal(&struct {
		Nonce          uint64
		PermMode       PermissionModel
		PermissionData []byte
	}{
		Nonce:          at.Nonce,
		PermMode:       at.permMode,
		PermissionData: permBytes,
	})
}

func (at *AccountToken) UnmarshalJSON(data []byte) error {
	var accToken struct {
		Nonce          uint64
		PermMode       PermissionModel
		PermissionData []byte
	}
	err := json.Unmarshal(data, &accToken)
	if err != nil {
		return err
	}

	at.Nonce = accToken.Nonce
	at.permMode = accToken.PermMode

	var perm Permission
	switch at.permMode {
	case PermissionModel_ROOT:
		perm = NewPermissionRoot()
	case PermissionModel_ContractMethod:
		perm = &PermissionContractMethod{}
	default:
		panic("Invalid permission mode: " + fmt.Sprintf("%d", at.permMode))
	}

	err = perm.UnmarshalJSON(accToken.PermissionData)
	if err == nil {
		at.Permission = perm
	}

	return err
}
