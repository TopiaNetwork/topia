package transactionpool

import (
	"context"
	tpconfig "github.com/TopiaNetwork/topia/configuration"
	txuni "github.com/TopiaNetwork/topia/transaction/universal"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/hashicorp/golang-lru"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/eventhub"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	tpnet "github.com/TopiaNetwork/topia/network"
	"github.com/TopiaNetwork/topia/network/protocol"
	"github.com/TopiaNetwork/topia/service"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

type transactionPool struct {
	nodeId string
	config *tpconfig.TransactionPoolConfig

	poolSize     int64
	poolCount    int64
	pendingSize  int64
	pendingCount int64
	isInRemove   int32
	isPicking    int32

	chanAddTxs     chan *addTxsItem
	chanSortedItem chan *sortedItem

	chanSysShutdown    chan struct{}
	chanBlockAdded     chan *tpchaintypes.Block
	chanBlocksRevert   chan []*tpchaintypes.Block
	chanDelTxsStorage  chan []txbasic.TxID
	chanSaveTxsStorage chan []*wrappedTx

	txServant TransactionPoolServant

	packagedTxIDs *lru.Cache
	pending       *accTxs
	prepareTxs    *accTxs
	pendingNonces *accountNonce
	allWrappedTxs *allLookupTxs

	sortedTxs *sortedTxList
	txCache   *lru.Cache

	wg        sync.WaitGroup
	log       tplog.Logger
	level     tplogcmm.LogLevel
	ctx       context.Context
	handler   *transactionPoolHandler
	marshaler codec.Marshaler
	hasher    tpcmm.Hasher
	mu        sync.RWMutex
}

func GasPriceLessWTX(wTxOrgin *wrappedTx, wTxTarget *wrappedTx) bool {
	txCatOrgin := string(wTxOrgin.Tx.Head.Category)
	txCatTarget := string(wTxTarget.Tx.Head.Category)
	if txCatOrgin != string(txbasic.TransactionCategory_Topia_Universal) {
		return false //for non topia tx category, its order is lower
	}

	if txCatTarget != string(txbasic.TransactionCategory_Topia_Universal) {
		return true //for non topia tx category, its order is lower
	}

	var txUniOrgin txuni.TransactionUniversal
	var txUniTarget txuni.TransactionUniversal
	if err := txUniOrgin.Unmarshal(wTxOrgin.Tx.Data.Specification); err != nil {
		panic("Invalid txUniOrgin")
	}
	if err := txUniTarget.Unmarshal(wTxOrgin.Tx.Data.Specification); err != nil {
		panic("Invalid txUniTarget")
	}

	return txUniOrgin.Head.GasPrice > txUniTarget.Head.GasPrice

}

func NewTransactionPool(nodeID string, ctx context.Context, conf *tpconfig.TransactionPoolConfig,
	level tplogcmm.LogLevel, log tplog.Logger, codecType codec.CodecType, stateQueryService service.StateQueryService,
	blockService service.BlockService, network tpnet.Network) txpooli.TransactionPool {
	confNew := conf.Check()
	poolLog := tplog.CreateModuleLogger(level, "TransactionPool", log)
	pool := &transactionPool{
		nodeId: nodeID,
		config: confNew,
		log:    poolLog,
		level:  level,
		ctx:    ctx,

		//pending:       newAccTxs(),
		//pendingNonces: newAccountNonce(stateQueryService),
		//prepareTxs:    newAccTxs(),
		allWrappedTxs: newAllLookupTxs(),
		sortedTxs:     newSortedTxList(GasPriceLessWTX),

		//chanAddTxs:     make(chan *addTxsItem, ChanAddTxsSize),
		//chanSortedItem: make(chan *sortedItem, ChanAddTxsSize),

		//chanSysShutdown:    make(chan struct{}),
		//chanBlockAdded:     make(chan *tpchaintypes.Block, ChanBlockAddedSize),
		//chanBlocksRevert:   make(chan []*tpchaintypes.Block),
		//chanDelTxsStorage:  make(chan []txbasic.TxID, ChanDelTxsStorage),
		//chanSaveTxsStorage: make(chan []*wrappedTx, ChanSaveTxsStorage),

		marshaler: codec.CreateMarshaler(codecType),
		hasher:    tpcmm.NewBlake2bHasher(0),
		wg:        sync.WaitGroup{},
		mu:        sync.RWMutex{},
	}

	//pool.packagedTxIDs, _ = lru.New(TxCacheSize)
	//pool.txCache, _ = lru.New(TxCacheSize)

	pool.txServant = newTransactionPoolServant(stateQueryService, blockService, network)
	if pool.config.IsLoadCfg {
		pool.loadAndSetPoolConfig(pool.config.PathConf)
	}
	if pool.config.IsLoadTxs {
		pool.LoadLocalTxsData(pool.config.PathTxsStorage)
	}
	pool.loopChanSelect()

	TxMsgSub = &txMsgSubProcessor{txPool: pool, log: pool.log, nodeID: pool.nodeId}
	//subscribe
	pool.txServant.Subscribe(ctx, protocol.SyncProtocolID_Msg,
		true,
		TxMsgSub.Validate)
	poolHandler := NewTransactionPoolHandler(poolLog, pool, TxMsgSub)
	pool.handler = poolHandler
	return pool
}

func (pool *transactionPool) Dispatch(context actor.Context, data []byte) {
	var txPoolMsg TxPoolMessage
	err := pool.marshaler.Unmarshal(data, &txPoolMsg)
	if err != nil {
		pool.log.Errorf("TransactionPool receive invalid data %v", data)
		return
	}

	switch txPoolMsg.MsgType {
	case TxPoolMessage_Tx:
		var msg TxMessage
		err := pool.marshaler.Unmarshal(txPoolMsg.Data, &msg)
		if err != nil {
			pool.log.Errorf("TransactionPool unmarshal msg %d err %v", txPoolMsg.MsgType, err)
			return
		}
		err = pool.processTX(&msg)
		if err != nil {
			return
		}
	default:
		pool.log.Errorf("TransactionPool receive invalid msg %d", txPoolMsg.MsgType)
		return
	}
}

func (pool *transactionPool) processTX(msg *TxMessage) error {
	err := pool.handler.ProcessTx(pool.ctx, msg)
	if err != nil {
		return err

	}
	return nil
}
func (pool *transactionPool) AddTx(tx *txbasic.Transaction, isLocal bool) error {
	if isLocal {
		return pool.addLocal(tx)
	} else {
		return pool.addRemote(tx)
	}
}

func (pool *transactionPool) addLocal(tx *txbasic.Transaction) error {
	err := pool.addLocals([]*txbasic.Transaction{tx})

	return err
}

func (pool *transactionPool) addRemote(tx *txbasic.Transaction) error {
	err := pool.addRemotes([]*txbasic.Transaction{tx})
	return err
}
func (pool *transactionPool) addLocals(txs []*txbasic.Transaction) error {
	return pool.addTxs(txs, true)
}

func (pool *transactionPool) addRemotes(txs []*txbasic.Transaction) error {
	return pool.addTxs(txs, false)
}
func (pool *transactionPool) addTx(tx *txbasic.Transaction, isLocal bool) error {
	err := pool.addTxs([]*txbasic.Transaction{tx}, isLocal)
	return err
}
func (pool *transactionPool) addTxs(txs []*txbasic.Transaction, isLocal bool) error {
	if len(txs) == 0 {
		return ErrTxIsNil
	}
	/*if atomic.LoadInt32(&pool.isInRemove) == int32(1) || atomic.LoadInt32(&pool.isPicking) == int32(1) {
		pool.chanAddTxs <- &addTxsItem{
			txs:     txs,
			isLocal: isLocal,
		}
	}*/
	pool.mu.Lock()
	defer pool.mu.Unlock()

	if pool.Count()+int64(len(txs)) > pool.config.TxPoolMaxCnt {
		for _, tx := range txs {
			id, _ := tx.TxID()
			pool.txCache.Add(id, txpooli.StateDroppedForTxPoolFull)
		}
		return ErrTxPoolFull
	}
	curHeight, err := pool.txServant.CurrentHeight()
	if err != nil {
		return err
	}

	//var newTxs []*txbasic.Transaction
	for _, tx := range txs {
		chainNonce, _ := pool.txServant.GetNonce(tpcrtypes.Address(tx.Head.FromAddr))
		if tx.Head.Nonce <= chainNonce {
			continue
		}
		txID, _ := tx.TxID()

		//if _, ok := pool.allWrappedTxs.Get(txID); ok {
		//	continue
		//} else {
		txInfo := &wrappedTx{
			TxID:          txID,
			IsLocal:       isLocal,
			Category:      txbasic.TransactionCategory(tx.Head.Category),
			LastTime:      time.Now(),
			LastHeight:    curHeight,
			TxState:       txpooli.StateTxAdded,
			IsRepublished: false,
			FromAddr:      tpcrtypes.Address(tx.Head.FromAddr),
			Nonce:         tx.Head.Nonce,
			Tx:            tx,
		}

		//pool.allWrappedTxs.Set(txID, txInfo)
		pool.sortedTxs.addTx(txInfo)
		//pool.txCache.Add(txID, txpooli.StateTxAdded)
		//go func(wTx *wrappedTx) {
		//	pool.chanSaveTxsStorage <- []*wrappedTx{wTx}
		//}(txInfo)
		//}
		//newTxs = append(newTxs, tx)
	}
	//if len(newTxs) == 0 {
	//	return ErrNoTxAdded
	//}
	/*getPendingNonce := func(addr tpcrtypes.Address) (uint64, error) {
		return pool.pendingNonces.get(addr)
	}
	setPendingNonce := func(addr tpcrtypes.Address, nonce uint64) {
		pool.pendingNonces.set(addr, nonce)
	}
	isPackaged := func(id txbasic.TxID) bool {
		_, ok := pool.packagedTxIDs.Get(id)
		return ok
	}
	dropOldTx := func(id txbasic.TxID) {
		pool.allWrappedTxs.Del(id)
		pool.txCache.Remove(id)
		pool.chanDelTxsStorage <- []txbasic.TxID{id}
	}*/

	/*addSize := func(pendingCnt, pendingSize, poolCnt, poolSize int64) {
		if pendingCnt != int64(0) {
			atomic.AddInt64(&pool.pendingCount, pendingCnt)
		}
		if pendingSize != int64(0) {
			atomic.AddInt64(&pool.pendingSize, pendingSize)
		}
		if poolCnt != int64(0) {
			for {
				old := atomic.LoadInt64(&pool.poolCount)
				if atomic.CompareAndSwapInt64(&pool.poolCount, old, old+poolCnt) {
					break
				}
			}
		}
		if poolSize != int64(0) {
			atomic.AddInt64(&pool.poolSize, poolSize)
		}
	}
	fetchTxsPrepared := func(address tpcrtypes.Address, nonce uint64) []*txbasic.Transaction {
		return pool.prepareTxs.fetchTxsToPending(address, nonce)
	}
	insertToPrepared := func(address tpcrtypes.Address, isLocal bool, tx *txbasic.Transaction,
		dropOldTx func(id txbasic.TxID), addTxInfo func(isLocal bool, tx *txbasic.Transaction), addSize func(pendingCnt, pendingSize, poolCnt, poolSize int64)) {
		pool.prepareTxs.addTxToPrepared(address, isLocal, tx, dropOldTx, addTxInfo, addSize)
	}

	pool.pending.addTxsToPending(newTxs, isLocal, getPendingNonce, setPendingNonce, isPackaged, dropOldTx, addTxInfo,
		addSize, addIntoSorted, fetchTxsPrepared, insertToPrepared)*/

	return nil
}

func (pool *transactionPool) UpdateTx(tx *txbasic.Transaction, txID txbasic.TxID) error {
	if atomic.LoadInt32(&pool.isInRemove) == int32(1) || atomic.LoadInt32(&pool.isPicking) == int32(1) {
		pool.chanAddTxs <- &addTxsItem{
			txs:     []*txbasic.Transaction{tx},
			isLocal: false,
		}
	}
	txOldWrapped, ok := pool.allWrappedTxs.Get(txID)
	if !ok {
		return ErrTxNotExist
	}

	if tpcrtypes.Address(tx.Head.FromAddr) == txOldWrapped.FromAddr &&
		tx.Head.Nonce == txOldWrapped.Nonce &&
		GasPrice(txOldWrapped.Tx) < GasPrice(tx) {

		if _, ok := pool.packagedTxIDs.Get(txID); ok {
			return ErrTxIsPackaged
		}

		err := pool.addTx(tx, true)

		if err != nil {
			return err
		}

		//commit when unit testing
		//**********
		//txRemoved := &eventhub.TxPoolEvent{
		//	EvType: eventhub.TxPoolEVType_Removed,
		//	Tx:     txOldWrapped.Tx,
		//}
		//eventhub.GetEventHubManager().GetEventHub(pool.nodeId).Trig(pool.ctx, eventhub.EventName_TxPoolChanged, txRemoved)
		////commit when unit testing
		//txAdded := &eventhub.TxPoolEvent{
		//	EvType: eventhub.TxPoolEVType_Received,
		//	Tx:     txOldWrapped.Tx,
		//}
		//eventhub.GetEventHubManager().GetEventHub(pool.nodeId).Trig(pool.ctx, eventhub.EventName_TxPoolChanged, txAdded)
		//***********
	}
	return nil
}

func (pool *transactionPool) RemoveTxByKey(txID txbasic.TxID) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	/*wTx, isExist := pool.allWrappedTxs.Get(txID)
	if !isExist {
		return ErrTxNotExist
	}*/

	//pool.allWrappedTxs.Del(txID)
	pool.sortedTxs.removeTx(txID)
	//pool.txCache.Remove(txID)

	//go func(txIDT txbasic.TxID) {
	//	pool.chanDelTxsStorage <- []txbasic.TxID{txIDT}
	//}(txID)
	////commit when unit testing
	//*******************
	//txRemoved := &eventhub.TxPoolEvent{
	//	EvType: eventhub.TxPoolEVType_Removed,
	//	Tx:     txOldWrapped.Tx,
	//}
	//eventhub.GetEventHubManager().GetEventHub(pool.nodeId).Trig(pool.ctx, eventhub.EventName_TxPoolChanged, txRemoved)
	//*******************
	return nil
}

