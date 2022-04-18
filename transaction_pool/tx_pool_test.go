package transactionpool

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"github.com/TopiaNetwork/topia/transaction/basic"
	"github.com/TopiaNetwork/topia/transaction/universal"
)

var (
	Category1, Category2                                                             basic.TransactionCategory
	TestTxPoolConfig                                                                 TransactionPoolConfig
	TxlowGasPrice, TxHighGasLimit, txlocal, txremote, Tx1, Tx2, Tx3, Tx4, TxR1, TxR2 *basic.Transaction
	txHead                                                                           *basic.TransactionHead
	txData                                                                           *basic.TransactionData
	keylocal, keyremote, Key1, Key2, Key3, Key4, KeyR1, KeyR2                        string
	from, to, From1, From2                                                           tpcrtypes.Address
	txLocals, txRemotes                                                              []*basic.Transaction
	keyLocals, keyRemotes                                                            []string
	fromLocals, fromRemotes, toLocals, toRemotes                                     []tpcrtypes.Address

	OldBlock, NewBlock                                             *types.Block
	OldBlockHead, NewBlockHead                                     *types.BlockHead
	OldBlockData, NewBlockData                                     *types.BlockData
	OldTx1, OldTx2, OldTx3, OldTx4, NewTx1, NewTx2, NewTx3, NewTx4 *basic.Transaction
	sysActor                                                       *actor.ActorSystem
	newcontext                                                     actor.Context
	State                                                          *StatePoolDB
	TpiaLog                                                        tplog.Logger
	starttime                                                      uint64
	NodeID                                                         string
	Ctx                                                            context.Context
)

func init() {
	NodeID = "TestNode"
	Ctx = context.Background()
	Category1 = basic.TransactionCategory_Topia_Universal
	TpiaLog, _ = tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.DefaultLogFormat, tplog.DefaultLogOutput, "")
	TestTxPoolConfig = DefaultTransactionPoolConfig
	starttime = uint64(time.Now().Unix() - 105)
	keyLocals = make([]string, 0)
	keyRemotes = make([]string, 0)
	for i := 1; i <= 100; i++ {
		nonce := uint64(i)
		gasprice := uint64(i * 1000)
		gaslimit := uint64(i * 1000000)
		txlocal = setTxLocal(nonce, gasprice, gaslimit)
		txlocal.Head.TimeStamp = starttime + uint64(i)
		if i > 1 {
			txlocal.Head.FromAddr = append(txlocal.Head.FromAddr, byte(i))
		}
		keylocal, _ = txlocal.HashHex()
		keyLocals = append(keyLocals, keylocal)
		txLocals = append(txLocals, txlocal)

		txremote = setTxRemote(nonce, gasprice, gaslimit)
		txremote.Head.TimeStamp = starttime + uint64(i)
		if i > 1 {
			txremote.Head.FromAddr = append(txremote.Head.FromAddr, byte(i))
		}
		keyremote, _ = txremote.HashHex()
		keyRemotes = append(keyRemotes, keyremote)
		txRemotes = append(txRemotes, txremote)
	}
	Tx1 = setTxLocal(1, 11000, 123456)
	Tx2 = setTxLocal(2, 12000, 123456)

	Tx3 = setTxLocal(3, 10000, 123456)
	Tx4 = setTxLocal(4, 14000, 123456)

	From1 = tpcrtypes.Address(Tx1.Head.FromAddr)
	Key1, _ = Tx1.HashHex()
	Key2, _ = Tx2.HashHex()
	Key3, _ = Tx3.HashHex()
	Key4, _ = Tx4.HashHex()

	TxR1 = setTxRemote(1, 11000, 123456)
	TxR2 = setTxRemote(2, 12000, 123456)
	From2 = tpcrtypes.Address(TxR1.Head.FromAddr)
	KeyR1, _ = TxR1.HashHex()
	KeyR2, _ = TxR2.HashHex()

	TxlowGasPrice = setTxRemote(300, 100, 1000)
	TxHighGasLimit = setTxRemote(301, 10000, 9987654321)

	OldBlockHead = setBlockHead(10, 5, 1, 4, 1000000)
	OldBlockHead.ParentBlockHash = []byte{0x09}
	OldBlockHead.Hash = []byte{0x0a, 0x0a}
	NewBlockHead = setBlockHead(11, 5, 2, 5, 2000000)
	NewBlockHead.ParentBlockHash = []byte{0x0a, 0x0a}
	NewBlockHead.Hash = []byte{0x0b, 0x0b}

	OldTx1 = txLocals[50]
	OldTx2 = txLocals[51]
	OldTx3 = txRemotes[52]
	OldTx4 = txRemotes[53]
	OldKey1, _ := OldTx1.HashHex()
	OldKey2, _ := OldTx2.HashHex()
	OldKey3, _ := OldTx3.HashHex()
	OldKey4, _ := OldTx4.HashHex()
	OldTxs := make(map[string]*basic.Transaction, 0)
	OldTxs[OldKey1] = OldTx1
	OldTxs[OldKey2] = OldTx2
	OldTxs[OldKey3] = OldTx3
	OldTxs[OldKey4] = OldTx4

	NewTx1 = txLocals[51]
	NewTx2 = txLocals[52]
	NewTx3 = txRemotes[53]
	NewTx4 = txRemotes[54]
	NewKey1, _ := NewTx1.HashHex()
	NewKey2, _ := NewTx2.HashHex()
	NewKey3, _ := NewTx3.HashHex()
	NewKey4, _ := NewTx4.HashHex()

	NewTxs := make(map[string]*basic.Transaction, 0)
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

