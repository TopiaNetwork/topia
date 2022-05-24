package universal

import (
	"context"
	"fmt"
	"math"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	tpvm "github.com/TopiaNetwork/topia/vm"
	tpvmmservice "github.com/TopiaNetwork/topia/vm/service"
	tpvmcmm "github.com/TopiaNetwork/topia/vm/type"
)

type TransactionUniversaSimulate struct {
	nodeID       string
	contractAddr tpcrtypes.Address
	method       string
	args         string
	txType       TransactionUniversalType
	txServant    txbasic.TransactionServant
}

func (sim *TransactionUniversaSimulate) Execute() (uint64, error) {
	vmServant := tpvmmservice.NewVMServant(sim.txServant, math.MaxUint64)
	vmContext := &tpvmmservice.VMContext{
		Context:      context.Background(),
		VMServant:    vmServant,
		NodeID:       sim.nodeID,
		ContractAddr: sim.contractAddr,
		Method:       sim.method,
		Args:         sim.args,
	}

	var vm tpvm.VirtualMachine
	switch sim.txType {
	case TransactionUniversalType_NativeInvoke:
		vm = tpvm.GetVMFactory().GetVM(tpvmcmm.VMType_NATIVE)
	case TransactionUniversalType_ContractInvoke:
		vm = tpvm.GetVMFactory().GetVM(tpvmcmm.VMType_TVM)
	}

	if vm == nil {
		return 0, fmt.Errorf("Can't find the vm of %s", sim.txType.String())
	}
	vmResult, err := vm.ExecuteContract(vmContext)
	if err != nil {
		return 0, err
	}
	if vmResult.Code != tpvmcmm.ReturnCode_Ok {
		return 0, fmt.Errorf("%s", vmResult.ErrMsg)
	}

	return vmResult.GasUsed, nil
}
