package transactionpool

import (
	"context"
	"encoding/hex"
	"fmt"
	txpoolcore "github.com/TopiaNetwork/topia/transaction_pool/core"
	"math/big"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpconfig "github.com/TopiaNetwork/topia/configuration"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/currency"
	"github.com/TopiaNetwork/topia/eventhub"
	"github.com/TopiaNetwork/topia/ledger"
	"github.com/TopiaNetwork/topia/ledger/backend"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpnet "github.com/TopiaNetwork/topia/network"
	"github.com/TopiaNetwork/topia/state"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txuni "github.com/TopiaNetwork/topia/transaction/universal"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
	txpoolmock "github.com/TopiaNetwork/topia/transaction_pool/mock"
)

var (
	txPool           txpooli.TransactionPool
	cryptService     tpcrt.CryptService
	stateServiceMock *txpoolmock.MockStateQueryService
	blockServiceMock *txpoolmock.MockBlockService

	txList []*txbasic.Transaction
)

func createLedger(log tplog.Logger, rootDir string, backendType backend.BackendType, i int, nodeType string) ledger.Ledger {
	ledgerID := fmt.Sprintf("%s%d", nodeType, i+1)

	return ledger.NewLedger(rootDir, ledger.LedgerID(ledgerID), log, backendType)
}

func build(gCtrl *gomock.Controller) {
	sysActor := actor.NewActorSystem()

	testMainLog, _ := tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.JSONFormat, tplog.StdErrOutput, "")

	cryptService = tpcrt.CreateCryptService(testMainLog, tpcrtypes.CryptType_Ed25519)

	fromPriKey, _ := hex.DecodeString("35a64a1b110f499e0847b2ce87480063f81ead7bc10c65202d7cadf87501c11ce91b83bc5b42918daa92e38691f4f976a30401f0f7d41a6d275d4370c80180ed")
	for i := 0; i < 5; i++ {
		_, toPubKey, _ := cryptService.GeneratePriPubKey()
		toAddr, _ := cryptService.CreateAddress(toPubKey)

		tx := txuni.ConstructTransactionWithUniversalTransfer(testMainLog, cryptService, fromPriKey, fromPriKey, uint64(i+1), 1000000000, 50000, toAddr,
			[]txuni.TargetItem{{currency.TokenSymbol_Native, big.NewInt(10)}})

		txList = append(txList, tx)
	}

	l := createLedger(testMainLog, "./TestTxpool", backend.BackendType_Badger, 0, "executor")

	network := tpnet.NewNetwork(context.Background(), testMainLog, tpconfig.DefNetworkConfiguration(), sysActor, "/ip4/127.0.0.1/tcp/41001", "topia0", state.NewNodeNetWorkStateWapper(testMainLog, l))

	eventhub.GetEventHubManager().CreateEventHub(network.ID(), tplogcmm.InfoLevel, testMainLog)

	stateServiceMock = txpoolmock.NewMockStateQueryService(gCtrl)
	blockServiceMock = txpoolmock.NewMockBlockService(gCtrl)

	stateServiceMock.EXPECT().GetLatestBlock().Return(&tpchaintypes.Block{Head: &tpchaintypes.BlockHead{Height: 100}}, nil).AnyTimes()

	txPool = NewTransactionPool(
		"testdomainid",
		network.ID(),
		context.Background(),
		tpconfig.DefaultTransactionPoolConfig(),
		tplogcmm.InfoLevel,
		testMainLog,
		codec.CodecType_PROTO,
		stateServiceMock,
		blockServiceMock,
		network,
		l,
	)

	txPool.Start(sysActor, network)
}

func TestTransactionPool_AddTx(t *testing.T) {
	gCtrl := gomock.NewController(t)
	build(gCtrl)

	assert.NotEqual(t, nil, txPool)

	for _, tx := range txList {
		txPool.AddTx(tx, true)
	}

	time.Sleep(5 * time.Second) //waiting for all txs are added

	assert.Equal(t, int64(len(txList)), txPool.Count())

	txps := txPool.PickTxs()
	localTxs := txPool.GetLocalTxs()

	assert.Equal(t, len(txList), len(txps))
	assert.Equal(t, len(txList), len(localTxs))

	txPool.Stop()

	t.Logf("Current tx pool count: %d", txPool.Count())
}

func TestTransactionPool_RemoveTxByKey(t *testing.T) {
	gCtrl := gomock.NewController(t)
	build(gCtrl)

	assert.NotEqual(t, nil, txPool)

	for _, tx := range txList {
		txPool.AddTx(tx, true)
	}

	time.Sleep(5 * time.Second) //waiting for all txs are added

	assert.Equal(t, int64(len(txList)), txPool.Count())

	txId, _ := txList[3].TxID()

	txPool.RemoveTxByKey(txId)

	assert.Equal(t, int64(len(txList)-1), txPool.Count())
}

func TestTransactionPool_Store_Save(t *testing.T) {
	gCtrl := gomock.NewController(t)
	build(gCtrl)

	assert.NotEqual(t, nil, txPool)

	txPoolS := txPool.(*transactionPool)

	for _, tx := range txList {
		txID, _ := tx.TxID()
		txPoolS.txServant.saveTxIntoStore(txPoolS.marshaler, txID, 100, tx)
	}

	txPoolS.txServant.stop()
}

func TestTransactionPool_Store_Load(t *testing.T) {
	gCtrl := gomock.NewController(t)
	build(gCtrl)

	assert.NotEqual(t, nil, txPool)

	txPoolS := txPool.(*transactionPool)

	txPoolS.txServant.loadAllTxs(txPoolS.marshaler, func(tx *txbasic.Transaction, height uint64) (txpoolcore.TxWrapper, error) {
		txID, _ := tx.TxID()
		t.Logf("Saved tx: id %s, height %d", txID, height)
		return nil, nil
	})
}