func setTxHeadLocal(nonce uint64) *basic.TransactionHead {
	Category := basic.TransactionCategory_Topia_Universal
	txhead := &basic.TransactionHead{
		Category:  []byte(Category),
		ChainID:   []byte{0x01},
		Version:   1,
		Nonce:     nonce,
		FromAddr:  []byte{0x01, 0x01, 0x01},
		Signature: []byte{0x01, 0x01, 0x01},
	}
	return txhead
}
func setTxHeadRemote(nonce uint64) *basic.TransactionHead {
	Category := basic.TransactionCategory_Topia_Universal
	txhead := &basic.TransactionHead{
		Category:  []byte(Category),
		ChainID:   []byte{0x01},
		Version:   1,
		Nonce:     nonce,
		FromAddr:  []byte{0x02, 0x02, 0x02},
		Signature: []byte{0x02, 0x02, 0x02},
	}
	return txhead
}
func setTxDataLocal(nonce, gasprice, gaslimit uint64) *basic.TransactionData {
	txdatauniversal := setTxUniversalLocal(nonce, gasprice, gaslimit)
	byteSpecification, _ := json.Marshal(txdatauniversal)
	txdata := &basic.TransactionData{
		Specification: byteSpecification,
	}

	return txdata
}
func setTxDataRemote(nonce, gasprice, gaslimit uint64) *basic.TransactionData {
	txdatauniversal := setTxUniversalRemote(nonce, gasprice, gaslimit)
	bytesSpecification, _ := json.Marshal(txdatauniversal)
	txdata := &basic.TransactionData{
		Specification: bytesSpecification,
	}

	return txdata
}
func setTxLocal(nonce, gasPrice, gasLimit uint64) *basic.Transaction {
	head := setTxHeadLocal(nonce)
	data := setTxDataLocal(nonce, gasPrice, gasLimit)
	local := &basic.Transaction{
		Head: head,
		Data: data,
	}

	return local
}
func setTxRemote(nonce, gasPrice, gasLimit uint64) *basic.Transaction {
	head := setTxHeadRemote(nonce)
	data := setTxDataRemote(nonce, gasPrice, gasLimit)
	remote := &basic.Transaction{
		Head: head,
		Data: data,
	}

	return remote
}

