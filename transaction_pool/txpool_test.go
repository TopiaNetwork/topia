package transactionpool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"runtime/pprof"
	"sync"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/golang-lru"
	"github.com/pkg/profile"
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
	Category1, Category2                                                                       txbasic.TransactionCategory
	TestTxPoolConfig                                                                           txpooli.TransactionPoolConfig
	TxlowGasPrice, TxHighGasLimit, txlocal, txremote, Tx1, Tx2, Tx3, Tx4, Tx5, Tx6, TxR1, TxR2 *txbasic.Transaction
	txHead                                                                                     *txbasic.TransactionHead
	txData                                                                                     *txbasic.TransactionData
	keylocal, keyremote, Key1, Key2, Key3, Key4, Key5, Key6, KeyR1, KeyR2                      txbasic.TxID
	from, to, From1, From2                                                                     tpcrtypes.Address
	txLocals, txRemotes                                                                        []*txbasic.Transaction
	keyLocals, keyRemotes                                                                      []txbasic.TxID
	fromLocals, fromRemotes, toLocals, toRemotes                                               []tpcrtypes.Address

	OldBlock, MidBlock, NewBlock             *tpchaintypes.Block
	OldBlockHead, MidBlockHead, NewBlockHead *tpchaintypes.BlockHead
	OldBlockData, MidBlockData, NewBlockData *tpchaintypes.BlockData
	OldTxs, MidTxs, NewTxs                   []*txbasic.Transaction
	sysActor                                 *actor.ActorSystem
	newcontext                               actor.Context
	TpiaLog                                  tplog.Logger
	starttime                                uint64
	NodeID                                   string
	Ctx                                      context.Context
)

func init() {
	NodeID = "TestNode"
	Ctx = context.Background()
	Category1 = txbasic.TransactionCategory_Topia_Universal
	TpiaLog, _ = tplog.CreateMainLogger(tplogcmm.InfoLevel, tplog.DefaultLogFormat, tplog.DefaultLogOutput, "")
	TestTxPoolConfig = txpooli.DefaultTransactionPoolConfig
	starttime = uint64(time.Now().Unix() - 105)
	keyLocals = make([]txbasic.TxID, 0)
	keyRemotes = make([]txbasic.TxID, 0)
	for i := 1; i <= 20000; i++ {
		nonce := uint64(i)
		gasprice := uint64(i * 1000)
		gaslimit := uint64(i * 1000000)
		txlocal = setTxLocal(nonce, gasprice, gaslimit)
		txlocal.Head.TimeStamp = starttime + uint64(i)
		if i > 1 {
			txlocal.Head.FromAddr = append(txlocal.Head.FromAddr, byte(i))
		}
		keylocal, _ = txlocal.TxID()
		keyLocals = append(keyLocals, keylocal)
		txLocals = append(txLocals, txlocal)

		txremote = setTxRemote(nonce, gasprice+2, gaslimit+2)
		txremote.Head.TimeStamp = starttime + uint64(i)
		if i > 1 {
			txremote.Head.FromAddr = append(txremote.Head.FromAddr, byte(200*i))
		}
		keyremote, _ = txremote.TxID()
		keyRemotes = append(keyRemotes, keyremote)
		txRemotes = append(txRemotes, txremote)
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
		FromAddr:  []byte{0x22, 0x22, 0x22},
		Signature: []byte{0x02, 0x02, 0x02},
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

		pending:            newPending(),
		allWrappedTxs:      newAllLookupTxs(),
		packagedTxs:        newPackagedTxs(),
		chanSysShutdown:    make(chan struct{}),
		chanBlockAdded:     make(chan *tpchaintypes.Block, ChanBlockAddedSize),
		chanBlocksRevert:   make(chan []*tpchaintypes.Block, ChanBlocksRevertSize),
		chanRmTxs:          make(chan []txbasic.TxID),
		chanDelTxsStorage:  make(chan []txbasic.TxID, ChanDelTxsStorage),
		chanSaveTxsStorage: make(chan []*wrappedTx, ChanSaveTxsStorage),

		marshaler: codec.CreateMarshaler(codecType),
		hasher:    tpcmm.NewBlake2bHasher(0),
		wg:        sync.WaitGroup{},
	}
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog

	stateService := txpoolmock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)
	blockService := txpoolmock.NewMockBlockService(ctrl)
	network := txpoolmock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	pool.TruncateTxPool()

	assert.Equal(t, 0, len(pool.GetLocalTxs()))
	assert.Equal(t, 0, len(pool.GetRemoteTxs()))
	txs := make([]*txbasic.Transaction, 0)
	txs = append(txs, Tx1)
	txs = append(txs, Tx2)
	txsMap := make(map[tpcrtypes.Address][]*txbasic.Transaction)
	txsMap[From1] = txs
	pool.addLocals(txs)

	want := txsMap[From1]

	got := pool.GetLocalTxs()

	assert.Equal(t, int64(2), pool.poolCount)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("want:%v\n,                 got:%v\n", want, got)
	}
}

