package transactionpool

import (
	"encoding/hex"
	"encoding/json"
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/TopiaNetwork/topia/account"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/common/types"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"github.com/TopiaNetwork/topia/transaction"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"math/big"
	"reflect"
	"testing"
	"time"
)

var (
	TestTxPoolConfig                                                    TransactionPoolConfig
	Tx1, Tx2, Tx3, Tx4, TxR1, TxR2, TxR3, TxlowGasPrice, TxHighGasLimit *transaction.Transaction
	Key1, Key2, Key3, Key4, KeyR1, KeyR2, KeyR3                         string
	From1, From2                                                        account.Address
	OldBlock, NewBlock                                                  *types.Block
	OldBlockHead, NewBlockHead                                          *types.BlockHead
	OldBlockData, NewBlockData                                          *types.BlockData
	OldTx1, OldTx2, OldTx3, OldTx4, NewTx1, NewTx2, NewTx3, NewTx4      *transaction.Transaction
	sysActor                                                            *actor.ActorSystem
	newcontext                                                          actor.Context
	State                                                               *StatePoolDB
	TpiaLog                                                             tplog.Logger
)

func init() {
	TpiaLog, _ = tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.DefaultLogFormat, tplog.DefaultLogOutput, "")

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
	OldBlockHead = setBlockHead(10, 5, 1, 4, 1000000)
	OldBlockHead.ParentBlockHash = []byte{0x09}
	OldBlockHead.DataHash = []byte{0x0a, 0x0a}
	OldBlockHead.TxHashRoot = []byte{0x0a, 0x0a}
	NewBlockHead = setBlockHead(11, 5, 2, 5, 2000000)
	NewBlockHead.ParentBlockHash = []byte{0x0a, 0x0a}
	NewBlockHead.DataHash = []byte{0x0b, 0x0b}
	NewBlockHead.TxHashRoot = []byte{0x0b, 0x0b}

	OldTx1 = setBlockTransaction([]byte{0xa1}, []byte{0xa2}, 10, 10000, 1000)
	OldTx2 = setBlockTransaction([]byte{0xa2}, []byte{0xa3}, 10, 10000, 1000)
	OldTx3 = setBlockTransaction([]byte{0xa3}, []byte{0xa4}, 10, 10000, 1000)
	OldTx4 = setBlockTransaction([]byte{0xa4}, []byte{0xa5}, 10, 10000, 1000)
	OldKey1, _ := OldTx1.TxID()
	OldKey2, _ := OldTx2.TxID()
	OldKey3, _ := OldTx3.TxID()
	OldKey4, _ := OldTx4.TxID()
	OldTxs := make(map[string]*transaction.Transaction, 0)
	OldTxs[OldKey1] = OldTx1
	OldTxs[OldKey2] = OldTx2
	OldTxs[OldKey3] = OldTx3
	OldTxs[OldKey4] = OldTx4

	NewTx1 = setBlockTransaction([]byte{0xb1}, []byte{0xb2}, 11, 10000, 1000)
	NewTx2 = setBlockTransaction([]byte{0xb2}, []byte{0xb3}, 12, 10000, 1000)
	NewTx3 = setBlockTransaction([]byte{0xb3}, []byte{0xb4}, 13, 10000, 1000)
	NewTx4 = setBlockTransaction([]byte{0xb4}, []byte{0xb5}, 14, 10000, 1000)
	NewKey1, _ := NewTx1.TxID()
	NewKey2, _ := NewTx2.TxID()
	NewKey3, _ := NewTx3.TxID()
	NewKey4, _ := NewTx4.TxID()

	NewTxs := make(map[string]*transaction.Transaction, 0)
	NewTxs[NewKey1] = NewTx1
	NewTxs[NewKey2] = NewTx2
	NewTxs[NewKey3] = NewTx3
	NewTxs[NewKey4] = NewTx4

	OldBlockData = SetBlockData(OldTxs)
	NewBlockData = SetBlockData(NewTxs)
	OldBlock = SetBlock(OldBlockHead, OldBlockData)
	NewBlock = SetBlock(NewBlockHead, NewBlockData)

	sysActor = actor.NewActorSystem()
	State = &StatePoolDB{}

}

