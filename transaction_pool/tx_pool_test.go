package transactionpool

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/TopiaNetwork/topia/account"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/common/types"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"github.com/TopiaNetwork/topia/transaction"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"reflect"
	"testing"
	"time"
)

var (
	TestTxPoolConfig                                                    TransactionPoolConfig
	Tx1, Tx2, Tx3, Tx4, TxR1, TxR2, TxR3, TxlowGasPrice, TxHighGasLimit *transaction.Transaction
	Key1, Key2, Key3, Key4, KeyR1, KeyR2, KeyR3                         string
	From1, From2                                                        account.Address
	TestBlock                                                           *types.Block
	TestBlockHead                                                       *types.BlockHead
	TestBlockData                                                       *types.BlockData
	TestBlockHash                                                       string
)

func init() {
	TestTxPoolConfig = DefaultTransactionPoolConfig

	Tx1 = settransactionlocal(1, 10000, 12345)
	time.Sleep(100 * time.Millisecond)
	Tx2 = settransactionlocal(2, 20000, 12345)
	time.Sleep(100 * time.Millisecond)
	Tx3 = settransactionlocal(3, 9999, 12345)
	time.Sleep(100 * time.Millisecond)
	Tx4 = settransactionlocal(4, 40000, 12345)
	time.Sleep(100 * time.Millisecond)
	Key1, _ = Tx1.TxID()
	Key2, _ = Tx2.TxID()
	Key3, _ = Tx3.TxID()
	Key4, _ = Tx4.TxID()
	From1 = account.Address(hex.EncodeToString(Tx1.FromAddr))
	TxR1 = settransactionremote(1, 50000, 12345)
	time.Sleep(100 * time.Millisecond)
	TxR2 = settransactionremote(2, 48888, 12345)
	time.Sleep(100 * time.Millisecond)
	TxR3 = settransactionremote(3, 46666, 12345)
	KeyR1, _ = TxR1.TxID()
	KeyR2, _ = TxR2.TxID()
	KeyR3, _ = TxR3.TxID()

	TxlowGasPrice = settransactionremote(1, 100, 1000)
	TxHighGasLimit = settransactionremote(2, 10000, 9987654321)

	From2 = account.Address(hex.EncodeToString(TxR1.FromAddr))

	TestBlockHash = "0x224111a2131c213b213112d121c1231e"

	TestBlockHead = &types.BlockHead{
		ChainID:              []byte{0x01},
		Version:              0,
		Height:               0,
		Epoch:                0,
		Round:                0,
		ParentBlockHash:      []byte{0x32, 0x54, 0x12, 0x12, 0x87, 0x68, 0x12, 0x62},
		ProposeSignature:     []byte{0x32, 0x54, 0x12, 0x12, 0x87, 0x13, 0x68, 0x43},
		VoteAggSignature:     []byte{0x32, 0x54, 0x23, 0x12, 0x12, 0x87, 0x68, 0x12},
		ResultHash:           []byte{0x12, 0x43, 0x54, 0x23, 0x12, 0x53, 0x12, 0x43},
		TxCount:              2,
		TxHashRoot:           []byte{0x12, 0x73, 0x24, 0x23, 0x12, 0x53, 0x12, 0x43},
		TimeStamp:            0,
		DataHash:             []byte{0x12, 0x73, 0x24, 0x23, 0x12, 0x53, 0x12, 0x43},
		Reserved:             []byte{0x12, 0x73, 0x24, 0x23, 0x12, 0x53, 0x12, 0x43},
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     []byte{0x12, 0x73, 0x24, 0x23, 0x12, 0x53, 0x12, 0x43},
		XXX_sizecache:        0,
	}
	TestBlockData = &types.BlockData{
		Version:              0,
		Txs:                  [][]byte{{0x12, 0x32, 0x12, 0x32, 0x12, 0x32, 0x12, 0x32, 0x12}, {0x12, 0x32, 0x43, 0x32, 0x12, 0x32, 0x12, 0x32, 0x12}},
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}
	TestBlock = &types.Block{
		Head:                 TestBlockHead,
		Data:                 TestBlockData,
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     []byte{0x43, 0x12, 0x53, 0x13, 0x12, 0x53, 0x15},
		XXX_sizecache:        123,
	}

}

