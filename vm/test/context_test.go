package test

import (
	"context"
	"fmt"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	"github.com/stretchr/testify/assert"
	"math"
	"reflect"
	"testing"
	"unsafe"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/ledger"
	"github.com/TopiaNetwork/topia/ledger/backend"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"github.com/TopiaNetwork/topia/state"
	tpvmcmm "github.com/TopiaNetwork/topia/vm/common"
)

func TestContextContract(t *testing.T) {
	ctx := context.Background()

	log, _ := tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.DefaultLogFormat, tplog.DefaultLogOutput, "")
	lg := ledger.NewLedger(".", "NCTest", log, backend.BackendType_Badger)

	compState := state.GetStateBuilder().CreateCompositionState(log, "NCTest", lg, 1, "NCTest")

	txServant := txbasic.NewTransactionServant(compState, compState)

	vmServant := tpvmcmm.NewVMServant(txServant, math.MaxUint64)

	addr := tpcrtypes.Address("testaddr")
	ctx = context.WithValue(ctx, tpvmcmm.VMCtxKey_VMServant, vmServant)
	ctx = context.WithValue(ctx, tpvmcmm.VMCtxKey_FromAddr, &addr)

	cCtx := &tpvmcmm.ContractContext{
		Context: ctx,
	}

	var vmS tpvmcmm.VMServant
	fmt.Printf("type %s", reflect.TypeOf(vmServant))
	var fromAddr *tpcrtypes.Address
	err := cCtx.GetCtxValues([]tpvmcmm.VMCtxKey{tpvmcmm.VMCtxKey_VMServant, tpvmcmm.VMCtxKey_FromAddr}, []unsafe.Pointer{unsafe.Pointer(&vmS), unsafe.Pointer(&fromAddr)})

	assert.Equal(t, nil, err)
	assert.NotEqual(t, nil, vmS)
	assert.NotEqual(t, nil, fromAddr)
	assert.Equal(t, "testaddr", string(*fromAddr))
}