func settransactionlocal(nonce uint64, gaseprice, gaseLimit uint64) *transaction.Transaction {
	tx := &transaction.Transaction{
		FromAddr:   []byte{0x01},
		TargetAddr: []byte{0x02}, Version: 1, ChainID: []byte{0x01},
		Nonce: nonce, Value: []byte{0x03}, GasPrice: gaseprice,
		GasLimit: gaseLimit, Data: []byte{0xaa, 0xbb, 0xbc},
		Signature: []byte{0x05}, Options: 0, Time: time.Now()}
	return tx
}
func settransactionremote(nonce uint64, gaseprice, gaseLimit uint64) *transaction.Transaction {
	tx := &transaction.Transaction{
		FromAddr:   []byte{0x11},
		TargetAddr: []byte{0x12}, Version: 1, ChainID: []byte{0x01},
		Nonce: nonce, Value: []byte{0x13}, GasPrice: gaseprice,
		GasLimit: gaseLimit, Data: []byte{0xcc, 0xdd, 0xde},
		Signature: []byte{0x15}, Options: 0, Time: time.Now()}
	return tx
}
func setBlockHead(height, epoch, round uint64, txcount uint32, timestamp uint64) *types.BlockHead {
	blockhead := &types.BlockHead{
		ChainID:          []byte{0x01},
		Version:          1,
		Height:           height,
		Epoch:            epoch,
		Round:            round,
		ParentBlockHash:  nil,
		ProposeSignature: nil,
		VoteAggSignature: nil,
		ResultHash:       nil,
		TxCount:          txcount,
		TxHashRoot:       nil,
		TimeStamp:        timestamp,
		DataHash:         nil,
		Reserved:         nil,
	}
	return blockhead
}
func SetBlock(head *types.BlockHead, data *types.BlockData) *types.Block {
	block := &types.Block{
		Head: head,
		Data: data,
	}
	return block
}
func SetBlockData(txs map[string]*transaction.Transaction) *types.BlockData {
	blockdata := &types.BlockData{
		Version: 1,
		Txs:     nil,
	}
	txsByte := make([][]byte, 0)
	for _, tx := range txs {
		txByte, _ := json.Marshal(tx)
		txsByte = append(txsByte, txByte)
	}
	blockdata.Txs = txsByte
	return blockdata
}

func setBlockTransaction(from, to []byte, nonce uint64, gaseprice, gaseLimit uint64) *transaction.Transaction {
	tx := &transaction.Transaction{
		FromAddr:   from,
		TargetAddr: to, Version: 1, ChainID: []byte{0x01},
		Nonce: nonce, Value: []byte{0x03}, GasPrice: gaseprice,
		GasLimit: gaseLimit, Data: []byte{0x04},
		Signature: []byte{0x05}, Options: 0, Time: time.Now()}
	return tx
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

		chanSysShutDown:   make(chan error),
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

	//pool.Reset(nil, pool.query.CurrentBlock().GetHead())
	//
	//pool.wg.Add(1)
	//go pool.scheduleReorgLoop()
	//
	//pool.loadLocal(conf.NoLocalFile, conf.PathLocal)
	//pool.loadRemote(conf.NoRemoteFile, conf.PathRemote)
	//pool.loadConfig(conf.NoConfigFile, conf.PathConfig)

	//pool.pubSubService = pool.query.SubChainHeadEvent(pool.chanChainHead)
	//pool.wg.Add(1)
	//go pool.loop()
	return pool
}

func Test_transactionPool_ValidateTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	if err := pool.ValidateTx(TxHighGasLimit, true); err != ErrTxGasLimit {
		t.Error("expected", ErrTxGasLimit, "got", err)
	}
	if err := pool.ValidateTx(TxlowGasPrice, false); err != ErrGasPriceTooLow {
		t.Error("expected", ErrGasPriceTooLow, "got", err)
	}
}

func Test_transactionPool_GetLocalTxs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
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
	want := txsMap
	got := pool.GetLocalTxs()
	if !reflect.DeepEqual(want, got) {
		t.Error("want:", want, "got:", got)
	}
}