func (pool *transactionPool) RemoveTxBatch(hashes []txbasic.TxID) []error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	defer func(t0 time.Time) {
		pool.log.Infof(pool.nodeId, "transaction pool RemoveTxBatch cost time:", time.Since(t0))
	}(time.Now())
	var errs []error
	for {
		old := atomic.LoadInt32(&pool.isInRemove)
		if atomic.CompareAndSwapInt32(&pool.isInRemove, old, int32(1)) {
			break
		}
	}
	for {
		old := atomic.LoadInt32(&pool.isPicking)
		if atomic.CompareAndSwapInt32(&pool.isPicking, old, int32(1)) {
			break
		}
	}

	if len(hashes) == 0 {
		errs = append(errs, ErrTxIsNil)
		for {
			old := atomic.LoadInt32(&pool.isInRemove)
			if atomic.CompareAndSwapInt32(&pool.isInRemove, old, int32(0)) {
				break
			}
		}
		for {
			old := atomic.LoadInt32(&pool.isPicking)
			if atomic.CompareAndSwapInt32(&pool.isPicking, old, int32(0)) {
				break
			}
		}

		return errs
	}
	for _, hash := range hashes {
		if err := pool.RemoveTxByKey(hash); err != nil {
			errs = append(errs, err)
		}
	}
	for {
		old := atomic.LoadInt32(&pool.isInRemove)
		if atomic.CompareAndSwapInt32(&pool.isInRemove, old, int32(0)) {
			break
		}
	}
	for {
		old := atomic.LoadInt32(&pool.isPicking)
		if atomic.CompareAndSwapInt32(&pool.isPicking, old, int32(0)) {
			break
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func (pool *transactionPool) Get(txID txbasic.TxID) (*txbasic.Transaction, bool) {
	wTx, ok := pool.allWrappedTxs.Get(txID)
	if ok {
		return wTx.Tx, true
	}
	return nil, false
}

func (pool *transactionPool) Size() int64 {
	return atomic.LoadInt64(&pool.poolSize)
}
func (pool *transactionPool) PendingAccountTxCnt(address tpcrtypes.Address) int64 {
	return pool.pending.accountTxCnt(address)
}

func (pool *transactionPool) Count() int64 {
	return atomic.LoadInt64(&pool.poolCount)
}

func (pool *transactionPool) Start(sysActor *actor.ActorSystem, network tpnet.Network) error {
	actorPID, err := CreateTransactionPoolActor(pool.level, pool.log, sysActor, pool)
	if err != nil {
		pool.log.Panicf("CreateTransactionPoolActor error: %v", err)
		return err
	}
	network.RegisterModule(txpooli.MOD_NAME, actorPID, pool.marshaler)

	ObsID, err = eventhub.GetEventHubManager().GetEventHub(pool.nodeId).
		Observe(pool.ctx, eventhub.EventName_BlockAdded, pool.handler.processBlockAddedEvent)
	if err != nil {
		pool.log.Panicf("processBlockAddedEvent error:%s", err)
	}

	pool.log.Infof("processBlockAddedEvent,obsID:%s", ObsID)

	return nil
}

func (pool *transactionPool) Stop() {

	// Unsubscribe subscriptions registered from blockchain
	pool.txServant.UnSubscribe(protocol.SyncProtocolID_Msg)
	eventhub.GetEventHubManager().GetEventHub(pool.nodeId).UnObserve(pool.ctx, ObsID, eventhub.EventName_BlockAdded)
	pool.log.Info("TransactionPool stopped")
}
func (pool *transactionPool) SysShutDown() {
	pool.chanSysShutdown <- struct{}{}

}

func (pool *transactionPool) PickTxs() []*txbasic.Transaction {
	defer func(t0 time.Time) {
		pool.log.Infof("PickTxs cost time: %d", time.Since(t0))
	}(time.Now())

	pool.mu.Lock()
	defer pool.mu.Unlock()

	return pool.sortedTxs.PickTxs(pool.config.BlockMaxBytes, pool.config.BlockMaxGas)

}
func (pool *transactionPool) GetLocalTxs() []*txbasic.Transaction {
	var txs []*txbasic.Transaction
	fetchTx := func(k interface{}, v interface{}) {
		txs = append(txs, v.(*wrappedTx).Tx)
	}
	pool.allWrappedTxs.localTxs.IterateCallback(fetchTx)
	return txs
}
func (pool *transactionPool) GetRemoteTxs() []*txbasic.Transaction {
	var txs []*txbasic.Transaction
	fetchTx := func(k interface{}, v interface{}) {
		txs = append(txs, v.(*wrappedTx).Tx)
	}
	pool.allWrappedTxs.remoteTxs.IterateCallback(fetchTx)
	return txs
}

func (pool *transactionPool) PendingOfAddress(addr tpcrtypes.Address) ([]*txbasic.Transaction, error) {
	return pool.pending.getTxsByAddr(addr)
}

func (pool *transactionPool) PeekTxState(txid txbasic.TxID) txpooli.TransactionState {
	value, ok := pool.txCache.Peek(txid)
	if ok {
		return value.(txpooli.TransactionState)
	} else {
		return txpooli.StateTxNil
	}
}
