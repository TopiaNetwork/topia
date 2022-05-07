package test

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/ledger"
	"github.com/TopiaNetwork/topia/ledger/backend"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"github.com/TopiaNetwork/topia/state"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	tpvmmservice "github.com/TopiaNetwork/topia/vm/service"
)

func TestContextContract(t *testing.T) {
	ctx := context.Background()

	log, _ := tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.DefaultLogFormat, tplog.DefaultLogOutput, "")
	lg := ledger.NewLedger(".", "NCTest", log, backend.BackendType_Badger)

	compState := state.GetStateBuilder().CreateCompositionState(log, "NCTest", lg, 1, "NCTest")

	txServant := txbasic.NewTransactionServant(compState, compState)

	vmServant := tpvmmservice.NewVMServant(txServant, math.MaxUint64)

	addr := tpcrtypes.Address("testaddr")
	ctx = context.WithValue(ctx, tpvmmservice.VMCtxKey_VMServant, vmServant)
	ctx = context.WithValue(ctx, tpvmmservice.VMCtxKey_FromAddr, &addr)

	cCtx := &tpvmmservice.ContractContext{
		Context: ctx,
	}

	var vmS tpvmmservice.VMServant
	fmt.Printf("type %s", reflect.TypeOf(vmServant))
	var fromAddr *tpcrtypes.Address
	err := cCtx.GetCtxValues([]tpvmmservice.VMCtxKey{tpvmmservice.VMCtxKey_VMServant, tpvmmservice.VMCtxKey_FromAddr}, []unsafe.Pointer{unsafe.Pointer(&vmS), unsafe.Pointer(&fromAddr)})

	assert.Equal(t, nil, err)
	assert.NotEqual(t, nil, vmS)
	assert.NotEqual(t, nil, fromAddr)
	assert.Equal(t, "testaddr", string(*fromAddr))
}
