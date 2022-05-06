package account

import tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"

type PermissionModel int

const (
	PermissionModel_Unknown PermissionModel = iota
	PermissionModel_ROOT
	PermissionModel_ContractMethod
)

type Permission interface {
	IsRoot() bool

	AddMethod(contractAddr tpcrtypes.Address, method string)

	RemoveMethod(contractAddr tpcrtypes.Address, method string)

	HasMethodPerm(contractAddr tpcrtypes.Address, method string) bool

	SamePerm(other Permission) bool

	MarshalJSON() ([]byte, error)

	UnmarshalJSON(data []byte) error
}

func (model PermissionModel) String() string {
	switch model {
	case PermissionModel_ROOT:
		return "ROOT"
	case PermissionModel_ContractMethod:
		return "ContractMethod"
	default:
		return "Unknown"
	}
}

func (model PermissionModel) Value(ms string) PermissionModel {
	switch ms {
	case "ROOT":
		return PermissionModel_ROOT
	case "ContractMethod":
		return PermissionModel_ContractMethod
	default:
		return PermissionModel_Unknown
	}
}
