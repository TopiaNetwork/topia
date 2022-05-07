package _type

type ReturnCode int

const (
	ReturnCode_Ok                 ReturnCode = iota
	ReturnCode_NotEnable                     //VM Disable
	ReturnCode_InvalidVMCtx                  //Invalid vm context
	ReturnCode_ContractNotFound              //Contract not found
	ReturnCode_MethodNotFound                //Method not found
	ReturnCode_InvalidParam                  //Invalid input params
	ReturnCode_InvalidReturn                 //Invalid return values
	ReturnCode_MethodSignatureErr            //Method signature error
	ReturnCode_ExecuteErr                    //VM execute error
	ReturnCode_InsufficientGas               //Insufficient Gas
	ReturnCode_CallStackOverFlow             //Call stack overflow
	ReturnCode_StorageErr                    //Data storage error
	ReturnCode_LoadErr                       //Data load error
	ReturnCode_InvalidAccount                //Invalid account info
	ReturnCode_PermissionErr                 //Permission error
	ReturnCode_UnknownErr
)

type VMType byte

const (
	VMType_Unknown VMType = iota
	VMType_NATIVE
	VMType_TVM
)

type VMResult struct {
	Code    ReturnCode
	ErrMsg  string
	GasUsed uint64
	Data    []byte
}

func (vt VMType) String() string {
	switch vt {
	case VMType_NATIVE:
		return "native"
	case VMType_TVM:
		return "tvm"
	default:
		return "unknown"
	}
}

func (vt VMType) Value(vtStr string) VMType {
	switch vtStr {
	case "native":
		return VMType_NATIVE
	case "tvm":
		return VMType_TVM
	default:
		return VMType_Unknown
	}
}

func (rc ReturnCode) String() string {
	switch rc {
	case ReturnCode_Ok:
		return "Ok"
	case ReturnCode_NotEnable:
		return "VM Disable"
	case ReturnCode_InvalidVMCtx:
		return "Invalid vm context"
	case ReturnCode_ContractNotFound:
		return "Contract not found"
	case ReturnCode_MethodNotFound:
		return "Method not found"
	case ReturnCode_InvalidParam:
		return "Invalid input params"
	case ReturnCode_InvalidReturn:
		return "Invalid return values"
	case ReturnCode_MethodSignatureErr:
		return "Method signature error"
	case ReturnCode_ExecuteErr:
		return "VM execute error"
	case ReturnCode_InsufficientGas:
		return "Insufficient Gas"
	case ReturnCode_CallStackOverFlow:
		return "Call stack overflow"
	case ReturnCode_StorageErr:
		return "Data storage error"
	case ReturnCode_LoadErr:
		return "Data load error"
	case ReturnCode_InvalidAccount:
		return "Invalid account info"
	case ReturnCode_PermissionErr:
		return "Permission error"
	case ReturnCode_UnknownErr:
		return "Unknown error"
	default:
		return "unknown"
	}
}
