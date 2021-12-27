package native

import (
	"github.com/TopiaNetwork/topia/log"
	tpvmcmm "github.com/TopiaNetwork/topia/vm/common"
)

type VMExecutorNative struct {
	log log.Logger
}

func NewVMExecutorNative(log log.Logger) *VMExecutorNative {
	return &VMExecutorNative{
		log: log,
	}
}

func (vm *VMExecutorNative) Version() string {
	//TODO implement me
	panic("implement me")
}

func (vm *VMExecutorNative) Name() string {
	//TODO implement me
	panic("implement me")
}

func (vm *VMExecutorNative) CreateSmartContract(vmIn *tpvmcmm.VMInputParas) (*tpvmcmm.VMResult, error) {
	//TODO implement me
	panic("implement me")
}

func (vm *VMExecutorNative) ExecuteSmartContract(vmIn *tpvmcmm.VMInputParas) (*tpvmcmm.VMResult, error) {
	//TODO implement me
	panic("implement me")
}