func Test_transactionPool_AddTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog
	stateService := txpoolmock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)
	blockService := txpoolmock.NewMockBlockService(ctrl)

	network := txpoolmock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	pool.TruncateTxPool()

	assert.Equal(t, int64(0), pool.poolCount)

	//add one local tx
	pool.AddTx(Tx1, true)
	pending1, _ := pool.PendingOfAddress(From1)
	assert.Equal(t, 1, len(pending1))
	assert.Equal(t, int64(1), pool.poolCount)
	assert.Equal(t, 1, len(pool.GetLocalTxs()))
	assert.Equal(t, 0, len(pool.GetRemoteTxs()))
	assert.Equal(t, txpooli.StateTxAdded, pool.PeekTxState(Key1))

	// add one remote tx
	pool.AddTx(TxR1, false)
	pending2, _ := pool.PendingOfAddress(From2)
	assert.Equal(t, 1, len(pending2))
	assert.Equal(t, int64(2), pool.poolCount)
	assert.Equal(t, 1, len(pool.GetLocalTxs()))
	assert.Equal(t, 1, len(pool.GetRemoteTxs()))
	assert.Equal(t, txpooli.StateTxAdded, pool.PeekTxState(KeyR1))
	// add one local tx
	pool.AddTx(Tx2, true)
	pending1, _ = pool.PendingOfAddress(From1)
	assert.Equal(t, 2, len(pending1))
	assert.Equal(t, int64(3), pool.poolCount)
	assert.Equal(t, 2, len(pool.GetLocalTxs()))
	assert.Equal(t, 1, len(pool.GetRemoteTxs()))
	assert.Equal(t, txpooli.StateTxAdded, pool.PeekTxState(Key2))
	// add one remote tx
	pool.AddTx(TxR2, false)
	pending2, _ = pool.PendingOfAddress(From2)
	assert.Equal(t, 2, len(pending2))
	assert.Equal(t, int64(4), pool.poolCount)
	assert.Equal(t, 2, len(pool.GetLocalTxs()))
	assert.Equal(t, 2, len(pool.GetRemoteTxs()))
	assert.Equal(t, txpooli.StateTxAdded, pool.PeekTxState(KeyR2))
	// add a same tx
	pool.AddTx(Tx2, true)
	pending1, _ = pool.PendingOfAddress(From1)
	assert.Equal(t, 2, len(pending1))
	assert.Equal(t, int64(4), pool.poolCount)
	assert.Equal(t, 2, len(pool.GetLocalTxs()))
	assert.Equal(t, 2, len(pool.GetRemoteTxs()))
	assert.Equal(t, txpooli.StateTxAdded, pool.PeekTxState(Key2))
}