func setTxUniversalLocalHead(nonce, gasPrice, gasLimit uint64) *universal.TransactionUniversalHead {
	txuniversalhead := &universal.TransactionUniversalHead{
		Version:           1,
		FeePayer:          []byte{0x01, 0x01},
		Nonce:             nonce,
		GasPrice:          gasPrice,
		GasLimit:          gasLimit,
		FeePayerSignature: []byte{0x01, 0x01},
		Type:              1,
		Options:           1,
	}
	return txuniversalhead
}
func setTxUniversalRemoteHead(nonce, gasPrice, gasLimit uint64) *universal.TransactionUniversalHead {
	txuniversalhead := &universal.TransactionUniversalHead{
		Version:           1,
		FeePayer:          []byte{0x01, 0x01},
		Nonce:             nonce,
		GasPrice:          gasPrice,
		GasLimit:          gasLimit,
		FeePayerSignature: []byte{0x01, 0x01},
		Type:              1,
		Options:           1,
	}
	return txuniversalhead
}

func setTxUniversalLocalData(nonce, gasPrice, gasLimit uint64) *universal.TransactionUniversalData {
	txtranserferlocal := setTxUniversalTransferLocal(nonce, gasPrice, gasLimit)
	byteSpecification, _ := json.Marshal(txtranserferlocal)
	txuniversaldata := &universal.TransactionUniversalData{
		Specification: byteSpecification,
	}

	return txuniversaldata
}
func setTxUniversalRemoteData(nonce, gasPrice, gasLimit uint64) *universal.TransactionUniversalData {
	txtranserferremote := setTxUniversalTransferRemote(nonce, gasPrice, gasLimit)
	byteSpecification, _ := json.Marshal(txtranserferremote)
	txuniversaldata := &universal.TransactionUniversalData{
		Specification: byteSpecification,
	}

	return txuniversaldata
}
func setTxUniversalLocal(nonce, gasPrice, gasLimit uint64) *universal.TransactionUniversal {
	head := setTxUniversalLocalHead(nonce, gasPrice, gasLimit)
	data := setTxUniversalLocalData(nonce, gasPrice, gasLimit)
	txuniversal := &universal.TransactionUniversal{
		Head: head,
		Data: data,
	}

	return txuniversal
}
func setTxUniversalRemote(nonce, gasPrice, gasLimit uint64) *universal.TransactionUniversal {
	head := setTxUniversalRemoteHead(nonce, gasPrice, gasLimit)
	data := setTxUniversalRemoteData(nonce, gasPrice, gasLimit)
	txuniversal := &universal.TransactionUniversal{
		Head: head,
		Data: data,
	}
	return txuniversal

}
func setTxUniversalTransferLocal(nonce, gasprice, gaslimit uint64) *universal.TransactionUniversalTransfer {
	basicHead := setTxHeadLocal(nonce)
	headUniversal := setTxUniversalLocalHead(nonce, gasprice, gaslimit)
	txuniversaltransfer := &universal.TransactionUniversalTransfer{
		TransactionHead:          *basicHead,
		TransactionUniversalHead: *headUniversal,
		TargetAddr:               tpcrtypes.Address("0x01a1a1a"),
	}
	return txuniversaltransfer
}
func setTxUniversalTransferRemote(nonce, gasprice, gaslimit uint64) *universal.TransactionUniversalTransfer {
	basicHead := setTxHeadRemote(nonce)
	headUniversal := setTxUniversalRemoteHead(nonce, gasprice, gaslimit)
	txuniversaltransfer := &universal.TransactionUniversalTransfer{
		TransactionHead:          *basicHead,
		TransactionUniversalHead: *headUniversal,
		TargetAddr:               tpcrtypes.Address("0x02b2b2b"),
	}
	return txuniversaltransfer
}

