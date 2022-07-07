package transactionpool

import (
	"context"
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
	nodeId    string
	config    txpooli.TransactionPoolConfig
	poolSize  int64
	poolCount int64

	chanSysShutdown    chan struct{}
	chanBlockAdded     chan *tpchaintypes.Block
	chanBlocksRevert   chan []*tpchaintypes.Block
	chanRmTxs          chan []txbasic.TxID
	chanDelTxsStorage  chan []txbasic.TxID
	chanSaveTxsStorage chan []*wrappedTx

	txServant TransactionPoolServant

	packagedTxs   *packagedTxs
	pending       *pendingTxList
	allWrappedTxs *allLookupTxs
	txCache       *lru.Cache

	wg        sync.WaitGroup
	log       tplog.Logger
	level     tplogcmm.LogLevel
	ctx       context.Context
	handler   *transactionPoolHandler
	marshaler codec.Marshaler
	hasher    tpcmm.Hasher
}

func NewTransactionPool(nodeID string, ctx context.Context, conf txpooli.TransactionPoolConfig,
	level tplogcmm.LogLevel, log tplog.Logger, codecType codec.CodecType, stateQueryService service.StateQueryService,
	blockService service.BlockService, network tpnet.Network) txpooli.TransactionPool {
	conf = (&conf).Check()
	poolLog := tplog.CreateModuleLogger(level, "TransactionPool", log)
	pool := &transactionPool{
		nodeId: nodeID,
		config: conf,
		log:    poolLog,
		level:  level,
		ctx:    ctx,

		pending:       newPending(),
		allWrappedTxs: newAllLookupTxs(),
		packagedTxs:   newPackagedTxs(),

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
		err = pool.processTx(&msg)
		if err != nil {
			return
		}
	default:
		pool.log.Errorf("TransactionPool receive invalid msg %d", txPoolMsg.MsgType)
		return
	}
}
func (pool *transactionPool) processTx(msg *TxMessage) error {
	err := pool.handler.ProcessTx(pool.ctx, msg)
	if err != nil {
		return err

	}
	return nil
}
func (pool *transactionPool) AddTx(tx *txbasic.Transaction, local bool) error {
	if local {
		return pool.addLocal(tx)
	} else {
		return pool.addRemote(tx)
	}
	return nil
}

func (pool *transactionPool) addLocal(tx *txbasic.Transaction) error {
	errs := pool.addLocals([]*txbasic.Transaction{tx})
	return errs[0]
}

func (pool *transactionPool) addRemote(tx *txbasic.Transaction) error {
	errs := pool.addRemotes([]*txbasic.Transaction{tx})
	return errs[0]
}
func (pool *transactionPool) addLocals(txs []*txbasic.Transaction) []error {
	return pool.addTxs(txs, true)
}