func settransactionlocal(nonce uint64, gaseprice, gaseLimit uint64) *transaction.Transaction {
	tx := &transaction.Transaction{
		FromAddr:   []byte{0x00, 0x00, 0x43, 0x53, 0x23, 0x34, 0x21, 0x12, 0x42, 0x12, 0x43},
		TargetAddr: []byte{0x00, 0x00, 0x34, 0x53, 0x23, 0x34, 0x21, 0x12, 0x42, 0x12, 0x43}, Version: 1, ChainID: []byte{0x01},
		Nonce: nonce, Value: []byte{0x12, 0x32}, GasPrice: gaseprice,
		GasLimit: gaseLimit, Data: []byte{0x32, 0x32, 0x32, 0x65, 0x32, 0x65, 0x32, 0x65},
		Signature: []byte{0x32, 0x23, 0x42, 0x23, 0x42, 0x23, 0x42}, Options: 0, Time: time.Now()}
	return tx
}
func settransactionremote(nonce uint64, gaseprice, gaseLimit uint64) *transaction.Transaction {
	tx := &transaction.Transaction{
		FromAddr:   []byte{0x01, 0x01, 0x43, 0x53, 0x23, 0x34, 0x21, 0x12, 0x42, 0x12, 0x43},
		TargetAddr: []byte{0x01, 0x01, 0x34, 0x53, 0x23, 0x34, 0x21, 0x12, 0x42, 0x12, 0x43}, Version: 1, ChainID: []byte{0x01},
		Nonce: nonce, Value: []byte{0x12, 0x32}, GasPrice: gaseprice,
		GasLimit: gaseLimit, Data: []byte{0x32, 0x32, 0x32, 0x65, 0x32, 0x65, 0x32, 0x65},
		Signature: []byte{0x32, 0x23, 0x42, 0x23, 0x42, 0x23, 0x42}, Options: 0, Time: time.Now()}
	return tx
}

func settxpoolconfig() *TransactionPoolConfig {
	conf := &TransactionPoolConfig{
		Locals: []account.Address{
			account.Address("0x2fadf9192731273"),
			account.Address("0x3fadfa123123123"),
			account.Address("0x3131313asa1daaf")},
		NoLocalFile:           false,
		NoRemoteFile:          false,
		NoConfigFile:          false,
		PathLocal:             "",
		PathRemote:            "",
		PathConfig:            "",
		ReStoredDur:           123,
		GasPriceLimit:         123,
		PendingAccountSlots:   123,
		PendingGlobalSlots:    123,
		QueueMaxTxsAccount:    123,
		QueueMaxTxsGlobal:     123,
		LifetimeForTx:         123,
		DurationForTxRePublic: 123,
	}
	return conf
}

func SetNewTransactionPool(conf TransactionPoolConfig, level tplogcmm.LogLevel, log tplog.Logger, codecType codec.CodecType) *transactionPool {
	conf = (&conf).check()
	poolLog := tplog.CreateModuleLogger(level, "TransactionPool", log)
	pool := &transactionPool{
		config:              conf,
		log:                 poolLog,
		level:               level,
		allTxsForLook:       newTxLookup(),
		ActivationIntervals: make(map[string]time.Time),

		chanChainHead:     make(chan transaction.ChainHeadEvent, chainHeadChanSize),
		chanReqReset:      make(chan *txPoolResetRequest),
		chanReqPromote:    make(chan *accountSet),
		chanReorgDone:     make(chan chan struct{}),
		chanReorgShutdown: make(chan struct{}),                 // requests shutdown of scheduleReorgLoop
		chanInitDone:      make(chan struct{}),                 // is closed once the pool is initialized (for tests)
		chanQueueTxEvent:  make(chan *transaction.Transaction), //check new tx insert to txpool
		chanRmTxs:         make(chan []string),

		marshaler: codec.CreateMarshaler(codecType),
		hasher:    tpcmm.NewBlake2bHasher(0),
	}
	//	pool.curMaxGasLimit = pool.query.GetMaxGasLimit()
	pool.pending = newPendingTxs()
	pool.queue = newQueueTxs()

	poolHandler := NewTransactionPoolHandler(poolLog, pool)

	pool.handler = poolHandler

	pool.locals = newAccountSet()
	if len(conf.Locals) > 0 {
		for _, addr := range conf.Locals {
			pool.log.Info("Setting new local account")
			pool.locals.add(addr)
		}
	}
	pool.sortedByPriced = newTxPricedList(pool.allTxsForLook) //done
	pool.curMaxGasLimit = uint64(987654321)

	//pool.Reset(nil, conf.chain.CurrentBlock().GetHead())
	//
	//pool.wg.Add(1)
	//go pool.scheduleReorgLoop()
	//
	//pool.loadLocal(conf.NoLocalFile, conf.PathLocal)
	//pool.loadRemote(conf.NoRemoteFile, conf.PathRemote)
	//pool.loadConfig(conf.NoConfigFile, conf.PathConfig)

	//pool.pubSubService = pool.config.chain.SubChainHeadEvent(pool.chanChainHead)
	//
	//go pool.loop()
	return pool
}