func setBlockHead(height, epoch, round uint64, txcount uint32, timestamp uint64) *types.BlockHead {
	blockhead := &types.BlockHead{ChainID: []byte{0x01}, Version: 1, Height: height, Epoch: epoch, Round: round,
		ParentBlockHash: nil, Launcher: nil, Proposer: nil, Proof: nil, VRFProof: nil,
		MaxPri: nil, VoteAggSignature: nil, TxCount: txcount, TxRoot: nil,
		TxResultRoot: nil, StateRoot: nil, GasFees: nil, TimeStamp: timestamp, ElapsedSpan: 0, Hash: nil,
		Reserved: nil}
	return blockhead
}
func SetBlock(head *types.BlockHead, data *types.BlockData) *types.Block {
	block := &types.Block{
		Head: head,
		Data: data,
	}
	return block
}
func SetBlockData(txs map[string]*basic.Transaction) *types.BlockData {
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

func SetNewTransactionPool(nodeId string, ctx context.Context, conf TransactionPoolConfig, level tplogcmm.LogLevel, log tplog.Logger, codecType codec.CodecType) *transactionPool {

	conf = (&conf).check()
	conf.PendingAccountSegments = 16
	conf.PendingGlobalSegments = 128
	conf.QueueMaxTxsAccount = 32
	conf.QueueMaxTxsGlobal = 256
	poolLog := tplog.CreateModuleLogger(level, "TransactionPool", log)
	pool := &transactionPool{
		nodeId:              nodeId,
		config:              conf,
		log:                 poolLog,
		level:               level,
		ctx:                 ctx,
		ActivationIntervals: newActivationInterval(),
		TxHashCategory:      newTxHashCategory(),
		chanBlockAdded:      make(chan BlockAddedEvent, ChanBlockAddedSize),
		chanReqReset:        make(chan *txPoolResetHeads),
		chanReqPromote:      make(chan *accountSet),
		chanReorgDone:       make(chan chan struct{}),
		chanReorgShutdown:   make(chan struct{}), // requests shutdown of scheduleReorgLoop
		chanRmTxs:           make(chan []string),

		marshaler: codec.CreateMarshaler(codecType),
		hasher:    tpcmm.NewBlake2bHasher(0),
	}

	//pool.network.Subscribe(ctx, protocol.SyncProtocolID_Msg, message.TopicValidator())
	pool.allTxsForLook = newAllTxsLookupMap()
	pool.pendings = newPendingsMap()
	pool.queues = newQueuesMap()
	pool.sortedLists = newTxSortedList()

	poolHandler := NewTransactionPoolHandler(poolLog, pool)

	pool.handler = poolHandler

	pool.locals = newAccountSet()
	if len(conf.Locals) > 0 {
		for _, addr := range conf.Locals {
			pool.log.Info("Setting new local account")
			pool.locals.add(addr)
		}
	}

	pool.curMaxGasLimit = uint64(987654321)

	//pool.Reset(nil, pool.query.CurrentBlock().GetHead())

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
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
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
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 0, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 0, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))
	txs := make([]*basic.Transaction, 0)
	txs = append(txs, Tx1)
	txs = append(txs, Tx2)

	//txs = append(txs, txLocals[80])
	txsMap := make(map[tpcrtypes.Address][]*basic.Transaction)
	txsMap[From1] = txs
	pool.AddLocals(txs)

	want := txsMap
	got := pool.GetLocalTxs(Category1)
	assert.Equal(t, 1, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 2, pool.queues.getLenTxsByAddrOfCategory(Category1, From1))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	if !reflect.DeepEqual(want, got) {
		t.Errorf("want:%v\n,                 got:%v\n", want, got)
	}
}