func (pool *transactionPool) addRemotes(txs []*txbasic.Transaction) []error {
	return pool.addTxs(txs, false)
}
func (pool *transactionPool) addTxs(txs []*txbasic.Transaction, isLocal bool) []error {
	errs := make([]error, len(txs))
	if len(txs) == 1 {
		err := pool.addTx(txs[0], isLocal)
		if err != nil {
			errs = append(errs, err)
		}
		return errs
	}

	if pool.Count()+int64(len(txs)) > pool.config.TxPoolMaxCnt {
		errs = append(errs, ErrTxPoolFull)
		return errs
	}

	curHeight, err := pool.txServant.CurrentHeight()
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	var newTxs []*txbasic.Transaction
	for _, tx := range txs {
		txID, _ := tx.TxID()
		if _, ok := pool.allWrappedTxs.Get(txID); ok {
			continue
		}
		newTxs = append(newTxs, tx)
	}
	dropOldTxID, dropOldSize, dropNewTxs := pool.pending.addTxsToPending(newTxs, pool.packagedTxs.isPackaged, isLocal)

	if len(dropOldTxID) > 0 {
		for _, dropID := range dropOldTxID {
			pool.txCache.Remove(dropID)
			pool.allWrappedTxs.Del(dropID)
		}
		pool.chanDelTxsStorage <- dropOldTxID
		//pool.DelLocalTxsData(pool.config.PathTxsStorage, dropOldTxID)
		atomic.AddInt64(&pool.poolSize, -dropOldSize)
		atomic.AddInt64(&pool.poolCount, -int64(len(dropOldTxID)))
	}

	addTxs := TxDifferenceList(newTxs, dropNewTxs)
	if len(addTxs) == 0 {
		return nil
	}
	var wrappedTxs []*wrappedTx
	var addTxsSize int64
	for _, tx := range addTxs {
		txID, _ := tx.TxID()
		pool.txCache.Add(txID, txpooli.StateTxAdded)
		wrappedTx := &wrappedTx{
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
		addTxsSize += int64(tx.Size())
		pool.allWrappedTxs.Set(txID, wrappedTx)
		wrappedTxs = append(wrappedTxs, wrappedTx)
	}
	atomic.AddInt64(&pool.poolSize, addTxsSize)
	atomic.AddInt64(&pool.poolCount, int64(len(addTxs)))
	if isLocal {
		pool.chanSaveTxsStorage <- wrappedTxs
		//pool.SaveLocalTxsData(pool.config.PathTxsStorage, wrappedTxs)
	}
	return errs
}

func (pool *transactionPool) addTx(tx *txbasic.Transaction, isLocal bool) error {
	if pool.Count() >= pool.config.TxPoolMaxCnt {
		return ErrTxPoolFull
	}
	txID, _ := tx.TxID()
	curHeight, err := pool.txServant.CurrentHeight()
	if err != nil {
		return err
	}

	if _, ok := pool.allWrappedTxs.Get(txID); ok {
		return ErrAlreadyKnown
	}
	wTx := &wrappedTx{
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
	replaced, oldID, oldSize := pool.pending.addTxToPending(tx, pool.packagedTxs.isPackaged, isLocal)

	if replaced {
		pool.txCache.Remove(oldID)
		pool.allWrappedTxs.Del(oldID)
		if isLocal {
			pool.chanDelTxsStorage <- []txbasic.TxID{oldID}
			//pool.DelLocalTxData(pool.config.PathTxsStorage, oldID)
		}
		atomic.AddInt64(&pool.poolSize, -int64(oldSize))
		atomic.AddInt64(&pool.poolCount, -1)
	}
	pool.txCache.Add(txID, txpooli.StateTxAdded)
	pool.allWrappedTxs.Set(txID, wTx)
	atomic.AddInt64(&pool.poolSize, int64(tx.Size()))
	atomic.AddInt64(&pool.poolCount, 1)
	if isLocal {
		pool.chanSaveTxsStorage <- []*wrappedTx{wTx}
		//pool.SaveLocalTxData(pool.config.PathTxsStorage, wTx)
	}

	return nil
}

func (pool *transactionPool) UpdateTx(tx *txbasic.Transaction, txID txbasic.TxID) error {
	txOldWrapped, ok := pool.allWrappedTxs.Get(txID)
	if !ok {
		return ErrTxNotExist
	}

	if tpcrtypes.Address(tx.Head.FromAddr) == txOldWrapped.FromAddr && tx.Head.Nonce == txOldWrapped.Nonce && GasPrice(txOldWrapped.Tx) < GasPrice(tx) {
		ok, err := pool.pending.isNonceContinuous(txOldWrapped.FromAddr)
		if err != nil {
			return err
		}
		if !ok {
			return ErrTxsNotContinuous
		}
		if pool.packagedTxs.isPackaged(txID) {
			return ErrTxIsPackaged
		}
		err = pool.RemoveTxByKey(txID, true)
		if err != nil {
			return err
		}

		err = pool.addTx(tx, true)

		if err != nil {
			return err
		}

		//commit when unit testing
		//**********
		txRemoved := &eventhub.TxPoolEvent{
			EvType: eventhub.TxPoolEVType_Removed,
			Tx:     txOldWrapped.Tx,
		}
		eventhub.GetEventHubManager().GetEventHub(pool.nodeId).Trig(pool.ctx, eventhub.EventName_TxPoolChanged, txRemoved)
		//commit when unit testing
		txAdded := &eventhub.TxPoolEvent{
			EvType: eventhub.TxPoolEVType_Received,
			Tx:     txOldWrapped.Tx,
		}
		eventhub.GetEventHubManager().GetEventHub(pool.nodeId).Trig(pool.ctx, eventhub.EventName_TxPoolChanged, txAdded)
		//***********
	}
	return nil
}

func (pool *transactionPool) callBackRemoveTxByKey(txID txbasic.TxID, force bool, isLocal bool) error {
	txOldWrapped, ok := pool.allWrappedTxs.Get(txID)
	if !ok {
		return ErrTxNotExist
	}
	err := pool.pending.removeTx(txOldWrapped.FromAddr, txOldWrapped.TxID, txOldWrapped.Nonce, force)
	if err != nil {
		return err
	}
	atomic.AddInt64(&pool.poolSize, -int64(txOldWrapped.Tx.Size()))
	atomic.AddInt64(&pool.poolCount, -1)
	pool.txCache.Remove(txID)
	pool.allWrappedTxs.callBackDel(txID, isLocal)
	////commit when unit testing
	//***********
	txRemoved := &eventhub.TxPoolEvent{
		EvType: eventhub.TxPoolEVType_Removed,
		Tx:     txOldWrapped.Tx,
	}
	eventhub.GetEventHubManager().GetEventHub(pool.nodeId).Trig(pool.ctx, eventhub.EventName_TxPoolChanged, txRemoved)
	//********
	return nil
}

func (pool *transactionPool) RemoveTxByKey(txID txbasic.TxID, force bool) error {
	txOldWrapped, ok := pool.allWrappedTxs.Get(txID)
	if !ok {
		return ErrTxNotExist
	}
	err := pool.pending.removeTx(txOldWrapped.FromAddr, txOldWrapped.TxID, txOldWrapped.Nonce, force)
	if err != nil {
		return err
	}
	atomic.AddInt64(&pool.poolSize, -int64(txOldWrapped.Tx.Size()))
	atomic.AddInt64(&pool.poolCount, -1)
	pool.txCache.Remove(txID)
	pool.allWrappedTxs.Del(txID)
	pool.chanDelTxsStorage <- []txbasic.TxID{txID}
	////commit when unit testing
	//*******************
	txRemoved := &eventhub.TxPoolEvent{
		EvType: eventhub.TxPoolEVType_Removed,
		Tx:     txOldWrapped.Tx,
	}
	eventhub.GetEventHubManager().GetEventHub(pool.nodeId).Trig(pool.ctx, eventhub.EventName_TxPoolChanged, txRemoved)
	//*******************

	return nil
}

func (pool *transactionPool) RemoveTxHashes(hashes []txbasic.TxID, force bool) error {
	defer func(t0 time.Time) {
		pool.log.Infof("transaction pool RemoveTxHashes cost time:", time.Since(t0))
	}(time.Now())
	if force {
		for _, hash := range hashes {
			err := pool.RemoveTxByKey(hash, force)
			if err != nil {
				pool.log.Infof("transaction pool RemoveTxHashes err:", err, "hash", hash)
			}
		}
		return nil
	} else {
		removeCache := func(id txbasic.TxID) { pool.txCache.Remove(id) }
		removeLookup := func(id txbasic.TxID) { pool.allWrappedTxs.Del(id) }
		err, dropCnt, dropSize := pool.checkAndRemoveHashes(hashes, removeCache, removeLookup)
		if err != nil {
			return err
		}
		atomic.AddInt64(&pool.poolCount, -dropCnt)
		atomic.AddInt64(&pool.poolSize, -dropSize)
	}

	return nil
}

func (pool *transactionPool) checkAndRemoveHashes(hashes []txbasic.TxID, removeCache func(id txbasic.TxID), removeLookup func(id txbasic.TxID)) (errs error, dropCnt, dropSize int64) {
	accounts := make(map[tpcrtypes.Address][]*txbasic.Transaction, len(hashes))
	for _, hash := range hashes {
		txOldWrapped, ok := pool.allWrappedTxs.Get(hash)
		if !ok {
			return ErrTxNotExist, 0, 0
		}
		if _, ok := accounts[txOldWrapped.FromAddr]; !ok {
			if _, err := pool.pending.isNonceContinuous(txOldWrapped.FromAddr); err != nil {
				return err, 0, 0
			}
			accounts[txOldWrapped.FromAddr] = []*txbasic.Transaction{txOldWrapped.Tx}
		} else {
			list := accounts[txOldWrapped.FromAddr]
			list = append(list, txOldWrapped.Tx)
			accounts[txOldWrapped.FromAddr] = list
		}
	}
	for addr, list := range accounts {
		Cnt, Size := pool.pending.removeTxs(addr, list, removeCache, removeLookup)
		dropCnt += Cnt
		dropSize += Size
	}

	return nil, dropCnt, dropSize
}

func (pool *transactionPool) Get(txID txbasic.TxID) (*txbasic.Transaction, bool) {
	wrappedTx, ok := pool.allWrappedTxs.Get(txID)
	if ok {
		return wrappedTx.Tx, true
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

func (pool *transactionPool) PickTxs() (txs []*txbasic.Transaction) {
	defer func(t0 time.Time) {
		pool.log.Infof("PickTxs cost time:", time.Since(t0))
	}(time.Now())

	txs = make([]*txbasic.Transaction, 0)
	txType := PickTxPending
	switch txType {
	case PickTxPending:
		packageTxs := func(list []*txbasic.Transaction) {
			pool.packagedTxs.purge()
			pool.packagedTxs.putTxs(list)
		}
		lastNonce := pool.txServant.GetNonce
		txs, dropCnt, dropSize := pool.pending.getAllCommitTxs(lastNonce, packageTxs)
		atomic.AddInt64(&pool.poolCount, -int64(dropCnt))
		atomic.AddInt64(&pool.poolSize, -int64(dropSize))
		return txs
	default:
		return nil
	}
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
