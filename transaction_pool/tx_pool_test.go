package transactionpool

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
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
	"github.com/TopiaNetwork/topia/transaction_pool/mock"
)

var (
	Category1, Category2                                                             txbasic.TransactionCategory
	TestTxPoolConfig                                                                 txpooli.TransactionPoolConfig
	TxlowGasPrice, TxHighGasLimit, txlocal, txremote, Tx1, Tx2, Tx3, Tx4, TxR1, TxR2 *txbasic.Transaction
	txHead                                                                           *txbasic.TransactionHead
	txData                                                                           *txbasic.TransactionData
	keylocal, keyremote, Key1, Key2, Key3, Key4, KeyR1, KeyR2                        txbasic.TxID
	from, to, From1, From2                                                           tpcrtypes.Address
	txLocals, txRemotes                                                              []*txbasic.Transaction
	keyLocals, keyRemotes                                                            []txbasic.TxID
	fromLocals, fromRemotes, toLocals, toRemotes                                     []tpcrtypes.Address

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
	for i := 1; i <= 200; i++ {
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

		txremote = setTxRemote(nonce, gasprice, gaslimit)
		txremote.Head.TimeStamp = starttime + uint64(i)
		if i > 1 {
			txremote.Head.FromAddr = append(txremote.Head.FromAddr, byte(i))
		}
		keyremote, _ = txremote.TxID()
		keyRemotes = append(keyRemotes, keyremote)
		txRemotes = append(txRemotes, txremote)
	}
	Tx1 = setTxLocal(2, 11000, 123456)
	Tx2 = setTxLocal(3, 12000, 123456)

	Tx3 = setTxLocal(4, 10000, 123456)
	Tx4 = setTxLocal(5, 14000, 123456)

	From1 = tpcrtypes.Address(Tx1.Head.FromAddr)
	Key1, _ = Tx1.TxID()
	Key2, _ = Tx2.TxID()
	Key3, _ = Tx3.TxID()
	Key4, _ = Tx4.TxID()

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

	OldTxs = txLocals[10:20]
	MidTxs = txLocals[20:30]
	NewTxs = txLocals[30:40]

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
		FromAddr:  []byte{0x02, 0x02, 0x02},
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
	conf.MaxSizeOfEachPendingAccount = 16 * 32 * 1024
	conf.MaxSizeOfPending = 128 * 32 * 1024
	conf.MaxSizeOfEachQueueAccount = 32 * 32 * 1024
	conf.MaxSizeOfQueue = 256 * 32 * 1024
	conf = (&conf).Check()
	poolLog := tplog.CreateModuleLogger(level, "TransactionPool", log)
	pool := &transactionPool{
		nodeId:              nodeID,
		config:              conf,
		log:                 poolLog,
		level:               level,
		ctx:                 ctx,
		allTxsForLook:       newAllTxsLookupMap(),
		activationIntervals: newActivationInterval(),
		heightIntervals:     newHeightInterval(),
		txHashCategory:      newTxHashCategory(),
		chanBlockAdded:      make(chan BlockAddedEvent, ChanBlockAddedSize),
		chanReqReset:        make(chan *txPoolResetHeads),
		chanReqTurn:         make(chan *accountCache),
		chanReorgDone:       make(chan chan struct{}),
		chanReorgShutdown:   make(chan struct{}), // requests shutdown of scheduleReorgLoop
		chanRmTxs:           make(chan []txbasic.TxID),

		marshaler: codec.CreateMarshaler(codecType),
		hasher:    tpcmm.NewBlake2bHasher(0),
	}
	pool.txCache, _ = lru.New(pool.config.TxCacheSize)
	pool.pendings = newPendingsMap()
	pool.queues = newQueuesMap()
	pool.allTxsForLook = newAllTxsLookupMap()
	pool.txServant = newTransactionPoolServant(stateQueryService, blockService, network)

	curBlock, err := pool.txServant.GetLatestBlock()
	if err != nil {
		pool.log.Errorf("NewTransactionPool get current block error:", err)
	}
	pool.reset(nil, curBlock.GetHead())
	pool.wg.Add(1)
	go pool.ReorgAndTurnTxLoop()

	pool.loopChanSelect()
	TxMsgSubProcessor = &txMessageSubProcessor{txpool: pool, log: pool.log, nodeID: pool.nodeId}
	//comment when unit tesing
	//pool.txServant.Subscribe(ctx, protocol.SyncProtocolID_Msg,
	//	true,
	//	TxMsgSubProcessor.Validate)
	poolHandler := NewTransactionPoolHandler(poolLog, pool, TxMsgSubProcessor)
	pool.handler = poolHandler

	pool.LoadTxsData(pool.config.PathTxsStorge)

	return pool

}

func Test_transactionPool_GetLocalTxs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog
	stateService := mock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)
	blockService := mock.NewMockBlockService(ctrl)
	network := mock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)

	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	txs := make([]*txbasic.Transaction, 0)
	txs = append(txs, Tx1)
	txs = append(txs, Tx2)
	txsMap := make(map[tpcrtypes.Address][]*txbasic.Transaction)
	txsMap[From1] = txs
	fmt.Println("test 001")
	pool.AddLocals(txs)
	want := txsMap[From1]
	fmt.Println("test 002")

	got := pool.GetLocalTxs(Category1)
	fmt.Println("test 003")

	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.queues.getLenTxsByAddrOfCategory(Category1, From1))
	assert.Equal(t, 1, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 2, len(pool.pendings.getTxsByCategory(Category1)))

	if !reflect.DeepEqual(want, got) {
		t.Errorf("want:%v\n,                 got:%v\n", want, got)
	}
}