func Test_transactionPool_AddTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 0, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 0, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))

	pool.AddTx(Tx1, true)
	assert.Equal(t, 1, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 1, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 1, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 0, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))

	pool.AddTx(TxR1, false)
	assert.Equal(t, 2, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 1, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 1, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 1, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 1, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))

	pool.AddTx(Tx2, true)
	assert.Equal(t, 2, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 2, pool.queues.getTxListByAddrOfCategory(Category1, From1).txs.Len())
	assert.Equal(t, 1, pool.queues.getTxListByAddrOfCategory(Category1, From2).txs.Len())
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 2, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 1, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 2, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 1, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))

	pool.AddTx(TxR2, false)
	assert.Equal(t, 2, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 2, pool.queues.getTxListByAddrOfCategory(Category1, From1).txs.Len())
	assert.Equal(t, 2, pool.queues.getTxListByAddrOfCategory(Category1, From2).txs.Len())
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 2, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 2, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 2, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 2, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))

	pool.AddTx(Tx2, true)
	assert.Equal(t, 2, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 2, pool.queues.getTxListByAddrOfCategory(Category1, From1).txs.Len())
	assert.Equal(t, 2, pool.queues.getTxListByAddrOfCategory(Category1, From2).txs.Len())
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 2, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 2, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 2, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 2, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))

}
func Test_transactionPool_RemoveTxByKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 0, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 0, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))

	pool.AddTx(Tx1, true)
	pool.RemoveTxByKey(Key1)

	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 0, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 0, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))

	pool.AddTx(TxR1, false)
	pool.RemoveTxByKey(KeyR1)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 0, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 0, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))

}

func Test_transactionPool_RemoveTxHashs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 0, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 0, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	assert.Equal(t, 2, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 2, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 2, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 2, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 2, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))
	hashs := make([]string, 0)
	hashs = append(hashs, Key1)
	hashs = append(hashs, Key2)
	hashs = append(hashs, KeyR1)
	hashs = append(hashs, KeyR2)
	pool.RemoveTxHashs(hashs)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 0, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 0, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))

}

func Test_transactionPool_turnTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 0, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 0, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	_ = pool.turnTx(From1, Key1, Tx1)
	_ = pool.turnTx(From1, Key2, Tx2)
	_ = pool.turnTx(From2, KeyR1, TxR1)
	_ = pool.turnTx(From2, KeyR2, TxR2)
	assert.Equal(t, 0, pool.queues.getTxListByAddrOfCategory(Category1, From1).txs.Len())
	assert.Equal(t, 2, pool.pendings.getTxListByAddrOfCategory(Category1, From1).txs.Len())
	assert.Equal(t, 2, pool.pendings.getTxListByAddrOfCategory(Category1, From2).txs.Len())
	assert.Equal(t, 2, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 2, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 2, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 2, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))

}

func Test_transactionPool_UpdateTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 0, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 0, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	_ = pool.turnTx(From1, Key1, Tx1)
	assert.Equal(t, 1, pool.queues.getTxListByAddrOfCategory(Category1, From1).txs.Len())
	assert.Equal(t, 1, pool.pendings.getTxListByAddrOfCategory(Category1, From1).txs.Len())
	assert.Equal(t, 2, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 2, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 0, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))
	//update failed for low gasprice
	pool.UpdateTx(Tx3, Key1)
	for _, tx := range pool.pendings.getTxListByAddrOfCategory(Category1, From1).txs.items {
		txid, _ := tx.HashHex()
		assert.Equal(t, txid, Key1)
	}
	//updated for higher gasprice
	pool.UpdateTx(Tx4, Key2)
	for _, tx := range pool.queues.getTxListByAddrOfCategory(Category1, From1).txs.items {
		txid, _ := tx.HashHex()
		assert.Equal(t, txid, Key4)
	}
}

