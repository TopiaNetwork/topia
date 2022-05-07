package test

import (
	"context"
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	tpvmmservice "github.com/TopiaNetwork/topia/vm/service"

	tpacc "github.com/TopiaNetwork/topia/account"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/ledger"
	"github.com/TopiaNetwork/topia/ledger/backend"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"github.com/TopiaNetwork/topia/state"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	tpvm "github.com/TopiaNetwork/topia/vm"
	tpvmtype "github.com/TopiaNetwork/topia/vm/type"
)

func TestExecuteContract(t *testing.T) {
	log, _ := tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.DefaultLogFormat, tplog.DefaultLogOutput, "")
	lg := ledger.NewLedger(".", "NCTest", log, backend.BackendType_Badger)

	compState := state.GetStateBuilder().CreateCompositionState(log, "NCTest", lg, 1, "NCTest")

	addr := tpcrtypes.Address("TestAddr")
	acc := tpacc.NewDefaultAccount("TestAddr")
	compState.AddAccount(acc)

	sParam := "testNode"

	txServant := txbasic.NewTransactionServant(compState, compState)

	vmServant := tpvmmservice.NewVMServant(txServant, math.MaxUint64)

	ctx := context.Background()
	ctx = context.WithValue(ctx, tpvmmservice.VMCtxKey_FromAddr, &addr)
	//ctx = context.WithValue(ctx, tpvmcmm.VMCtxKey_VMServant, vmServant)

	vmContext := &tpvmmservice.VMContext{
		Context:      ctx,
		VMServant:    vmServant,
		ContractAddr: "ContractTest",
		Method:       "TestFuncWithStruct",
		Args:         sParam,
	}

	tpvm.GetVMFactory().SetLogger(tplogcmm.InfoLevel, log)

	vmResult, err := tpvm.GetVMFactory().GetVM(tpvmtype.VMType_NATIVE).ExecuteContract(vmContext)
	assert.Equal(t, nil, err)
	assert.NotEqual(t, nil, vmResult)
	assert.Equal(t, tpvmtype.ReturnCode_Ok, vmResult.Code)

	rtnValMap := make(map[string][]byte)

	err = json.Unmarshal(vmResult.Data, &rtnValMap)
	for key, val := range rtnValMap {
		switch key {
		case "*common.NodeInfo_1":
			var nodeInfo tpcmm.NodeInfo
			err = json.Unmarshal(val, &nodeInfo)
			assert.Equal(t, nil, err)
			assert.Equal(t, uint64(1000), nodeInfo.Weight)
		case "int_2":
			var iRtn2 int
			err = json.Unmarshal(val, &iRtn2)
			assert.Equal(t, nil, err)
			assert.Equal(t, 100, iRtn2)
		case "string_3":
			var sRtn3 string
			err = json.Unmarshal(val, &sRtn3)
			assert.Equal(t, nil, err)
			assert.Equal(t, "ContractTest", sRtn3)
		}
	}
}
