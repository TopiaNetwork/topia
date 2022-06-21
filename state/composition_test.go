package state

import (
	tpconfig "github.com/TopiaNetwork/topia/configuration"
	"github.com/TopiaNetwork/topia/ledger"
	"github.com/TopiaNetwork/topia/ledger/backend"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMultiCompositionState(t *testing.T) {
	testLog, _ := tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.JSONFormat, tplog.StdErrOutput, "")

	l := ledger.NewLedger("./TestCS", ledger.LedgerID("testledger"), testLog, backend.BackendType_Badger)

	config := tpconfig.GetConfiguration()

	compState := GetStateBuilder(CompStateBuilderType_Full).CreateCompositionState(testLog, "", l, 1, "tester")

	compState.SetLatestBlock(config.Genesis.Block)

	compState.SetLatestBlockResult(config.Genesis.BlockResult)

	compState.SetLatestEpoch(config.Genesis.Epoch)

	compState.Commit()

	compState2 := GetStateBuilder(CompStateBuilderType_Full).CreateCompositionState(testLog, "", l, 2, "tester")

	compState3 := GetStateBuilder(CompStateBuilderType_Full).CreateCompositionState(testLog, "", l, 3, "tester")

	config.Genesis.Block.Head.Height = 2
	compState2.SetLatestBlock(config.Genesis.Block)
	compState2.Commit()

	compStateRN := CreateCompositionStateReadonly(testLog, l)
	latestBlock, _ := compStateRN.GetLatestBlock()
	compStateRN.Stop()
	assert.Equal(t, uint64(2), latestBlock.Head.Height)

	config.Genesis.Block.Head.Height = 3
	err := compState3.SetLatestBlock(config.Genesis.Block)
	assert.Equal(t, nil, err)
	latestBlockBeforeCommit, _ := compState3.GetLatestBlock()
	assert.Equal(t, uint64(3), latestBlockBeforeCommit.Head.Height)
	err = compState3.Commit()
	assert.Equal(t, nil, err)

	compStateRN = CreateCompositionStateReadonly(testLog, l)
	latestBlock, _ = compStateRN.GetLatestBlock()
	compStateRN.Stop()
	assert.Equal(t, uint64(3), latestBlock.Head.Height)

	compStateRN = CreateCompositionStateReadonlyAt(testLog, l, 2)
	latestBlock, _ = compStateRN.GetLatestBlock()
	compStateRN.Stop()
	assert.Equal(t, uint64(2), latestBlock.Head.Height)
}

func TestMemCompositionState(t *testing.T) {
	testLog, _ := tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.JSONFormat, tplog.StdErrOutput, "")

	l := ledger.NewLedger("./TestCS", ledger.LedgerID("testledger"), testLog, backend.BackendType_Badger)

	config := tpconfig.GetConfiguration()

	compState := GetStateBuilder(CompStateBuilderType_Full).CreateCompositionState(testLog, "", l, 1, "tester")

	compState.SetLatestBlock(config.Genesis.Block)

	compState.SetLatestBlockResult(config.Genesis.BlockResult)

	compState.SetLatestEpoch(config.Genesis.Epoch)

	compState.Commit()

	compState2 := GetStateBuilder(CompStateBuilderType_Full).CreateCompositionState(testLog, "", l, 2, "tester")

	compStateMem := CreateCompositionStateMem(testLog, compState2)

	latestBlock, _ := compStateMem.GetLatestBlock()
	assert.Equal(t, uint64(1), latestBlock.Head.Height)

	config.Genesis.Block.Head.Height = 2
	compStateMem.SetLatestBlock(config.Genesis.Block)
	latestBlock, _ = compStateMem.GetLatestBlock()
	assert.Equal(t, uint64(2), latestBlock.Head.Height)
}
