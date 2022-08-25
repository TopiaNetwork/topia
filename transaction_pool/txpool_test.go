package transactionpool

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/pkg/profile"
	"os"
	"reflect"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/golang-lru"
	"github.com/stretchr/testify/assert"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpnet "github.com/TopiaNetwork/topia/network"
	"github.com/TopiaNetwork/topia/service"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txuni "github.com/TopiaNetwork/topia/transaction/universal"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
	txpoolmock "github.com/TopiaNetwork/topia/transaction_pool/mock"
)

var (
	TestTxPoolConfig, TestTxPoolConfig2                                                        txpooli.TransactionPoolConfig
	TxlowGasPrice, TxHighGasLimit, txlocal, txremote, Tx1, Tx2, Tx3, Tx4, Tx5, Tx6, TxR1, TxR2 *txbasic.Transaction
	keylocal, keyremote, Key1, Key2, Key3, Key4, Key5, Key6, KeyR1, KeyR2                      txbasic.TxID
	From1, From2                                                                               tpcrtypes.Address
	txLocals, txRemotes                                                                        []*txbasic.Transaction
	keyLocals, keyRemotes                                                                      []txbasic.TxID
	fromLocals                                                                                 []tpcrtypes.Address

	OldBlock, MidBlock, NewBlock             *tpchaintypes.Block
	OldBlockHead, MidBlockHead, NewBlockHead *tpchaintypes.BlockHead
	OldBlockData, MidBlockData, NewBlockData *tpchaintypes.BlockData
	OldTxs, MidTxs, NewTxs                   []*txbasic.Transaction
	sysActor                                 *actor.ActorSystem
	newcontext                               actor.Context
	TpiaLog                                  tplog.Logger
	starttime                                uint64
	TestNodeID1, TestNodeID2                 string
	Ctx                                      context.Context
	T                                        *testing.T
	pool1, pool2                             *transactionPool
	TxPoolHandler1, TxPoolHandler2           *transactionPoolHandler
	network                                  *txpoolmock.MockNetwork
)

