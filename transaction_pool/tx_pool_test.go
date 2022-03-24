package transactionpool

import (
	"encoding/hex"
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
	"testing"
	"time"
)

var (
	TestTxPoolConfig                                                    TransactionPoolConfig
	Tx1, Tx2, Tx3, Tx4, TxR1, TxR2, TxR3, TxlowGasPrice, TxHighGasLimit *transaction.Transaction
	TestBlock                                                           *types.Block
	TestBlockHead                                                       *types.BlockHead
	TestBlockData                                                       *types.BlockData
	TestBlockHash                                                       string
)

func init() {
	TestTxPoolConfig = DefaultTransactionPoolConfig

	Tx1 = settransactionlocal(1, 10000, 12345)
	Tx2 = settransactionlocal(2, 20000, 12345)
	Tx3 = settransactionlocal(3, 9999, 12345)
	Tx4 = settransactionlocal(4, 40000, 12345)

	TxR1 = settransactionremote(1, 20000, 12345)
	TxR2 = settransactionremote(2, 3000, 12345)
	TxR3 = settransactionremote(3, 40000, 12345)
	TxlowGasPrice = settransactionremote(1, 100, 1000)
	TxHighGasLimit = settransactionremote(2, 10000, 9987654321)

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
		chain: nil,
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

func TestInValidateTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	TestTxPoolConfig.chain = servant
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	fmt.Println("test for ErrTxGasLimit:")
	if err := pool.ValidateTx(TxHighGasLimit, true); err != ErrTxGasLimit {
		t.Error("expected", ErrTxGasLimit, "got", err)
	}
	fmt.Println("test for ErrGasPriceTooLow:")
	if err := pool.ValidateTx(TxlowGasPrice, false); err != ErrGasPriceTooLow {
		t.Error("expected", ErrGasPriceTooLow, "got", err)
	}
}