func Test_transactionPool_AddTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog
	stateService := mock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)
	blockService := mock.NewMockBlockService(ctrl)

	network := mock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	//add one local tx
	pool.AddTx(Tx1, true)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 1, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 1, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	// add one remote tx
	pool.AddTx(TxR1, false)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 2, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 1, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 1, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	// add one local tx
	pool.AddTx(Tx2, true)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.queues.getLenTxsByAddrOfCategory(Category1, From1))
	assert.Equal(t, 0, pool.queues.getLenTxsByAddrOfCategory(Category1, From2))
	assert.Equal(t, 3, len(pool.pendings.getTxsByCategory(Category1)))
	assert.Equal(t, 2, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 1, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	// add one remote tx
	pool.AddTx(TxR2, false)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.queues.getLenTxsByAddrOfCategory(Category1, From1))
	assert.Equal(t, 0, pool.queues.getLenTxsByAddrOfCategory(Category1, From2))
	assert.Equal(t, 4, len(pool.pendings.getTxsByCategory(Category1)))
	assert.Equal(t, 2, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 2, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	// add a same tx
	pool.AddTx(Tx2, true)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.queues.getLenTxsByAddrOfCategory(Category1, From1))
	assert.Equal(t, 0, pool.queues.getLenTxsByAddrOfCategory(Category1, From2))
	assert.Equal(t, 4, len(pool.pendings.getTxsByCategory(Category1)))
	assert.Equal(t, 2, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 2, pool.allTxsForLook.getRemoteCountByCategory(Category1))

}
func Test_transactionPool_RemoveTxByKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog
	stateService := mock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := mock.NewMockBlockService(ctrl)
	network := mock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))

	pool.AddTx(Tx1, true)
	pool.RemoveTxByKey(Key1)

	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))

	pool.AddTx(TxR1, false)
	pool.RemoveTxByKey(KeyR1)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))

}

func Test_transactionPool_RemoveTxHashs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog
	stateService := mock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := mock.NewMockBlockService(ctrl)
	network := mock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 2, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 2, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 2, pool.allTxsForLook.getRemoteCountByCategory(Category1))

	hashs := make([]txbasic.TxID, 0)
	hashs = append(hashs, Key1)
	hashs = append(hashs, Key2)
	hashs = append(hashs, KeyR1)
	hashs = append(hashs, KeyR2)
	pool.RemoveTxHashs(hashs)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))

}

func Test_transactionPool_UpdateTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog
	stateService := mock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := mock.NewMockBlockService(ctrl)
	network := mock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	assert.Equal(t, 0, pool.queues.getLenTxsByAddrOfCategory(Category1, From1))
	assert.Equal(t, 2, len(pool.pendings.getTxsByAddrOfCategory(Category1, From1)))
	assert.Equal(t, 2, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))

	//update failed for low gasprice
	pool.UpdateTx(Tx3, Key1)
	want := make([]txbasic.TxID, 0)
	got := make([]txbasic.TxID, 0)
	want = append(want, Key1)
	want = append(want, Key2)
	for _, tx := range pool.pendings.getTxsByAddrOfCategory(Category1, From1) {
		txid, _ := tx.TxID()
		got = append(got, txid)
	}
	assert.EqualValues(t, want, got)
	//updated for higher gasprice
	pool.UpdateTx(Tx4, Key2)
	want = make([]txbasic.TxID, 0)
	got = make([]txbasic.TxID, 0)
	want = append(want, Key1)
	want = append(want, Key4)
	for _, tx := range pool.pendings.getTxsByAddrOfCategory(Category1, From1) {
		txid, _ := tx.TxID()
		got = append(got, txid)
	}
	assert.EqualValues(t, want, got)
}

func Test_transactionPool_PickTxs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog
	stateService := mock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := mock.NewMockBlockService(ctrl)
	network := mock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(Tx4, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	want := make([]*txbasic.Transaction, 0)
	got := make([]*txbasic.Transaction, 0)
	want = append(want, Tx1)
	want = append(want, Tx2)
	want = append(want, Tx4)
	want = append(want, TxR1)
	want = append(want, TxR2)
	pending := pool.PickTxs()

	for _, tx := range pending {
		got = append(got, tx)
	}
	assert.EqualValues(t, want, got)

}

func Test_transactionPool_Get(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog
	stateService := mock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := mock.NewMockBlockService(ctrl)
	network := mock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))

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

func Test_transactionPool_Count(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog
	stateService := mock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := mock.NewMockBlockService(ctrl)
	network := mock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	want := 4
	if got := pool.allTxsForLook.getCountFromAllTxsLookupByCategory(Category1); want != got {
		t.Error("want", want, "got", got)
	}
}
func Test_transactionPool_Size(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog
	stateService := mock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := mock.NewMockBlockService(ctrl)
	network := mock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	want := Tx1.Size()
	want += Tx2.Size()
	want += TxR1.Size()
	want += TxR2.Size()
	if got := pool.allTxsForLook.getSizeFromAllTxsLookupByCategory(Category1); int64(want) != got {
		t.Error("want", want, "got", got)
	}
}
func Test_transactionPool_Start(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog
	stateService := mock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := mock.NewMockBlockService(ctrl)
	network := mock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))

	if err := pool.Start(sysActor, network); err != nil {
		t.Error("want", nil, "got", err)
	}
}

func Test_transactionPool_Stop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog
	stateService := mock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := mock.NewMockBlockService(ctrl)
	network := mock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	pool.Stop()
	if 1 != 1 {
		t.Error("stop error")
	}
}