func init() {
	TestNodeID1 = "TestNode1"
	TestNodeID2 = "TestNode2"

	Ctx = context.Background()
	TpiaLog, _ = tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.DefaultLogFormat, tplog.DefaultLogOutput, "")
	TestTxPoolConfig = txpooli.DefaultTransactionPoolConfig
	starttime = uint64(time.Now().Unix() - 105)
	keyLocals = make([]txbasic.TxID, 0)
	keyRemotes = make([]txbasic.TxID, 0)
	for i := 0; i < 20000; i++ {
		nonce := uint64(1 + i%10)
		gasprice := uint64(1000 + i%10)
		gaslimit := uint64(1000000 + i%10)
		txlocal = setTxLocal(nonce, gasprice, gaslimit)
		txlocal.Head.TimeStamp = starttime + uint64(i%100)
		txlocal.Head.FromAddr = append(txlocal.Head.FromAddr, byte(i/10))

		keylocal, _ = txlocal.TxID()
		keyLocals = append(keyLocals, keylocal)
		txLocals = append(txLocals, txlocal)

		txremote = setTxRemote(nonce, gasprice+2, gaslimit+2)
		txremote.Head.TimeStamp = starttime + uint64(i%100)

		txremote.Head.FromAddr = append(txremote.Head.FromAddr, byte(i/10))

		keyremote, _ = txremote.TxID()
		keyRemotes = append(keyRemotes, keyremote)
		txRemotes = append(txRemotes, txremote)
		if i%100 == 99 {
			nonce = 11
		}

	}
	Tx1 = setTxLocal(2, 11000, 123456)
	Tx2 = setTxLocal(3, 12000, 123456)

	Tx3 = setTxLocal(4, 10000, 123456)
	Tx4 = setTxLocal(5, 14000, 123456)

	Tx5 = setTxLocal(2, 10000, 123456) //try to replace Tx1 :false
	Tx6 = setTxLocal(3, 15000, 123456) //try to replace Tx2 :true

	From1 = tpcrtypes.Address(Tx1.Head.FromAddr)
	Key1, _ = Tx1.TxID()
	Key2, _ = Tx2.TxID()
	Key3, _ = Tx3.TxID()
	Key4, _ = Tx4.TxID()
	Key5, _ = Tx5.TxID()
	Key6, _ = Tx6.TxID()

	TxR1 = setTxRemote(2, 11000, 123456)
	TxR2 = setTxRemote(3, 12000, 123456)
	From2 = tpcrtypes.Address(TxR1.Head.FromAddr)
	KeyR1, _ = TxR1.TxID()
	KeyR2, _ = TxR2.TxID()

	TxlowGasPrice = setTxRemote(300, 100, 1000)
	TxHighGasLimit = setTxRemote(301, 10000, 9987654321)

	OldBlockHead = setBlockHead(10, 5, 1, 10, 1000000, []byte{0xaa, 0xaa, 0xaa}, []byte{0xa9, 0xa9, 0xa9})
	MidBlockHead = setBlockHead(11, 5, 1, 10, 2000000, []byte{0xab, 0xab, 0xab}, []byte{0xaa, 0xaa, 0xaa})
	NewBlockHead = setBlockHead(12, 5, 2, 10, 3000000, []byte{0xac, 0xac, 0xac}, []byte{0xab, 0xab, 0xab})

	OldTxs = append(OldTxs, txLocals[10:20]...)
	OldTxs = append(OldTxs, txRemotes[10:20]...)
	MidTxs = append(MidTxs, txLocals[20:30]...)
	MidTxs = append(MidTxs, txRemotes[20:30]...)
	NewTxs = append(NewTxs, txLocals[30:40]...)
	NewTxs = append(NewTxs, txRemotes[30:40]...)

	OldBlockData = SetBlockData(OldTxs)
	MidBlockData = SetBlockData(MidTxs)
	NewBlockData = SetBlockData(NewTxs)
	OldBlock = SetBlock(OldBlockHead, OldBlockData)
	MidBlock = SetBlock(MidBlockHead, MidBlockData)
	NewBlock = SetBlock(NewBlockHead, NewBlockData)

	sysActor = actor.NewActorSystem()

	T = &testing.T{}
	Ctrl := gomock.NewController(T)
	//defer ctrl.Finish()
	log := TpiaLog

	stateService := txpoolmock.NewMockStateQueryService(Ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := txpoolmock.NewMockBlockService(Ctrl)

	blockService.EXPECT().GetBlockByHash(tpchaintypes.BlockHash(OldBlock.Head.Hash)).AnyTimes().Return(OldBlock, nil)
	blockService.EXPECT().GetBlockByHash(tpchaintypes.BlockHash(MidBlock.Head.Hash)).AnyTimes().Return(MidBlock, nil)
	blockService.EXPECT().GetBlockByHash(tpchaintypes.BlockHash(NewBlock.Head.Hash)).AnyTimes().Return(NewBlock, nil)
	blockService.EXPECT().GetBlockByNumber(OldBlock.BlockNum()).AnyTimes().Return(OldBlock, nil)
	blockService.EXPECT().GetBlockByNumber(MidBlock.BlockNum()).AnyTimes().Return(MidBlock, nil)
	blockService.EXPECT().GetBlockByNumber(NewBlock.BlockNum()).AnyTimes().Return(NewBlock, nil)

	network = txpoolmock.NewMockNetwork(Ctrl)

	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool1 = SetNewTransactionPool(TestNodeID1, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	TxPoolHandler1 = NewTransactionPoolHandler(log, pool1, TxMsgSub)

	TestTxPoolConfig2 = txpooli.DefaultTransactionPoolConfig
	TestTxPoolConfig2.PathConf = "StorageInfo/StorageConfig2.json"
	TestTxPoolConfig2.PathTxsStorage = "StorageInfo/StorageTxs2"
	pool2 = SetNewTransactionPool(TestNodeID2, Ctx, TestTxPoolConfig2, 1, log, codec.CodecType(1), stateService, blockService, network)

}

func setTxHeadLocal(nonce uint64) *txbasic.TransactionHead {
	Category := txbasic.TransactionCategory_Topia_Universal
	txhead := &txbasic.TransactionHead{
		Category:  []byte(Category),
		ChainID:   []byte{0x01},
		Version:   1,
		Nonce:     nonce,
		FromAddr:  []byte{0x01, 0x01, 0x01},
		Signature: []byte{0x01, 0x01, 0x01},
	}
	return txhead
}
func setTxHeadRemote(nonce uint64) *txbasic.TransactionHead {
	Category := txbasic.TransactionCategory_Topia_Universal

	txhead := &txbasic.TransactionHead{
		Category:  []byte(Category),
		ChainID:   []byte{0x01},
		Version:   1,
		Nonce:     nonce,
		FromAddr:  []byte{0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01},
		Signature: []byte{0x01, 0x01, 0x01, 0x01},
	}
	return txhead
}
func setTxDataLocal(nonce, gasprice, gaslimit uint64) *txbasic.TransactionData {
	txdatauniversal := setTxUniversalLocal(nonce, gasprice, gaslimit)
	byteSpecification, _ := json.Marshal(txdatauniversal)
	txdata := &txbasic.TransactionData{
		Specification: byteSpecification,
	}

	return txdata
}
func setTxDataRemote(nonce, gasprice, gaslimit uint64) *txbasic.TransactionData {
	txdatauniversal := setTxUniversalRemote(nonce, gasprice, gaslimit)
	bytesSpecification, _ := json.Marshal(txdatauniversal)
	txdata := &txbasic.TransactionData{
		Specification: bytesSpecification,
	}

	return txdata
}
func setTxLocal(nonce, gasPrice, gasLimit uint64) *txbasic.Transaction {
	head := setTxHeadLocal(nonce)
	data := setTxDataLocal(nonce, gasPrice, gasLimit)
	local := &txbasic.Transaction{
		Head: head,
		Data: data,
	}

	return local
}
func setTxRemote(nonce, gasPrice, gasLimit uint64) *txbasic.Transaction {
	head := setTxHeadRemote(nonce)
	data := setTxDataRemote(nonce, gasPrice, gasLimit)
	remote := &txbasic.Transaction{
		Head: head,
		Data: data,
	}

	return remote
}

func setTxUniversalLocalHead(gasPrice, gasLimit uint64) *txuni.TransactionUniversalHead {
	txuniversalhead := &txuni.TransactionUniversalHead{
		Version:           1,
		FeePayer:          []byte{0x01, 0x01},
		GasPrice:          gasPrice,
		GasLimit:          gasLimit,
		FeePayerSignature: []byte{0x01, 0x01},
		Type:              1,
		Options:           1,
	}
	return txuniversalhead
}
func setTxUniversalRemoteHead(gasPrice, gasLimit uint64) *txuni.TransactionUniversalHead {
	txuniversalhead := &txuni.TransactionUniversalHead{
		Version:           1,
		FeePayer:          []byte{0x01, 0x01},
		GasPrice:          gasPrice,
		GasLimit:          gasLimit,
		FeePayerSignature: []byte{0x01, 0x01},
		Type:              1,
		Options:           1,
	}
	return txuniversalhead
}

func setTxUniversalLocalData(nonce, gasPrice, gasLimit uint64) *txuni.TransactionUniversalData {
	txtranserferlocal := setTxUniversalTransferLocal(nonce, gasPrice, gasLimit)
	byteSpecification, _ := json.Marshal(txtranserferlocal)
	txuniversaldata := &txuni.TransactionUniversalData{
		Specification: byteSpecification,
	}

	return txuniversaldata
}
func setTxUniversalRemoteData(nonce, gasPrice, gasLimit uint64) *txuni.TransactionUniversalData {
	txtranserferremote := setTxUniversalTransferRemote(nonce, gasPrice, gasLimit)
	byteSpecification, _ := json.Marshal(txtranserferremote)
	txuniversaldata := &txuni.TransactionUniversalData{
		Specification: byteSpecification,
	}

	return txuniversaldata
}
func setTxUniversalLocal(nonce, gasPrice, gasLimit uint64) *txuni.TransactionUniversal {
	head := setTxUniversalLocalHead(gasPrice, gasLimit)
	data := setTxUniversalLocalData(nonce, gasPrice, gasLimit)
	txuniversal := &txuni.TransactionUniversal{
		Head: head,
		Data: data,
	}

	return txuniversal
}
func setTxUniversalRemote(nonce, gasPrice, gasLimit uint64) *txuni.TransactionUniversal {
	head := setTxUniversalRemoteHead(gasPrice, gasLimit)
	data := setTxUniversalRemoteData(nonce, gasPrice, gasLimit)
	txuniversal := &txuni.TransactionUniversal{
		Head: head,
		Data: data,
	}
	return txuniversal

}
func setTxUniversalTransferLocal(nonce, gasprice, gaslimit uint64) *txuni.TransactionUniversalTransfer {
	basicHead := setTxHeadLocal(nonce)
	headUniversal := setTxUniversalLocalHead(gasprice, gaslimit)
	txuniversaltransfer := &txuni.TransactionUniversalTransfer{
		TransactionHead:          *basicHead,
		TransactionUniversalHead: *headUniversal,
		TargetAddr:               tpcrtypes.Address("0x01a1a1a"),
	}
	return txuniversaltransfer
}
func setTxUniversalTransferRemote(nonce, gasprice, gaslimit uint64) *txuni.TransactionUniversalTransfer {
	basicHead := setTxHeadRemote(nonce)
	headUniversal := setTxUniversalRemoteHead(gasprice, gaslimit)
	txuniversaltransfer := &txuni.TransactionUniversalTransfer{
		TransactionHead:          *basicHead,
		TransactionUniversalHead: *headUniversal,
		TargetAddr:               tpcrtypes.Address("0x02b2b2b"),
	}
	return txuniversaltransfer
}

func setBlockHead(height, epoch, round uint64, txcount uint32, timestamp uint64, hash, parrenthash []byte) *tpchaintypes.BlockHead {
	blockhead := &tpchaintypes.BlockHead{ChainID: []byte{0x01}, Version: 1, Height: height, Epoch: epoch, Round: round,
		ParentBlockHash: parrenthash, Launcher: nil, Proposer: nil, Proof: nil, VRFProof: nil,
		MaxPri: nil, VoteAggSignature: nil, TxCount: txcount, TxRoot: nil,
		TxResultRoot: nil, StateRoot: nil, GasFees: nil, TimeStamp: timestamp, ElapsedSpan: 0, Hash: hash,
		Reserved: nil}
	return blockhead
}
func SetBlock(head *tpchaintypes.BlockHead, data *tpchaintypes.BlockData) *tpchaintypes.Block {
	block := &tpchaintypes.Block{
		Head: head,
		Data: data,
	}
	return block
}
func SetBlockData(txs []*txbasic.Transaction) *tpchaintypes.BlockData {
	blockdata := &tpchaintypes.BlockData{
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

func SetNewTransactionPool(nodeID string, ctx context.Context, conf txpooli.TransactionPoolConfig, level tplogcmm.LogLevel,
	log tplog.Logger, codecType codec.CodecType, stateQueryService service.StateQueryService,
	blockService service.BlockService, network tpnet.Network) *transactionPool {

	conf = (&conf).Check()
	conf.MaxCntOfEachAccount = 16
	conf.TxPoolMaxSize = 1024
	conf = (&conf).Check()
	poolLog := tplog.CreateModuleLogger(level, "TransactionPool", log)

	pool := &transactionPool{
		nodeId: nodeID,
		config: conf,
		log:    poolLog,
		level:  level,
		ctx:    ctx,

		pending:       newAccTxs(),
		prepareTxs:    newAccTxs(),
		pendingNonces: newAccountNonce(stateQueryService),
		allWrappedTxs: newAllLookupTxs(),
		sortedTxs:     newSortedTxList(sortedByMaxGasPrice),

		chanAddTxs:     make(chan *addTxsItem, ChanAddTxsSize),
		chanSortedItem: make(chan *sortedItem, ChanAddTxsSize),

		chanSysShutdown:    make(chan struct{}),
		chanBlockAdded:     make(chan *tpchaintypes.Block, ChanBlockAddedSize),
		chanBlocksRevert:   make(chan []*tpchaintypes.Block),
		chanDelTxsStorage:  make(chan []txbasic.TxID, ChanDelTxsStorage),
		chanSaveTxsStorage: make(chan []*wrappedTx, ChanSaveTxsStorage),

		marshaler: codec.CreateMarshaler(codecType),
		hasher:    tpcmm.NewBlake2bHasher(0),
		wg:        sync.WaitGroup{},
		mu:        sync.RWMutex{},
	}
	pool.packagedTxIDs, _ = lru.New(TxCacheSize)
	pool.txCache, _ = lru.New(TxCacheSize)

	pool.txServant = newTransactionPoolServant(stateQueryService, blockService, network)

	if pool.config.IsLoadCfg {
		pool.loadAndSetPoolConfig(pool.config.PathConf)
	}
	if pool.config.IsLoadTxs {
		pool.LoadLocalTxsData(pool.config.PathTxsStorage)
	}

	pool.loopChanSelect()

	TxMsgSub = &txMsgSubProcessor{txPool: pool, log: pool.log, nodeID: pool.nodeId}
	//
	////subscribe
	//pool.txServant.Subscribe(ctx, protocol.SyncProtocolID_Msg,
	//	true,
	//	TxMsgSub.Validate)
	//
	poolHandler := NewTransactionPoolHandler(poolLog, pool, TxMsgSub)
	pool.handler = poolHandler
	return pool
}

func Test_transactionPool_GetLocalTxs(t *testing.T) {

	pool1.TruncateTxPool()
	assert.Equal(t, 0, len(pool1.GetLocalTxs()))
	assert.Equal(t, 0, len(pool1.GetRemoteTxs()))
	txs := make([]*txbasic.Transaction, 0)
	txs = append(txs, Tx1)
	txs = append(txs, Tx2)
	txsMap := make(map[tpcrtypes.Address][]*txbasic.Transaction)
	txsMap[From1] = txs
	pool1.addLocals(txs)

	want := txsMap[From1]

	got := pool1.GetLocalTxs()

	assert.Equal(t, int64(2), pool1.poolCount)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("want:%v\n,                 got:%v\n", want, got)
	}
}

func Test_transactionPool_AddTx(t *testing.T) {
	pool1.TruncateTxPool()

	assert.Equal(t, int64(0), pool1.poolCount)

	//add one local tx
	err := pool1.AddTx(Tx1, true)
	if err != nil {
		fmt.Println(err)
	}
	call1 := func(k interface{}, v interface{}) {
		fmt.Println("preparetxs", v.(*mapNonceTxs).flatten())
	}
	pool1.prepareTxs.addrTxs.IterateCallback(call1)
	call2 := func(k interface{}, v interface{}) {
		fmt.Println("pendingTxs", v.(*mapNonceTxs).flatten())
	}
	pool1.prepareTxs.addrTxs.IterateCallback(call2)

	pending1, _ := pool1.PendingOfAddress(From1)
	assert.Equal(t, 1, len(pending1))
	assert.Equal(t, int64(1), pool1.poolCount)
	assert.Equal(t, 1, len(pool1.GetLocalTxs()))
	assert.Equal(t, 0, len(pool1.GetRemoteTxs()))
	assert.Equal(t, txpooli.StateTxAdded, pool1.PeekTxState(Key1))

	// add one remote tx
	pool1.AddTx(TxR1, false)
	pending2, _ := pool1.PendingOfAddress(From2)
	assert.Equal(t, 1, len(pending2))
	assert.Equal(t, int64(2), pool1.poolCount)
	assert.Equal(t, 1, len(pool1.GetLocalTxs()))
	assert.Equal(t, 1, len(pool1.GetRemoteTxs()))
	assert.Equal(t, txpooli.StateTxAdded, pool1.PeekTxState(KeyR1))
	// add one local tx
	pool1.AddTx(Tx2, true)
	pending1, _ = pool1.PendingOfAddress(From1)
	assert.Equal(t, 2, len(pending1))
	assert.Equal(t, int64(3), pool1.poolCount)
	assert.Equal(t, 2, len(pool1.GetLocalTxs()))
	assert.Equal(t, 1, len(pool1.GetRemoteTxs()))
	assert.Equal(t, txpooli.StateTxAdded, pool1.PeekTxState(Key2))
	// add one remote tx
	pool1.AddTx(TxR2, false)
	pending2, _ = pool1.PendingOfAddress(From2)
	assert.Equal(t, 2, len(pending2))
	assert.Equal(t, int64(4), pool1.poolCount)
	assert.Equal(t, 2, len(pool1.GetLocalTxs()))
	assert.Equal(t, 2, len(pool1.GetRemoteTxs()))
	assert.Equal(t, txpooli.StateTxAdded, pool1.PeekTxState(KeyR2))
	// add a same tx
	pool1.AddTx(Tx2, true)
	pending1, _ = pool1.PendingOfAddress(From1)
	assert.Equal(t, 2, len(pending1))
	assert.Equal(t, int64(4), pool1.poolCount)
	assert.Equal(t, 2, len(pool1.GetLocalTxs()))
	assert.Equal(t, 2, len(pool1.GetRemoteTxs()))
	assert.Equal(t, txpooli.StateTxAdded, pool1.PeekTxState(Key2))
}

func Test_transactionPool_RemoveTxByKey(t *testing.T) {
	pool1.TruncateTxPool()

	assert.Equal(t, int64(0), pool1.poolCount)

	pool1.AddTx(Tx1, true)
	assert.Equal(t, int64(1), pool1.poolCount)
	fmt.Println(pool1.pending.getTxsByAddr(From1))
	fmt.Println(pool1.prepareTxs.getTxsByAddr(From1))

	pool1.RemoveTxByKey(Key1)
	assert.Equal(t, int64(0), pool1.poolCount)
	fmt.Println("0001")
	pool1.AddTx(TxR1, false)
	assert.Equal(t, int64(1), pool1.poolCount)
	fmt.Println("0002")

	pool1.RemoveTxByKey(KeyR1)
	assert.Equal(t, int64(0), pool1.poolCount)
	fmt.Println("0003")

	pool1.AddTx(Tx1, true)
	fmt.Println(pool1.Get(Key1))
	fmt.Println(pool1.pending.getTxsByAddr(From1))
	fmt.Println(pool1.prepareTxs.getTxsByAddr(From1))
	pool1.AddTx(Tx2, true)
	fmt.Println(pool1.Get(Key2))
	fmt.Println(pool1.pending.getTxsByAddr(From1))
	fmt.Println(pool1.prepareTxs.getTxsByAddr(From1))
	assert.Equal(t, int64(2), pool1.poolCount)
	err := pool1.RemoveTxByKey(Key2)

	fmt.Println(pool1.pending.getTxsByAddr(From1))
	fmt.Println(pool1.prepareTxs.getTxsByAddr(From1))
	fmt.Println(err)
	assert.Equal(t, int64(1), pool1.poolCount)
	fmt.Println(pool1.Get(Key2))
	err = pool1.RemoveTxByKey(Key1)
	fmt.Println(err)
	assert.Equal(t, int64(0), pool1.poolCount)
	fmt.Println(pool1.Get(Key1))

}

func Test_transactionPool_RemoveTxHashes(t *testing.T) {

	pool1.TruncateTxPool()

	assert.Equal(t, int64(0), pool1.poolCount)
	pool1.AddTx(Tx1, true)
	pool1.AddTx(Tx2, true)
	pool1.AddTx(TxR1, false)
	pool1.AddTx(TxR2, false)
	assert.Equal(t, 2, len(pool1.GetLocalTxs()))
	assert.Equal(t, 2, len(pool1.GetRemoteTxs()))

	var hashes []txbasic.TxID
	hashes = append(hashes, Key1)
	hashes = append(hashes, Key2)
	hashes = append(hashes, KeyR1)
	hashes = append(hashes, KeyR2)

	pool1.RemoveTxBatch(hashes)
	assert.Equal(t, 0, len(pool1.GetLocalTxs()))
	assert.Equal(t, 0, len(pool1.GetRemoteTxs()))

	pool1.AddTx(Tx1, true)
	pool1.AddTx(Tx2, true)
	pool1.AddTx(Tx4, true)
	pool1.AddTx(TxR1, false)
	pool1.AddTx(TxR2, false)
	hashes1 := make([]txbasic.TxID, 0)
	hashes1 = append(hashes1, Key1)
	hashes1 = append(hashes1, Key2)
	hashes1 = append(hashes1, Key4)
	hashes2 := make([]txbasic.TxID, 0)
	hashes2 = append(hashes2, KeyR1)
	hashes2 = append(hashes2, KeyR2)

	assert.Equal(t, 3, len(pool1.GetLocalTxs()))
	err := pool1.RemoveTxBatch(hashes1)
	fmt.Println(err)
	assert.Equal(t, 0, len(pool1.GetLocalTxs()))

	//Test unforced the removal of consecutive transactions
	assert.Equal(t, 2, len(pool1.GetRemoteTxs()))
	err = pool1.RemoveTxBatch(hashes2)
	assert.Equal(t, 0, len(pool1.GetRemoteTxs()))
	//Test forced the removal of discontinuous transactions
	errs := pool1.RemoveTxBatch(hashes1)
	for _, err := range errs {
		fmt.Println(err)
	}
}

func Test_transactionPool_UpdateTx(t *testing.T) {
	pool1.TruncateTxPool()

	assert.Equal(t, int64(0), pool1.poolCount)

	pool1.AddTx(Tx1, true)
	pool1.AddTx(Tx2, true)
	fmt.Println("001:", pool1.poolCount)
	assert.Equal(t, 2, len(pool1.GetLocalTxs()))

	//update failed for low gasPrice
	pool1.UpdateTx(Tx5, Key1)
	want1 := true
	want2 := false
	_, got1 := pool1.Get(Key1)
	_, got2 := pool1.Get(Key5)
	assert.EqualValues(t, want1, got1)
	assert.EqualValues(t, want2, got2)
	fmt.Println("002:", pool1.poolCount)

	//updated for higher gasPrice
	pool1.UpdateTx(Tx6, Key2)
	want1 = false
	want2 = true
	_, got1 = pool1.Get(Key2)
	_, got2 = pool1.Get(Key6)
	assert.EqualValues(t, want1, got1)
	assert.EqualValues(t, want2, got2)
	fmt.Println("003", pool1.poolCount)
	pool1.TruncateTxPool()

	pool1.AddTx(Tx1, true)
	_, ok := pool1.allWrappedTxs.Get(Key1)
	fmt.Println("004", ok)

	pool1.AddTx(Tx2, true)
	_, ok = pool1.allWrappedTxs.Get(Key2)
	fmt.Println(ok)
	pool1.AddTx(Tx3, true)
	_, ok = pool1.allWrappedTxs.Get(Key3)
	fmt.Println(ok)

	pool1.RemoveTxByKey(Key2)
	fmt.Println("005", pool1.poolCount)
	fmt.Println("006", pool1.pendingCount)
	//update tx2 for higher price ,tx2 has been removed
	pool1.AddTx(Tx6, true)
	want1 = false
	want2 = true
	_, got1 = pool1.Get(Key2)
	_, got2 = pool1.Get(Key6)
	assert.EqualValues(t, want1, got1)
	assert.EqualValues(t, want2, got2)
	fmt.Println("007", pool1.poolCount)
	fmt.Println("008", pool1.pendingCount)

}

func Test_transactionPool_PickTxs(t *testing.T) {
	pool1.TruncateTxPool()

	assert.Equal(t, int64(0), pool1.poolCount)
	fmt.Println(pool1.isInRemove)
	pool1.AddTx(TxR1, false)
	pool1.AddTx(TxR2, false)
	pool1.AddTx(Tx1, true)
	pool1.AddTx(Tx2, true)
	pool1.AddTx(Tx3, true)

	pool1.RemoveTxByKey(Key3)
	want := make([]*txbasic.Transaction, 0)
	want = append(want, TxR1)
	want = append(want, TxR2)

	want = append(want, Tx1)
	want = append(want, Tx2)
	//want = append(want, Tx3)

	call1 := func(k interface{}, v interface{}) {
		addr, list := k.(tpcrtypes.Address), v.(*mapNonceTxs)
		for _, tx := range list.flatten() {
			fmt.Println("001 pending addr:", addr, "nonce", tx.Head.Nonce)
		}
	}
	pool1.pending.addrTxs.IterateCallback(call1)
	call2 := func(k interface{}, v interface{}) {
		addr, list := k.(tpcrtypes.Address), v.(*mapNonceTxs)
		for _, tx := range list.flatten() {
			fmt.Println("001 prepare addr:", addr, "nonce", tx.Head.Nonce)
		}
	}
	pool1.prepareTxs.addrTxs.IterateCallback(call2)

	got := pool1.PickTxs()

	assert.EqualValues(t, want, got)
	pool1.TruncateTxPool()

	assert.Equal(t, int64(0), pool1.poolCount)
	pool1.AddTx(Tx1, true)
	call3 := func(k interface{}, v interface{}) {
		addr, list := k.(tpcrtypes.Address), v.(*mapNonceTxs)
		for _, tx := range list.flatten() {
			fmt.Println("002 pending addr:", addr, "nonce", tx.Head.Nonce)
		}
	}
	pool1.pending.addrTxs.IterateCallback(call3)
	fmt.Println("002", pool1.pendingCount, pool1.poolCount)
	assert.Equal(t, 1, len(pool1.GetLocalTxs()))
	pool1.AddTx(Tx2, true)
	fmt.Println("002", pool1.pendingCount, pool1.poolCount)
	assert.Equal(t, 2, len(pool1.GetLocalTxs()))
	pool1.AddTx(Tx4, true)
	fmt.Println("002", pool1.pendingCount, pool1.poolCount)
	assert.Equal(t, 3, len(pool1.GetLocalTxs()))
	pool1.AddTx(TxR1, false)
	fmt.Println("002", pool1.pendingCount, pool1.poolCount)
	assert.Equal(t, 1, len(pool1.GetRemoteTxs()))
	pool1.AddTx(TxR2, false)
	fmt.Println("002", pool1.pendingCount, pool1.poolCount)
	assert.Equal(t, 2, len(pool1.GetRemoteTxs()))
	call4 := func(k interface{}, v interface{}) {
		addr, list := k.(tpcrtypes.Address), v.(*mapNonceTxs)
		for _, tx := range list.flatten() {
			fmt.Println("003 pending addr:", addr, "nonce", tx.Head.Nonce)
		}
	}
	pool1.pending.addrTxs.IterateCallback(call4)

	call5 := func(k interface{}, v interface{}) {
		addr, list := k.(tpcrtypes.Address), v.(*mapNonceTxs)
		for _, tx := range list.flatten() {
			fmt.Println("004 prepare addr:", addr, "nonce", tx.Head.Nonce)
		}
	}
	pool1.prepareTxs.addrTxs.IterateCallback(call5)

	want = make([]*txbasic.Transaction, 0)
	want = append(want, Tx1)
	want = append(want, Tx2)
	want = append(want, TxR1)
	want = append(want, TxR2)
	fmt.Println("005 sorted len", pool1.sortedTxs.txList.Len())
	got = pool1.PickTxs()
	assert.EqualValues(t, want, got)
	assert.Equal(t, int64(1), pool1.poolCount)
}

func Test_transactionPool_Count(t *testing.T) {
	pool1.TruncateTxPool()

	assert.Equal(t, int64(0), pool1.Size())

	pool1.AddTx(Tx1, true)
	pool1.AddTx(Tx2, true)
	pool1.AddTx(TxR1, false)
	pool1.AddTx(TxR2, false)
	want := 4
	fmt.Println(pool1.Count())
	assert.Equal(t, int64(want), pool1.Count())
	pool1.RemoveTxByKey(Key1)
	want = 3
	assert.Equal(t, int64(want), pool1.Count())
	fmt.Println(pool1.Count())

}

func Test_transactionPool_Size(t *testing.T) {
	pool1.TruncateTxPool()

	assert.Equal(t, int64(0), pool1.Size())

	pool1.AddTx(Tx1, true)
	pool1.AddTx(Tx2, true)
	pool1.AddTx(TxR1, false)
	pool1.AddTx(TxR2, false)
	want := Tx1.Size()
	want += Tx2.Size()
	want += TxR1.Size()
	want += TxR2.Size()
	fmt.Println(pool1.Size())
	assert.Equal(t, int64(want), pool1.Size())
	pool1.RemoveTxByKey(Key1)
	want -= Tx1.Size()
	assert.Equal(t, int64(want), pool1.Size())
	fmt.Println(pool1.Size())

}

func Test_transactionPool_Start(t *testing.T) {
	pool1.TruncateTxPool()

	assert.Equal(t, int64(0), pool1.Size())

	if err := pool1.Start(sysActor, network); err != nil {
		t.Error("want", nil, "got", err)
	}
}

func Test_transactionPool_Stop(t *testing.T) {
	pool1.TruncateTxPool()

	pool1.Stop()
	if 1 != 1 {
		t.Error("stop error")
	}
}

func Test_transactionPool_SetTxPoolConfig(t *testing.T) {
	pool1.TruncateTxPool()

	newconf := pool1.config
	newconf.TxPoolMaxSize = 123456789
	pool1.SetTxPoolConfig(newconf)
	want := newconf.TxPoolMaxSize
	got := pool1.config.TxPoolMaxSize
	if !assert.Equal(t, want, got) {
		t.Error("want", want, "got", got)
	}
}

func Test_transactionPool_all(t *testing.T) {

	pool1.TruncateTxPool()
	pool2.TruncateTxPool()

	defer profile.Start(profile.MemProfile, profile.MemProfileRate(1)).Stop()
	f, _ := os.OpenFile("cpu.pprof", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	defer f.Close()
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()
	var localTx, remoteTx *txbasic.Transaction
	//var localTxs, RemoteTxs []*txbasic.Transaction
	wgall := sync.WaitGroup{}
	bytesBuffer := bytes.NewBuffer([]byte{})

	giveTxs := func(start, end int) {
		defer wgall.Done()
		for i := start; i <= end; i++ {
			nonce := uint64(1 + i%20)
			if i%40 == 35 {
				nonce = 10
			}
			if i%80 == 75 {
				nonce = 23
			}
			gasprice := uint64(1234 + i%20)
			gaslimit := uint64(1000000 + i%20)

			x := int32(i / 20)
			bytesBuffer.Truncate(0)
			binary.Write(bytesBuffer, binary.BigEndian, x)
			localTx = setTxLocal(nonce, gasprice, gaslimit)
			localTx.Head.TimeStamp = starttime + uint64(i%20)*10
			localTx.Head.FromAddr = append(localTx.Head.FromAddr, bytesBuffer.Bytes()...)
			localTxData, _ := json.Marshal(localTx)

			msg1 := &TxMessage{
				Data: localTxData,
			}

			remoteTx = setTxRemote(nonce, gasprice+2, gaslimit+2)
			remoteTx.Head.TimeStamp = starttime + uint64(i%20)*10
			remoteTx.Head.FromAddr = append(remoteTx.Head.FromAddr, bytesBuffer.Bytes()...)
			remoteTxData, _ := json.Marshal(remoteTx)
			msg2 := &TxMessage{
				Data: remoteTxData,
			}

			go pool1.AddTx(localTx, true)

			go pool2.processTX(msg1)

			go pool2.AddTx(remoteTx, true)

			go pool1.processTX(msg2)

			time.Sleep(10 * time.Microsecond)
		}
	}
	wgall.Add(1)
	go giveTxs(1, 100000)
	wgall.Add(1)
	go func() {
		defer wgall.Done()
		var report = time.NewTicker(1000 * time.Millisecond)
		defer report.Stop()
		for {

			select {
			case <-report.C:
				fmt.Println(pool1.nodeId, time.Now(), "len pending account",
					len(pool1.pending.addrTxs.AllKeys()),
					"len prepare account", len(pool1.prepareTxs.addrTxs.AllKeys()),
					"pool1.count", pool1.Count())
				fmt.Println(pool2.nodeId, time.Now(), "len pending account",
					len(pool2.pending.addrTxs.AllKeys()),
					"len prepare account", len(pool2.prepareTxs.addrTxs.AllKeys()),
					"pool2.count", pool2.Count())
			}
		}
	}()
	packInterval := 2 * time.Second

	wgall.Add(1)
	go func() {
		defer wgall.Done()
		var pack = time.NewTicker(packInterval)
		defer pack.Stop()

		var isNode1Pack = true
		height := uint64(0)
		var preBlockHash = []byte{0x01}
		var revertBlocks = make([]*tpchaintypes.Block, 2)

		for {
			select {
			case <-pack.C:
				fmt.Println("#####*******#######height:", height, "isNode1Pack", isNode1Pack,
					"pendingCnt1", atomic.LoadInt64(&pool1.pendingCount), "pendingCnt2", atomic.LoadInt64(&pool2.pendingCount), "******************************************")

				if isNode1Pack {
					packTxs := pool1.PickTxs()
					fmt.Println("node1 pick *************** len txs:", len(packTxs), "************************")
					curBlockHead := setBlockHead(height, 1, 1, uint32(len(packTxs)), uint64(time.Now().Nanosecond()), []byte{0x00, 0x00, 0x00, 0x01}, preBlockHash)
					curBlockData := SetBlockData(packTxs)
					curBlock := SetBlock(curBlockHead, curBlockData)
					curBlock.Head.Hash, _ = pool1.marshaler.Marshal(curBlock)

					pool1.dropTxsForBlockAdded(curBlock)
					pool2.dropTxsForBlockAdded(curBlock)

					if height%20 == 18 || height%20 == 19 {
						revertBlocks = append(revertBlocks, curBlock)
					}
					if len(revertBlocks) == 2 {
						pool1.chanBlocksRevert <- revertBlocks
						pool2.chanBlocksRevert <- revertBlocks
						revertBlocks = make([]*tpchaintypes.Block, 2)
					}
					height += 1
					preBlockHash = curBlock.Head.Hash
					isNode1Pack = false
					pack.Reset(packInterval)
				} else {
					packTxs := pool2.PickTxs()
					fmt.Println("node2 *************** pick txs:", len(packTxs), "*******************")

					curBlockHead := setBlockHead(height, 1, 1, uint32(len(packTxs)), uint64(time.Now().Nanosecond()), []byte{0x01, 0x02}, preBlockHash)
					curBlockData := SetBlockData(packTxs)
					curBlock := SetBlock(curBlockHead, curBlockData)
					curBlock.Head.Hash, _ = pool2.marshaler.Marshal(curBlock)

					pool1.dropTxsForBlockAdded(curBlock)
					pool2.dropTxsForBlockAdded(curBlock)

					if height%10 == 8 || height%10 == 9 {
						revertBlocks = append(revertBlocks, curBlock)
					}
					if len(revertBlocks) == 2 {
						pool1.chanBlocksRevert <- revertBlocks
						pool2.chanBlocksRevert <- revertBlocks
						revertBlocks = make([]*tpchaintypes.Block, 2)
					}
					height += 1
					preBlockHash = curBlock.Head.Hash
					isNode1Pack = true
					pack.Reset(packInterval)
				}
			}
		}
	}()
	time.Sleep(240 * time.Second)
	fmt.Println("test done!")

}