func TestTransactionPool_ValidateTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("test for ErrTxGasLimit:")
	if err := pool.ValidateTx(TxHighGasLimit, true); err != ErrTxGasLimit {
		t.Error("expected", ErrTxGasLimit, "got", err)
	}
	fmt.Println("test for ErrGasPriceTooLow:")
	if err := pool.ValidateTx(TxlowGasPrice, false); err != ErrGasPriceTooLow {
		t.Error("expected", ErrGasPriceTooLow, "got", err)
	}
}

func TestTransactionPool_AddLocal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	pool.AddLocal(Tx1)
	assert.Equal(t, 1, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 1, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 1, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
}
func TestTransactionPool_AddLocals(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	txs := make([]*transaction.Transaction, 0)
	txs = append(txs, Tx1)
	txs = append(txs, Tx2)
	pool.AddLocals(txs)
	assert.Equal(t, 1, len(pool.queue.accTxs))
	assert.Equal(t, 2, pool.queue.accTxs[From1].txs.Len())
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 2, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 2, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
}
func TestTransactionPool_LocalAccounts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	txs := make([]*transaction.Transaction, 0)
	txs = append(txs, Tx1)
	txs = append(txs, TxR1)
	pool.AddLocals(txs)
	accounts := make([]account.Address, 0)
	accounts = append(accounts, From1)
	accounts = append(accounts, From2)
	if !reflect.DeepEqual(accounts, pool.LocalAccounts()) {
		t.Error("want:", accounts, "got:", pool.LocalAccounts())
	}

}
func TestTransactionPool_GetLocalTxs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	txs := make([]*transaction.Transaction, 0)
	txs = append(txs, Tx1)
	txs = append(txs, Tx2)
	txsMap := make(map[account.Address][]*transaction.Transaction)
	txsMap[From1] = txs
	pool.AddLocals(txs)
	got := pool.GetLocalTxs()
	if !reflect.DeepEqual(txsMap, pool.GetLocalTxs()) {
		t.Error("want:", txsMap, "got:", got)
	}
}

func TestTransactionPool_AddRemote(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	pool.AddRemote(TxR1)
	assert.Equal(t, 1, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 1, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 1, len(pool.sortedByPriced.all.remotes))
}
func TestTransactionPool_AddRemotes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	txs := make([]*transaction.Transaction, 0)
	txs = append(txs, TxR1)
	txs = append(txs, TxR2)
	txsMap := make(map[account.Address][]*transaction.Transaction)
	txsMap[From2] = txs
	pool.AddRemotes(txs)
	assert.Equal(t, 1, len(pool.queue.accTxs))
	assert.Equal(t, 2, pool.queue.accTxs[From2].txs.Len())
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 2, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 2, len(pool.sortedByPriced.all.remotes))
}