func Test_transactionPool_RemoveTxByKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog

	stateService := txpoolmock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := txpoolmock.NewMockBlockService(ctrl)
	network := txpoolmock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	pool.TruncateTxPool()

	assert.Equal(t, int64(0), pool.poolCount)

	pool.AddTx(Tx1, true)
	assert.Equal(t, int64(1), pool.poolCount)

	pool.RemoveTxByKey(Key1, true)
	assert.Equal(t, int64(0), pool.poolCount)

	pool.AddTx(TxR1, false)
	assert.Equal(t, int64(1), pool.poolCount)

	pool.RemoveTxByKey(KeyR1, true)
	assert.Equal(t, int64(0), pool.poolCount)
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	assert.Equal(t, int64(2), pool.poolCount)
	err := pool.RemoveTxByKey(Key2, false)
	fmt.Println(err)
	assert.Equal(t, int64(2), pool.poolCount)
	pool.RemoveTxByKey(Key1, false)
	assert.Equal(t, int64(1), pool.poolCount)

}

func Test_transactionPool_RemoveTxHashes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog

	stateService := txpoolmock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := txpoolmock.NewMockBlockService(ctrl)
	network := txpoolmock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)

	pool.TruncateTxPool()

	assert.Equal(t, int64(0), pool.poolCount)
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	assert.Equal(t, 2, len(pool.GetLocalTxs()))
	assert.Equal(t, 2, len(pool.GetRemoteTxs()))

	hashs := make([]txbasic.TxID, 0)
	hashs = append(hashs, Key1)
	hashs = append(hashs, Key2)
	hashs = append(hashs, KeyR1)
	hashs = append(hashs, KeyR2)
	// Test remove forced transactions under different accounts
	pool.RemoveTxHashes(hashs, true)
	assert.Equal(t, 0, len(pool.GetLocalTxs()))
	assert.Equal(t, 0, len(pool.GetRemoteTxs()))

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(Tx4, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	hashes1 := make([]txbasic.TxID, 0)
	hashes1 = append(hashes1, Key1)
	hashes1 = append(hashes1, Key2)
	hashes1 = append(hashes1, Key4)
	hashes2 := make([]txbasic.TxID, 0)
	hashes2 = append(hashes2, KeyR1)
	hashes2 = append(hashes2, KeyR2)

	//Test  remove unforced discontinuous transactions
	err := pool.RemoveTxHashes(hashes1, false)
	fmt.Println(err)
	assert.Equal(t, 3, len(pool.GetLocalTxs()))

	//Test unforced the removal of consecutive transactions
	err = pool.RemoveTxHashes(hashes2, false)
	assert.Equal(t, err, nil)
	assert.Equal(t, 0, len(pool.GetRemoteTxs()))

	//Test forced the removal of discontinuous transactions
	err = pool.RemoveTxHashes(hashes1, true)
	assert.Equal(t, err, nil)
	assert.Equal(t, 0, len(pool.GetLocalTxs()))

}

func Test_transactionPool_UpdateTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog

	stateService := txpoolmock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := txpoolmock.NewMockBlockService(ctrl)
	network := txpoolmock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	pool.TruncateTxPool()

	assert.Equal(t, int64(0), pool.poolCount)

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)

	assert.Equal(t, 2, len(pool.GetLocalTxs()))

	//update failed for low gasPrice
	pool.UpdateTx(Tx5, Key1)
	want1 := true
	want2 := false
	_, got1 := pool.Get(Key1)
	_, got2 := pool.Get(Key5)
	assert.EqualValues(t, want1, got1)
	assert.EqualValues(t, want2, got2)

	//updated for higher gasPrice
	pool.UpdateTx(Tx6, Key2)
	want1 = false
	want2 = true
	_, got1 = pool.Get(Key2)
	_, got2 = pool.Get(Key6)
	assert.EqualValues(t, want1, got1)
	assert.EqualValues(t, want2, got2)

}

