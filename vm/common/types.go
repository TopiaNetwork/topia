package common

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