func TestTransactionPool_AddTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	fmt.Println("test add local tx")
	pool.AddTx(Tx1, true)
	assert.Equal(t, 1, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 1, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 1, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	fmt.Println("test add remote tx")
	pool.AddTx(TxR1, false)
	assert.Equal(t, 2, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 1, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 1, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 1, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 1, len(pool.sortedByPriced.all.remotes))
	fmt.Println("test add another local tx")
	pool.AddTx(Tx2, true)
	assert.Equal(t, 2, len(pool.queue.accTxs))
	assert.Equal(t, 2, pool.queue.accTxs[From1].txs.Len())
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 2, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 1, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 2, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 1, len(pool.sortedByPriced.all.remotes))
	fmt.Println("test add another remote tx")
	pool.AddTx(TxR2, false)
	assert.Equal(t, 2, len(pool.queue.accTxs))
	assert.Equal(t, 2, pool.queue.accTxs[From1].txs.Len())
	assert.Equal(t, 2, pool.queue.accTxs[From2].txs.Len())
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 2, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 2, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 2, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 2, len(pool.sortedByPriced.all.remotes))
	fmt.Println("test add same local tx")
	pool.AddTx(Tx2, true)
	assert.Equal(t, 2, len(pool.queue.accTxs))
	assert.Equal(t, 2, pool.queue.accTxs[From1].txs.Len())
	assert.Equal(t, 2, pool.queue.accTxs[From2].txs.Len())
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 2, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 2, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 2, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 2, len(pool.sortedByPriced.all.remotes))

}
func TestTransactionPool_RemoveTxByKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	pool.AddTx(Tx1, true)
	fmt.Println("remove local tx by key")
	pool.RemoveTxByKey(Key1)
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	pool.AddTx(TxR1, false)
	fmt.Println("remove remote tx by key")
	pool.RemoveTxByKey(KeyR1)
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
}

func TestTransactionPool_RemoveTxHashs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	assert.Equal(t, 2, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 2, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 2, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 2, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 2, len(pool.sortedByPriced.all.remotes))
	var hashs []string
	hashs = append(hashs, Key1)
	hashs = append(hashs, Key2)
	hashs = append(hashs, KeyR1)
	hashs = append(hashs, KeyR2)
	pool.RemoveTxHashs(hashs)
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))

}
func TestTransactionPool_turnTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	ok := pool.turnTx(From1, Key1, Tx1)
	if !ok {
		fmt.Println("tx1 turn error")
	}

	ok = pool.turnTx(From1, Key2, Tx2)
	if !ok {
		fmt.Println("tx2 turn error")
	}
	ok = pool.turnTx(From2, KeyR1, TxR1)
	if !ok {
		fmt.Println("txr1 turn error")
	}
	ok = pool.turnTx(From2, KeyR2, TxR2)
	if !ok {
		fmt.Println("txr1 turn error")
	}
	fmt.Println("test for turn tx from queue to pending")
	assert.Equal(t, 0, pool.queue.accTxs[From1].txs.Len())
	assert.Equal(t, 2, pool.pending.accTxs[From1].txs.Len())
	assert.Equal(t, 2, pool.pending.accTxs[From2].txs.Len())
	assert.Equal(t, 2, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 2, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 2, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 2, len(pool.sortedByPriced.all.remotes))

}

func TestTransactionPool_UpdateTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)

	ok := pool.turnTx(From1, Key1, Tx1)
	if !ok {
		fmt.Println("turn error")
	}

	fmt.Println("test for update in pending tx failed")
	assert.Equal(t, 1, pool.queue.accTxs[From1].txs.Len())
	assert.Equal(t, 1, pool.pending.accTxs[From1].txs.Len())
	assert.Equal(t, 2, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 2, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))

	pool.UpdateTx(Tx3, Key1)
	for _, tx := range pool.pending.accTxs[From1].txs.items {
		txid, _ := tx.TxID()
		assert.Equal(t, txid, Key1)
	}

	fmt.Println("test for update  in queue tx done")
	pool.UpdateTx(Tx4, Key2)
	for _, tx := range pool.queue.accTxs[From1].txs.items {
		txid, _ := tx.TxID()
		assert.Equal(t, txid, Key4)
	}
}