func Test_transactionPool_PickTxs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog

	stateService := txpoolmock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := txpoolmock.NewMockBlockService(ctrl)
	network := txpoolmock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	pool.TruncateTxPool()

	assert.Equal(t, int64(0), pool.poolCount)

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(Tx3, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	want := make([]*txbasic.Transaction, 0)
	want = append(want, Tx1)
	want = append(want, Tx2)
	want = append(want, Tx3)
	want = append(want, TxR1)
	want = append(want, TxR2)
	got := pool.PickTxs()
	assert.EqualValues(t, want, got)
	pool.TruncateTxPool()
	assert.Equal(t, int64(0), pool.poolCount)
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(Tx4, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	want = make([]*txbasic.Transaction, 0)
	want = append(want, TxR1)
	want = append(want, TxR2)
	got = pool.PickTxs()
	assert.EqualValues(t, want, got)
	assert.Equal(t, int64(5), pool.poolCount)
}

func Test_transactionPool_Size(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog

	stateService := txpoolmock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := txpoolmock.NewMockBlockService(ctrl)
	network := txpoolmock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	pool.TruncateTxPool()

	assert.Equal(t, int64(0), pool.Size())

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	want := Tx1.Size()
	want += Tx2.Size()
	want += TxR1.Size()
	want += TxR2.Size()
	fmt.Println(pool.Size())
	assert.Equal(t, int64(want), pool.Size())
	pool.RemoveTxByKey(Key1, true)
	want -= Tx1.Size()
	assert.Equal(t, int64(want), pool.Size())
	fmt.Println(pool.Size())

}

func Test_transactionPool_Start(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog

	stateService := txpoolmock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := txpoolmock.NewMockBlockService(ctrl)
	network := txpoolmock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	network.EXPECT().RegisterModule(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	pool.TruncateTxPool()

	assert.Equal(t, int64(0), pool.Size())

	if err := pool.Start(sysActor, network); err != nil {
		t.Error("want", nil, "got", err)
	}
}

func Test_transactionPool_Stop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog

	stateService := txpoolmock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := txpoolmock.NewMockBlockService(ctrl)
	network := txpoolmock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	network.EXPECT().UnSubscribe(gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	pool.TruncateTxPool()

	pool.Stop()
	if 1 != 1 {
		t.Error("stop error")
	}
}

func Test_transactionPool_SetTxPoolConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog

	stateService := txpoolmock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := txpoolmock.NewMockBlockService(ctrl)
	network := txpoolmock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	pool.TruncateTxPool()

	newconf := pool.config
	newconf.TxPoolMaxSize = 123456789
	pool.SetTxPoolConfig(newconf)
	want := newconf.TxPoolMaxSize
	got := pool.config.TxPoolMaxSize
	if !assert.Equal(t, want, got) {
		t.Error("want", want, "got", got)
	}
}

func Test_transactionPool_all(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	log := TpiaLog

	stateService := txpoolmock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := txpoolmock.NewMockBlockService(ctrl)

	blockService.EXPECT().GetBlockByHash(tpchaintypes.BlockHash(OldBlock.Head.Hash)).AnyTimes().Return(OldBlock, nil)
	blockService.EXPECT().GetBlockByHash(tpchaintypes.BlockHash(MidBlock.Head.Hash)).AnyTimes().Return(MidBlock, nil)
	blockService.EXPECT().GetBlockByHash(tpchaintypes.BlockHash(NewBlock.Head.Hash)).AnyTimes().Return(NewBlock, nil)
	blockService.EXPECT().GetBlockByNumber(OldBlock.BlockNum()).AnyTimes().Return(OldBlock, nil)
	blockService.EXPECT().GetBlockByNumber(MidBlock.BlockNum()).AnyTimes().Return(MidBlock, nil)
	blockService.EXPECT().GetBlockByNumber(NewBlock.BlockNum()).AnyTimes().Return(NewBlock, nil)

	network := txpoolmock.NewMockNetwork(ctrl)

	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)

	pool.TruncateTxPool()
	defer profile.Start(profile.MemProfile, profile.MemProfileRate(1)).Stop()
	f, _ := os.OpenFile("cpu.pprof", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	defer f.Close()
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	//add a local tx
	pool.AddTx(Tx1, true)
	assert.Equal(t, 1, len(pool.GetLocalTxs()))
	//add a remote tx
	pool.AddTx(TxR1, false)
	assert.Equal(t, 1, len(pool.GetRemoteTxs()))
	//add local txs
	pool.addTxs(txLocals[10:30], true)
	assert.Equal(t, 21, len(pool.GetLocalTxs()))
	//add remote txs
	pool.addTxs(txRemotes[10:30], false)
	assert.Equal(t, 21, len(pool.GetRemoteTxs()))
	//remove a local tx
	pool.RemoveTxByKey(Key1, true)
	assert.Equal(t, 20, len(pool.GetLocalTxs()))
	//remove a remote tx
	pool.RemoveTxByKey(KeyR1, true)
	assert.Equal(t, 20, len(pool.GetRemoteTxs()))
	//remove txHashes
	var localHashes, remoteHashes []txbasic.TxID
	for _, tx := range txLocals[10:20] {
		id, _ := tx.TxID()
		localHashes = append(localHashes, id)
	}
	for _, tx := range txRemotes[10:20] {
		id, _ := tx.TxID()
		remoteHashes = append(remoteHashes, id)
	}
	pool.RemoveTxHashes(localHashes, true)
	assert.Equal(t, 10, len(pool.GetLocalTxs()))
	pool.RemoveTxHashes(remoteHashes, false)
	assert.Equal(t, 10, len(pool.GetRemoteTxs()))
	//update tx
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	assert.Equal(t, 12, len(pool.GetLocalTxs()))
	//update failed for low gasPrice
	pool.UpdateTx(Tx5, Key1)
	want1 := true
	want2 := false
	_, got1 := pool.Get(Key1)
	_, got2 := pool.Get(Key5)
	assert.EqualValues(t, want1, got1)
	assert.EqualValues(t, want2, got2)
	//updated for higher gasPrice
	pool.UpdateTx(Tx6, Key2)
	want1 = false
	want2 = true
	_, got1 = pool.Get(Key2)
	_, got2 = pool.Get(Key6)
	assert.EqualValues(t, want1, got1)
	assert.EqualValues(t, want2, got2)
	//pickTxs 12*local+10*remote
	assert.Equal(t, 22, len(pool.PickTxs()))

	//full pool
	t1 := time.Now()
	fmt.Println("start add local txs")
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		pool.addTxs(txLocals[:10000], true)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		pool.addTxs(txRemotes[:10000], false)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		pool.addTxs(txLocals[9000:20000], true)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		pool.addTxs(txRemotes[9000:20000], false)
	}()
	wg.Wait()
	du := time.Since(t1)
	fmt.Println(du)
	fmt.Println(len(pool.GetLocalTxs()))
	fmt.Println(len(pool.GetRemoteTxs()))
	//tps :40002/0.17 = 240000 tps
	pool.TruncateTxPool()

	t2 := time.Now()
	fmt.Println("start add tx one by one")
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, tx := range txLocals[:10000] {
			pool.addLocal(tx)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, tx := range txRemotes[:10000] {
			pool.addRemote(tx)
		}
	}()
	wg.Wait()
	du2 := time.Since(t2)
	fmt.Println("duration", du2)
	fmt.Println(len(pool.GetLocalTxs()))
	fmt.Println(len(pool.GetRemoteTxs()))
	//tps:20000/0.08 = 250000 tps
	time.Sleep(1 * time.Second) //give time for saving tx to storage,it's done
	//revertBlocks add txs
	pool.TruncateTxPool()
	txs := txLocals[:20]
	pool.addTxs(txs, true)
	pool.addTxs(txRemotes[:20], false)
	assert.Equal(t, 20, len(pool.GetRemoteTxs()))
	var blocks []*tpchaintypes.Block
	blocks = append(blocks, OldBlock)
	blocks = append(blocks, MidBlock)
	blocks = append(blocks, NewBlock)
	pool.chanBlocksRevert <- blocks
	time.Sleep(5 * time.Second)
	assert.Equal(t, 20, len(pool.GetRemoteTxs()))
	//addBlock remove txs
	pool.chanBlockAdded <- OldBlock
	time.Sleep(5 * time.Second)
	assert.Equal(t, 10, len(pool.GetLocalTxs()))
	assert.Equal(t, 10, len(pool.GetRemoteTxs()))

}
