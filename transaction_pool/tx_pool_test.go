package transactionpool

import (
	"fmt"
	"github.com/TopiaNetwork/topia/account"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/common/types"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"github.com/TopiaNetwork/topia/transaction"
	"github.com/golang/mock/gomock"
	"testing"
	"time"
)

var (
	testTxPoolConfig   TransactionPoolConfig
	Tx1, Tx2, Tx3, Tx4 *transaction.Transaction
	TestBlock          *types.Block
	TestBlockHead      *types.BlockHead
	TestBlockData      *types.BlockData
	TestBlockHash      string
)

func init() {
	testTxPoolConfig = DefaultTransactionPoolConfig
	Tx1 = settransaction(1)
	TestBlockHash = "0x224111a2131c213b213112d121c1231e"
	TestBlockHead = &types.BlockHead{
		ChainID:              []byte{0x01},
		Version:              0,
		Height:               0,
		Epoch:                0,
		Round:                0,
		ParentBlockHash:      nil,
		ProposeSignature:     nil,
		VoteAggSignature:     nil,
		ResultHash:           nil,
		TxCount:              0,
		TxHashRoot:           nil,
		TimeStamp:            0,
		DataHash:             nil,
		Reserved:             nil,
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
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
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}

}

func settransaction(nonce uint64) *transaction.Transaction {
	tx := &transaction.Transaction{
		FromAddr:   []byte{0x21, 0x23, 0x43, 0x53, 0x23, 0x34, 0x21, 0x12, 0x42, 0x12, 0x43},
		TargetAddr: []byte{0x23, 0x34, 0x34, 0x53, 0x23, 0x34, 0x21, 0x12, 0x42, 0x12, 0x43}, Version: 1, ChainID: []byte{0x01},
		Nonce: nonce, Value: []byte{0x12, 0x32}, GasPrice: 10000,
		GasLimit: 100, Data: []byte{0x32, 0x32, 0x32},
		Signature: []byte{0x32, 0x23, 0x42}, Options: 0, Time: time.Now()}
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
	//pool.Reset(nil, conf.chain.CurrentBlock().GetHead())
	//
	//pool.wg.Add(1)
	//go pool.scheduleReorgLoop()
	//
	//pool.loadLocal(conf.NoLocalFile, conf.PathLocal)
	//pool.loadRemote(conf.NoRemoteFile, conf.PathRemote)
	//pool.loadConfig(conf.NoConfigFile, conf.PathConfig)
	//
	//pool.pubSubService = pool.config.chain.SubChainHeadEvent(pool.chanChainHead)
	//
	//go pool.loop()
	return pool
}

func TestInValidateTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	testTxPoolConfig.chain = servant
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(testTxPoolConfig, 1, log, codec.CodecType(1))
	tx := Tx1
	if err := pool.ValidateTx(tx, true); err != ErrTxGasLimit {
		t.Error("expected", ErrTxGasLimit, "got", err)
	}
}

func TestTransactionQueue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	//cost := big.NewInt(1234)
	//servant.EXPECT().EstimateTxCost(gomock.Any()).Return(cost)
	testTxPoolConfig.chain = servant
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(testTxPoolConfig, 1, log, codec.CodecType(1))
	fmt.Println("len of queue is 0:", len(pool.queue.accTxs))
	txid1, _ := Tx1.TxID()
	pool.queueAddTx(txid1, Tx1, true, true)
	if len(pool.queue.accTxs) != 1 {
		t.Error("except len of queue is 1,got", len(pool.queue.accTxs))
	}
}

func TestTransactionPending(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	testTxPoolConfig.chain = servant
	log := NewMockLogger(ctrl)
	pool := SetNewTransactionPool(testTxPoolConfig, 1, log, codec.CodecType(1))
	ok, err := pool.add(Tx1, true)
	if !ok {
		fmt.Println(err)
	}
}