func TestTransactionPool_Pending(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(Tx4, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	pool.turnTx(From1, Key1, Tx1)
	pending := pool.Pending()
	for _, txs := range pending {
		for _, tx := range txs {
			if !reflect.DeepEqual(tx, Tx1) {
				t.Error("want", Tx1, "got", tx)
			}
		}
	}
}
func TestTransactionPool_CommitTxsByPriceAndNonce(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)

	ok := pool.turnTx(From1, Key1, Tx1)
	if !ok {
		fmt.Println("tx1 turn error")
	}
	ok = pool.turnTx(From1, Key2, Tx2)
	if !ok {
		fmt.Println("tx2 turn error")
	}
	ok = pool.turnTx(From2, KeyR1, TxR1)
	if !ok {
		fmt.Println("txr1 turn error")
	}
	ok = pool.turnTx(From2, KeyR2, TxR2)
	if !ok {
		fmt.Println("txr2 turn error")
	}
	pending := pool.Pending()

	txs := make([]*transaction.Transaction, 0)
	txSet := transaction.NewTxsByPriceAndNonce(pending)
	var cnt int
	for {
		tx := txSet.Peek()
		if tx == nil {
			break
		}
		txs = append(txs, tx)
		cnt++
		txSet.Pop()
	}

	if got := pool.CommitTxsByPriceAndNonce(); !reflect.DeepEqual(got, txs) {
		t.Errorf("CommitTxsByPriceAndNonce() = %v, want %v", got, txs)
	}

}
func TestTransactionPool_CommitTxsForPending(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	pool.AddTx(Tx1, true)
	pool.turnTx(From1, Key1, Tx1)
	txls := pool.pending.accTxs
	var txs = make([]*transaction.Transaction, 0)
	for _, txlist := range txls {
		for _, tx := range txlist.txs.cache {
			txs = append(txs, tx)
		}
	}

	if got := pool.CommitTxsForPending(); !reflect.DeepEqual(got, txs) {
		t.Errorf("CommitTxsForPending() = %v, want %v", got, txs)
	}

}
func TestTransactionPool_PickTxs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)

	ok := pool.turnTx(From1, Key1, Tx1)
	if !ok {
		fmt.Println("tx1 turn error")
	}
	ok = pool.turnTx(From1, Key2, Tx2)
	if !ok {
		fmt.Println("tx2 turn error")
	}
	ok = pool.turnTx(From2, KeyR1, TxR1)
	if !ok {
		fmt.Println("txr1 turn error")
	}
	ok = pool.turnTx(From2, KeyR2, TxR2)
	if !ok {
		fmt.Println("txr2 turn error")
	}
	pending := pool.Pending()
	txs := make([]*transaction.Transaction, 0)
	for _, v := range pending {
		for _, tx := range v {
			txs = append(txs, tx)
		}
	}
	if got := pool.PickTxs(0); !reflect.DeepEqual(got, txs) {
		t.Errorf("PickTxs(0) = %v, want %v", got, txs)
	}

	txs2 := make([]*transaction.Transaction, 0)
	txSet := transaction.NewTxsByPriceAndNonce(pending)
	for {

		tx := txSet.Peek()
		if tx == nil {
			break
		}
		txs2 = append(txs2, tx)
		txSet.Pop()
	}

	if got := pool.PickTxs(1); !reflect.DeepEqual(got, txs2) {
		t.Errorf("CommitTxsByPriceAndNonce() = %v, want %v", got, txs)
	}
}

func TestTransactionPool_BroadCastTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
}
func TestTransactionPool_SaveConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant

	pool.config.PathRemote = "newremote.json"

	if err := pool.SaveConfig(); err != nil {
		fmt.Println(err)
	}

	data, err := ioutil.ReadFile(pool.config.PathConfig)
	if err != nil {
		fmt.Println(err)
	}
	var conf TransactionPoolConfig
	config := &conf
	err = json.Unmarshal(data, &config)
	if !reflect.DeepEqual(*config, pool.config) {
		t.Error("want", pool.config, "got", *config)
	}
}
func TestTransactionPool_LoadConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant

	pool.config.PathRemote = "newremote.json"
	if err := pool.SaveConfig(); err != nil {
		fmt.Println(err)
	}

	conf, _ := pool.LoadConfig()
	if !reflect.DeepEqual(*conf, pool.config) {
		t.Error("want", pool.config, "got", *conf)
	}
}

