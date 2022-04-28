package common

type ReturnCode int

const (
	ReturnCode_Ok ReturnCode = iota
	ReturnCode_NotEnable
	ReturnCode_InvalidVMCtx
	ReturnCode_ContractNotFound
	ReturnCode_MethodNotFound
	ReturnCode_InvalidParam
	ReturnCode_InvalidReturn
	ReturnCode_MethodErr
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
