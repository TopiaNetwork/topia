package tvm

import (
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"github.com/TopiaNetwork/topia/vm/common"
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

func (tvm *TopiaVM) Type() common.VMType {
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

func (tvm *TopiaVM) DeployContract(ctx *common.VMContext) (*common.VMResult, error) {
	//TODO implement me
	panic("implement me")
}

func (tvm *TopiaVM) ExecuteContract(ctx *common.VMContext) (*common.VMResult, error) {
	//TODO implement me
	panic("implement me")
}