func TestTransactionPool_UpdateTxPoolConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant

	pool.config.PathRemote = "newremote.json"
	if err := pool.SaveConfig(); err != nil {
		fmt.Println(err)
	}

	conf, _ := pool.LoadConfig()
	pool.UpdateTxPoolConfig(*conf)
	if !reflect.DeepEqual(*conf, pool.config) {
		t.Error("want", *conf, "got", pool.config)
	}

}

func TestTransactionPool_SaveLocalTxs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)

	if err := pool.SaveLocalTxs(); err != nil {
		fmt.Println(err)
	}
	data, err := ioutil.ReadFile(pool.config.PathLocal)
	if err != nil {
		fmt.Println(err)
	}
	var locals map[string]*transaction.Transaction
	err = json.Unmarshal(data, &locals)
	if err != nil {
		fmt.Println(err)
	}
	if !reflect.DeepEqual(locals, pool.allTxsForLook.locals) {
		t.Error("want", pool.allTxsForLook.locals, "got", locals)
	}

}
func TestTransactionPool_LoadLocalTxs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	fmt.Println("add two local txs")
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	//pool.AddTx(TxR1, false)
	//pool.AddTx(TxR2, false)
	rawLocaltxs := pool.allTxsForLook.locals
	if err := pool.SaveLocalTxs(); err != nil {
		fmt.Println(err)
	}
	fmt.Println("init txpool")
	pool = SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	pool.LoadLocalTxs()
	fmt.Println("assert load local txs")
	if !assert.Equal(t, pool.queue.accTxs[From1].txs.Len(), 2) ||
		!assert.Equal(t, len(pool.allTxsForLook.locals), 2) {
		t.Error("want", rawLocaltxs, "got", pool.allTxsForLook.locals)
	}

}
func TestTransactionPool_SaveRemoteTxs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)

	if err := pool.SaveRemoteTxs(); err != nil {
		fmt.Println(err)
	}
	data, err := ioutil.ReadFile(pool.config.PathRemote)
	if err != nil {
		fmt.Println(err)
	}
	remotetxs := &remoteTxs{}
	err = json.Unmarshal(data, &remotetxs)
	if err != nil {
		fmt.Println(err)
	}
	for k, v := range remotetxs.Txs {
		if !reflect.DeepEqual(pool.allTxsForLook.remotes[k], v) {
			fmt.Println(pool.allTxsForLook.remotes[k])
			fmt.Println(v)
			t.Error("want", pool.allTxsForLook.remotes[k], "got", v)
		}
	}

}
func TestTransactionPool_LoadRemoteTxs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)

	want := pool.allTxsForLook.remotes
	if err := pool.SaveRemoteTxs(); err != nil {
		fmt.Println(err)
	}
	fmt.Println("init txpool")
	pool = SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	if err := pool.LoadRemoteTxs(); err != nil {
		fmt.Println(err)
	}
	if got := pool.allTxsForLook.remotes; !reflect.DeepEqual(got, want) {
		fmt.Println(got)
		fmt.Println(want)
		t.Error("want", want, "got")
	}

}

func TestTransactionPool_Cost(t *testing.T) {
	fmt.Println("Waiting for test")
}
func TestTransactionPool_Gas(t *testing.T) {
	fmt.Println("Waiting for test")
}
func TestTransactionPool_Get(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)

	want := Tx1
	if got := pool.Get(Key1); !reflect.DeepEqual(want, got) {
		t.Error("want", want, "got", got)
	}
	want = Tx3
	if got := pool.Get(Key3); !reflect.DeepEqual(want, got) {
		t.Error("want", want, "got", got)
	}

}
func TestTransactionPool_Size(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	want := 4
	if got := pool.allTxsForLook.Count(); want != got {
		t.Error("want", want, "got", got)
	}
}
func TestTransactionPool_Dispatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
}
func TestTransactionPool_Reset(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
}
func TestTransactionPool_Start(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
}
func TestTransactionPool_Stop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
}
