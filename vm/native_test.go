package vm

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/ledger"
	"github.com/TopiaNetwork/topia/ledger/backend"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"github.com/TopiaNetwork/topia/state"
	tpvmcmm "github.com/TopiaNetwork/topia/vm/common"
)

func TestExecuteContract(t *testing.T) {
	log, _ := tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.DefaultLogFormat, tplog.DefaultLogOutput, "")
	lg := ledger.NewLedger(".", "NCTest", log, backend.BackendType_Badger)

	compState := state.GetStateBuilder().CreateCompositionState(log, "NCTest", lg, 1, "NCTest")

	sParam := "testNode"

	vmContext := &tpvmcmm.VMContext{
		Context:          context.Background(),
		CompositionState: compState,
		ContractName:     "ContractTest",
		Method:           "TestFuncWithStruct",
		Args:             []reflect.Value{reflect.ValueOf(sParam)},
	}

	GetVMFactory().SetLogger(tplogcmm.InfoLevel, log)

	vmResult, err := GetVMFactory().GetVM(tpvmcmm.VMType_NATIVE).ExecuteContract(vmContext)
	assert.Equal(t, nil, err)
	assert.NotEqual(t, nil, vmResult)
	assert.Equal(t, tpvmcmm.ReturnCode_Ok, vmResult.Code)
	assert.Equal(t, 1000, int(vmResult.Data.(*tpcmm.NodeInfo).Weight))
}
