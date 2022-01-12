package tvm

import (
	"github.com/TopiaNetwork/topia/log"
	tpvmcmm "github.com/TopiaNetwork/topia/vm/common"
)

type VMExecutorTVM struct {
	log log.Logger
}

func NewVMExecutorTVM(log log.Logger) *VMExecutorTVM {
	return &VMExecutorTVM{
		log: log,
	}
}

func (vm *VMExecutorTVM) Version() string {
	//TODO implement me
	panic("implement me")
}

func (vm *VMExecutorTVM) Name() string {
	//TODO implement me
	panic("implement me")
}

func (vm *VMExecutorTVM) CreateSmartContract(vmIn *tpvmcmm.VMInputParas) (*tpvmcmm.VMResult, error) {
	//TODO implement me
	panic("implement me")
}

func (vm *VMExecutorTVM) ExecuteSmartContract(vmIn *tpvmcmm.VMInputParas) (*tpvmcmm.VMResult, error) {
	//TODO implement me
	panic("implement me")
}
