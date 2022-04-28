package vm

import (
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpvmcmm "github.com/TopiaNetwork/topia/vm/common"
)

type VirtualMachine interface {
	Version() int

	Type() tpvmcmm.VMType

	Enable() bool

	UpdateState(state bool)

	SetLogger(level tplogcmm.LogLevel, log tplog.Logger)

	DeployContract(ctx *tpvmcmm.VMContext) (*tpvmcmm.VMResult, error)

	ExecuteContract(ctx *tpvmcmm.VMContext) (*tpvmcmm.VMResult, error)
}
