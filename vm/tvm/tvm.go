package tvm

import (
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpvmmservice "github.com/TopiaNetwork/topia/vm/service"
	tpvmtype "github.com/TopiaNetwork/topia/vm/type"
)

func NewTopiaVM() *TopiaVM {
	return &TopiaVM{}
}

type TopiaVM struct {
}

func (tvm *TopiaVM) init() {
	//TODO implement me
	panic("implement me")
}

func (tvm *TopiaVM) Version() int {
	//TODO implement me
	panic("implement me")
}

func (tvm *TopiaVM) Type() tpvmtype.VMType {
	//TODO implement me
	panic("implement me")
}

func (tvm *TopiaVM) Enable() bool {
	//TODO implement me
	panic("implement me")
}

func (tvm *TopiaVM) UpdateState(state bool) {
	//TODO implement me
	panic("implement me")
}

func (tvm *TopiaVM) SetLogger(level tplogcmm.LogLevel, log tplog.Logger) {

}

func (tvm *TopiaVM) DeployContract(ctx *tpvmmservice.VMContext) (*tpvmtype.VMResult, error) {
	//TODO implement me
	panic("implement me")
}

func (tvm *TopiaVM) ExecuteContract(ctx *tpvmmservice.VMContext) (*tpvmtype.VMResult, error) {
	//TODO implement me
	panic("implement me")
}