func TestTransactionPool(t *testing.T) {
	from1 := account.Address(hex.EncodeToString(Tx1.FromAddr))
	txid1, _ := Tx1.TxID()
	txid2, _ := Tx2.TxID()
	//txid3, _ := Tx3.TxID()
	txid4, _ := Tx4.TxID()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	TestTxPoolConfig.chain = servant
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))

	fmt.Println("txPool init queue,pending,allTxs,sortedByPrice are all zero")
	assert.True(t, len(pool.queue.accTxs) == 0, "len of queue is 0")
	assert.True(t, len(pool.pending.accTxs) == 0, "len of pending is 0")
	assert.True(t, pool.allTxsForLook.LocalCount() == 0, "count of local in  allTxsForLook is 0")
	assert.True(t, pool.allTxsForLook.RemoteCount() == 0, "count of remote in allTxsForLook is 0")
	assert.True(t, len(pool.sortedByPriced.all.locals) == 0, "count of local in sortedByPriced is 0")
	assert.True(t, len(pool.sortedByPriced.all.remotes) == 0, "count of remote in sortedByPriced is 0")

	fmt.Println("test for add one local tx")
	pool.queueAddTx(txid1, Tx1, true, true)
	txlist := pool.queue.accTxs[from1]
	assert.True(t, txlist.Len() == 1, "len txs of queue is 1")
	assert.True(t, len(pool.pending.accTxs) == 0, "len of pending is 0")
	assert.True(t, pool.allTxsForLook.LocalCount() == 1, "count of local in  allTxsForLook is 1")
	assert.True(t, pool.allTxsForLook.RemoteCount() == 0, "count of remote in allTxsForLook is 0")
	assert.True(t, len(pool.sortedByPriced.all.locals) == 1, "count of local in sortedByPriced is 1")
	assert.True(t, len(pool.sortedByPriced.all.remotes) == 0, "count of remote in sortedByPriced is 0")

	fmt.Println("test for add remote tx ")
	pool.queueAddTx(txid2, Tx2, false, true)
	assert.True(t, txlist.Len() == 2, "len txs of queue is 2")
	assert.True(t, len(pool.pending.accTxs) == 0, "len of pending is 0")
	assert.True(t, pool.allTxsForLook.LocalCount() == 1, "count of local in  allTxsForLook is 1")
	assert.True(t, pool.allTxsForLook.RemoteCount() == 1, "count of remote in allTxsForLook is 1")
	assert.True(t, len(pool.sortedByPriced.all.locals) == 1, "count of local in sortedByPriced is 1")
	assert.True(t, len(pool.sortedByPriced.all.remotes) == 1, "count of remote in sortedByPriced is 1")

	fmt.Println("test for turn tx from queue to pending")
	ok := pool.turnTx(from1, txid1, Tx1)
	if !ok {
		fmt.Println("turn error")
	}
	txlist = pool.queue.accTxs[from1]
	assert.True(t, txlist.Len() == 1, "len txs of queue is 1")
	assert.True(t, len(pool.pending.accTxs) == 1, "len of pending is 0")
	assert.True(t, pool.allTxsForLook.LocalCount() == 1, "count of local in  allTxsForLook is 1")
	assert.True(t, pool.allTxsForLook.RemoteCount() == 1, "count of remote in allTxsForLook is 1")
	assert.True(t, len(pool.sortedByPriced.all.locals) == 1, "count of local in sortedByPriced is 1")
	assert.True(t, len(pool.sortedByPriced.all.remotes) == 1, "count of remote in sortedByPriced is 1")

	fmt.Println("test for remove tx by key from pending")
	err := pool.RemoveTxByKey(txid1)
	if err != nil {
		fmt.Println(err)
	}
	txlist = pool.queue.accTxs[from1]
	assert.True(t, txlist.Len() == 1, "len txs of queue is 1")
	assert.True(t, len(pool.pending.accTxs) == 0, "len of pending is 0")
	assert.True(t, pool.allTxsForLook.LocalCount() == 0, "count of local in  allTxsForLook is 0")
	assert.True(t, pool.allTxsForLook.RemoteCount() == 1, "count of remote in allTxsForLook is 1")
	assert.True(t, len(pool.sortedByPriced.all.locals) == 0, "count of local in sortedByPriced is 0")
	assert.True(t, len(pool.sortedByPriced.all.remotes) == 1, "count of remote in sortedByPriced is 1")
	fmt.Println("test for remove tx by key from queue")
	err = pool.RemoveTxByKey(txid2)
	if err != nil {
		fmt.Println(err)
	}
	assert.True(t, txlist.Len() == 0, "len txs of queue is 0")
	assert.True(t, len(pool.pending.accTxs) == 0, "len of pending is 0")
	assert.True(t, pool.allTxsForLook.LocalCount() == 0, "count of local in  allTxsForLook is 0")
	assert.True(t, pool.allTxsForLook.RemoteCount() == 0, "count of remote in allTxsForLook is 0")
	assert.True(t, len(pool.sortedByPriced.all.locals) == 0, "count of local in sortedByPriced is 0")
	assert.True(t, len(pool.sortedByPriced.all.remotes) == 0, "count of remote in sortedByPriced is 0")

	pool.queueAddTx(txid1, Tx1, true, true)
	pool.queueAddTx(txid2, Tx2, false, true)
	ok = pool.turnTx(from1, txid1, Tx1)
	if !ok {
		fmt.Println("turn error")
	}
	fmt.Println("test for update in queue tx done")
	txlist = pool.queue.accTxs[from1]
	assert.True(t, txlist.Len() == 1, "len txs of queue is 1")
	assert.True(t, len(pool.pending.accTxs) == 1, "len of pending is 0")
	assert.True(t, pool.allTxsForLook.LocalCount() == 1, "count of local in  allTxsForLook is 1")
	assert.True(t, pool.allTxsForLook.RemoteCount() == 1, "count of remote in allTxsForLook is 1")
	assert.True(t, len(pool.sortedByPriced.all.locals) == 1, "count of local in sortedByPriced is 1")
	assert.True(t, len(pool.sortedByPriced.all.remotes) == 1, "count of remote in sortedByPriced is 1")

	txlist = pool.pending.accTxs[from1]
	pool.UpdateTx(Tx4, txid1)
	for _, tx := range txlist.txs.items {
		txid, _ := tx.TxID()
		assert.True(t, txid == txid4)
	}
	fmt.Println("test for update  in queue tx done")
	pool.UpdateTx(Tx4, txid2)
	txlist = pool.queue.accTxs[from1]
	for _, tx := range txlist.txs.items {
		txid, _ := tx.TxID()
		assert.True(t, txid == txid4)
	}
	fmt.Println("test for update in queue tx failed")
	pool.UpdateTx(Tx3, txid2)
	for _, tx := range txlist.txs.items {
		txid, _ := tx.TxID()
		assert.True(t, txid == txid2)
	}

	if len(pool.queue.accTxs) != 1 {
		t.Error("except len of queue is 0,got", len(pool.queue.accTxs))
	}

}

func TestUpdateTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	TestTxPoolConfig.chain = servant
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.add(Tx1, true)
	var key string
	for k, v := range pool.allTxsForLook.locals {
		fmt.Println(k, v)
		key = k
	}
	fmt.Println("gasprice lower, no update ")
	pool.UpdateTx(Tx2, key)
	for k, _ := range pool.allTxsForLook.locals {
		if k != key {
			t.Error("want", key, "got", k)
		}
	}

	fmt.Println("gasprice higher ,updated ")
	key3, _ := Tx3.TxID()
	pool.UpdateTx(Tx3, key)
	for k, _ := range pool.allTxsForLook.locals {
		if k != key3 {
			t.Error("want", key3, "got", k)
		}
	}
}