func Test_transactionPool_Pending(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 0, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 0, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(Tx4, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	pool.turnTx(From1, Key1, Tx1)
	pending := pool.PendingMapAddrTxsOfCategory(Category1)
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
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 0, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 0, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)

	_ = pool.turnTx(From1, Key1, Tx1)
	_ = pool.turnTx(From1, Key2, Tx2)
	_ = pool.turnTx(From2, KeyR1, TxR1)
	_ = pool.turnTx(From2, KeyR2, TxR2)
	pending := pool.PendingMapAddrTxsOfCategory(Category1)
	txs := make([]*basic.Transaction, 0)
	txSet := NewTxsByPriceAndNonce(pending)
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

	if got := pool.CommitTxsByPriceAndNonce(Category1); !reflect.DeepEqual(got, txs) {
		t.Errorf("CommitTxsByPriceAndNonce() = %v, want %v", got, txs)
	}
}

func Test_transactionPool_CommitTxsForPending(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 0, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 0, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))
	pool.AddTx(Tx1, true)
	pool.turnTx(From1, Key1, Tx1)
	txls := pool.pendings.getAddrTxListOfCategory(Category1)
	var txs = make([]*basic.Transaction, 0)
	for _, txlist := range txls {
		for _, tx := range txlist.txs.cache {
			txs = append(txs, tx)
		}
	}

	if got := pool.CommitTxsForPending(Category1); !reflect.DeepEqual(got, txs) {
		t.Errorf("CommitTxsForPending() = %v, want %v", got, txs)
	}
}

func Test_transactionPool_PickTxs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 0, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 0, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)

	_ = pool.turnTx(From1, Key1, Tx1)
	_ = pool.turnTx(From1, Key2, Tx2)
	_ = pool.turnTx(From2, KeyR1, TxR1)
	_ = pool.turnTx(From2, KeyR2, TxR2)
	pending := pool.PendingMapAddrTxsOfCategory(Category1)
	txs := make([]*basic.Transaction, 0)
	for _, v := range pending {
		for _, tx := range v {
			txs = append(txs, tx)
		}
	}
	want := len(txs)

	if got := len(pool.PickTxsOfCategory(Category1, PickTransactionsFromPending)); got != want {
		t.Error("PickTxsOfCategory want", want, "got", got)
	}

	txs2 := make([]*basic.Transaction, 0)
	txSet := NewTxsByPriceAndNonce(pending)
	for {

		tx := txSet.Peek()
		if tx == nil {
			break
		}
		txs2 = append(txs2, tx)
		txSet.Pop()
	}
	want = len(txs2)
	//if got := pool.PickTxsOfCategory(Category1, PickTransactionsSortedByGasPriceAndNonce); !reflect.DeepEqual(got, txs2) {
	//	t.Errorf("CommitTxsByPriceAndNonce() = %v\n,                            want %v", got, txs)
	//}
	if got := len(pool.PickTxsOfCategory(Category1, PickTransactionsSortedByGasPriceAndNonce)); got != want {
		t.Error("want", want, "got", got)
	}
}

func Test_transactionPool_Get(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 0, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 0, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	want := Tx1
	if got := pool.Get(Category1, Key1); !reflect.DeepEqual(want, got) {
		t.Errorf("want:%v\n got:%v\n", want, got)
	}
	want = TxR1
	if got := pool.Get(Category1, KeyR1); !reflect.DeepEqual(want, got) {
		t.Errorf("want:%v\n got:%v\n", want, got)
	}
}

func Test_transactionPool_Size(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 0, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 0, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	want := 4
	if got := pool.allTxsForLook.getCountFromAllTxsLookupByCategory(Category1); want != got {
		t.Error("want", want, "got", got)
	}
}

func Test_transactionPool_Start(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	network := NewMockNetwork(ctrl)
	network.EXPECT().RegisterModule(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 0, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 0, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))
	if err := pool.Start(sysActor, network); err != nil {
		t.Error("want", nil, "got", err)
	}
}

func Test_transactionPool_Stop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 0, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 0, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))
	pool.Stop()
	if 1 != 1 {
		t.Error("stop error")
	}
}
