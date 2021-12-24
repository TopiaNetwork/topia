package vm

import (
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpvmcmm "github.com/TopiaNetwork/topia/vm/common"
	"github.com/TopiaNetwork/topia/vm/native"
	"github.com/TopiaNetwork/topia/vm/tvm"
)

type VMExecutorType byte

const (
	VMExecutorType_Unknown VMExecutorType = iota
	VMExecutorType_NATIVE
	VMExecutorType_TVM
)

type VMExecutor interface {
	Version() string

	Name() string

	CreateSmartContract(vmIn *tpvmcmm.VMInputParas) (*tpvmcmm.VMResult, error)

	ExecuteSmartContract(vmIn *tpvmcmm.VMInputParas) (*tpvmcmm.VMResult, error)
}

func CreateVMExecutor(vmType VMExecutorType, log tplog.Logger) VMExecutor {
	vmLog := tplog.CreateModuleLogger(tplogcmm.InfoLevel, "vmexecutor", log)
	switch vmType {
	case VMExecutorType_NATIVE:
		return native.NewVMExecutorNative(log)
	case VMExecutorType_TVM:
		return tvm.NewVMExecutorTVM(log)
	default:
		vmLog.Panicf("invalid vm type %d", vmType)
	}

	return nil
}
