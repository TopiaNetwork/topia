package vm

import (
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpvmmservice "github.com/TopiaNetwork/topia/vm/service"
	tpvmcmm "github.com/TopiaNetwork/topia/vm/type"
)

type VirtualMachine interface {
	Version() int

	Type() tpvmcmm.VMType

	Enable() bool

	UpdateState(state bool)

	SetLogger(level tplogcmm.LogLevel, log tplog.Logger)

	DeployContract(ctx *tpvmmservice.VMContext) (*tpvmcmm.VMResult, error)

	ExecuteContract(ctx *tpvmmservice.VMContext) (*tpvmcmm.VMResult, error)
}