func Test_transactionPool_AddTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	pool.AddTx(Tx1, true)
	assert.Equal(t, 1, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 1, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 1, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	pool.AddTx(TxR1, false)
	assert.Equal(t, 2, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 1, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 1, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 1, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 1, len(pool.sortedByPriced.all.remotes))
	pool.AddTx(Tx2, true)
	assert.Equal(t, 2, len(pool.queue.accTxs))
	assert.Equal(t, 2, pool.queue.accTxs[From1].txs.Len())
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 2, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 1, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 2, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 1, len(pool.sortedByPriced.all.remotes))
	pool.AddTx(TxR2, false)
	assert.Equal(t, 2, len(pool.queue.accTxs))
	assert.Equal(t, 2, pool.queue.accTxs[From1].txs.Len())
	assert.Equal(t, 2, pool.queue.accTxs[From2].txs.Len())
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 2, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 2, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 2, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 2, len(pool.sortedByPriced.all.remotes))
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
func Test_transactionPool_RemoveTxByKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	pool.AddTx(Tx1, true)
	pool.RemoveTxByKey(Key1)
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	pool.AddTx(TxR1, false)
	pool.RemoveTxByKey(KeyR1)
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
}

func Test_transactionPool_RemoveTxHashs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
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
func Test_transactionPool_turnTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
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
	_ = pool.turnTx(From1, Key1, Tx1)
	_ = pool.turnTx(From1, Key2, Tx2)
	_ = pool.turnTx(From2, KeyR1, TxR1)
	_ = pool.turnTx(From2, KeyR2, TxR2)
	assert.Equal(t, 0, pool.queue.accTxs[From1].txs.Len())
	assert.Equal(t, 2, pool.pending.accTxs[From1].txs.Len())
	assert.Equal(t, 2, pool.pending.accTxs[From2].txs.Len())
	assert.Equal(t, 2, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 2, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 2, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 2, len(pool.sortedByPriced.all.remotes))

}

func Test_transactionPool_UpdateTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	_ = pool.turnTx(From1, Key1, Tx1)
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

	pool.UpdateTx(Tx4, Key2)
	for _, tx := range pool.queue.accTxs[From1].txs.items {
		txid, _ := tx.TxID()
		assert.Equal(t, txid, Key4)
	}
}

func Test_transactionPool_Pending(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
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
func Test_transactionPool_CommitTxsByPriceAndNonce(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
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

	_ = pool.turnTx(From1, Key1, Tx1)
	_ = pool.turnTx(From1, Key2, Tx2)
	_ = pool.turnTx(From2, KeyR1, TxR1)
	_ = pool.turnTx(From2, KeyR2, TxR2)
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
func Test_transactionPool_CommitTxsForPending(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
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
func Test_transactionPool_PickTxs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
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

	_ = pool.turnTx(From1, Key1, Tx1)
	_ = pool.turnTx(From1, Key2, Tx2)
	_ = pool.turnTx(From2, KeyR1, TxR1)
	_ = pool.turnTx(From2, KeyR2, TxR2)
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

func Test_transactionPool_Cost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	servant.EXPECT().EstimateTxCost(gomock.Any()).Return(big.NewInt(90000000000)).AnyTimes()
	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	want := big.NewInt(90000000000)
	if got := pool.Cost(Tx1); !reflect.DeepEqual(want, got) {
		t.Error("want", want, "got", got)
	}

}
func Test_transactionPool_Gas(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	servant.EXPECT().EstimateTxGas(gomock.Any()).Return(uint64(1234567)).AnyTimes()
	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	want := uint64(1234567)
	if got := pool.Gas(Tx1); !reflect.DeepEqual(want, got) {
		t.Error("want", want, "got", got)
	}

}
func Test_transactionPool_Get(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
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
func Test_transactionPool_Size(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
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

func Test_transactionPool_Start(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	network := NewMockNetwork(ctrl)
	network.EXPECT().RegisterModule(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	if err := pool.Start(sysActor, network); err != nil {
		t.Error("want", nil, "got", err)
	}

}
func Test_transactionPool_Stop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	pool.Stop()
	if 1 != 1 {
		t.Error("stop error")
	}

}
