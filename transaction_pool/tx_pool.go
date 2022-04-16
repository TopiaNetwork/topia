package transactionpool

import (
	"context"
	"encoding/json"
	"errors"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/eventhub"
	"github.com/TopiaNetwork/topia/transaction/universal"
	"sync"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"

	"github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"github.com/TopiaNetwork/topia/network"
	"github.com/TopiaNetwork/topia/network/protocol"
	"github.com/TopiaNetwork/topia/transaction/basic"
)

type PickTxType uint32

var ObsID string

const (
	PickTransactionsFromPending PickTxType = iota
	PickTransactionsSortedByGasPriceAndNonce
)

const (
	chainHeadChanSize = 10
	MOD_NAME          = "TransactionPool"
)

var (
	ErrAlreadyKnown       = errors.New("transaction is already know")
	ErrGasPriceTooLow     = errors.New("transaction gas price too low for miner")
	ErrReplaceUnderpriced = errors.New("new transaction price under priced")
	ErrUnderpriced        = errors.New("transaction underpriced")
	ErrTxGasLimit         = errors.New("exceeds block gas limit")
	ErrTxNotExist         = errors.New("transaction not found")
	ErrTxPoolOverflow     = errors.New("txPool is full")
	ErrOversizedData      = errors.New("transaction overSized data")
	ErrTxUpdate           = errors.New("can not update tx")
	ErrPendingIsNil       = errors.New("pending is nil")
	ErrUnRooted           = errors.New("UnRooted new chain")
	ErrQueueEmpty         = errors.New("queue is empty")
)

type TransactionPool interface {
	AddTx(tx *basic.Transaction, local bool) error

	RemoveTxByKey(key string) error

	RemoveTxHashs(hashs []string) []error

	Reset(oldHead, newHead *types.BlockHead) error

	UpdateTx(tx *basic.Transaction, txKey string) error

	Pending() ([]*basic.Transaction, error)

	Size() int

	Start(sysActor *actor.ActorSystem, network network.Network) error

	PickTxs(txType PickTxType) []*basic.Transaction
}

type transactionPool struct {
	nodeId string
	config TransactionPoolConfig

	chanSysShutDown   chan error
	chanChainHead     chan ChainHeadEvent
	chanReqReset      chan *txPoolResetRequest
	chanReqPromote    chan *accountSet
	chanReorgDone     chan chan struct{}
	chanReorgShutdown chan struct{} // requests shutdown of scheduleReorgLoop
	chanRmTxs         chan []string
	query             TransactionPoolServant
	locals            *accountSet

	pendings            *pendingsMap
	queues              *queuesMap
	allTxsForLook       *allTxsLookupMap
	sortedLists         *txSortedList
	ActivationIntervals *activationInterval // ActivationInterval from each tx
	TxHashCategory      *txHashCategory
	curState            StatePoolDB
	pendingNonces       uint64
	curMaxGasLimit      uint64
	log                 tplog.Logger
	level               tplogcmm.LogLevel
	network             network.Network
	ctx                 context.Context
	handler             TransactionPoolHandler
	marshaler           codec.Marshaler
	hasher              tpcmm.Hasher
	changesSinceReorg   int // A counter for how many drops we've performed in-between reorg.
	wg                  sync.WaitGroup
}

func NewTransactionPool(nodeID string, ctx context.Context, conf TransactionPoolConfig, level tplogcmm.LogLevel, log tplog.Logger, codecType codec.CodecType) *transactionPool {
	conf = (&conf).check()
	poolLog := tplog.CreateModuleLogger(level, "TransactionPool", log)
	pool := &transactionPool{
		nodeId:              nodeID,
		config:              conf,
		log:                 poolLog,
		level:               level,
		ctx:                 ctx,
		allTxsForLook:       newAllTxsLookupMap(),
		ActivationIntervals: newActivationInterval(),
		TxHashCategory:      newTxHashCategory(),
		chanChainHead:       make(chan ChainHeadEvent, chainHeadChanSize),
		chanReqReset:        make(chan *txPoolResetRequest),
		chanReqPromote:      make(chan *accountSet),
		chanReorgDone:       make(chan chan struct{}),
		chanReorgShutdown:   make(chan struct{}), // requests shutdown of scheduleReorgLoop
		chanRmTxs:           make(chan []string),

		marshaler: codec.CreateMarshaler(codecType),
		hasher:    tpcmm.NewBlake2bHasher(0),
	}
	//pool.network.Subscribe(ctx, protocol.SyncProtocolID_Msg, message.TopicValidator())
	pool.allTxsForLook.setAllTxsLookup(basic.TransactionCategory_Topia_Universal, newTxLookup())
	pool.curMaxGasLimit = pool.query.GetMaxGasLimit()
	pool.pendings = newPendingsMap()
	pool.queues = newQueuesMap()

	poolHandler := NewTransactionPoolHandler(poolLog, pool)
	pool.handler = poolHandler

	pool.locals = newAccountSet()
	if len(conf.Locals) > 0 {
		for _, addr := range conf.Locals {
			pool.log.Infof("Setting new local account ", "address:", addr)
			pool.locals.add(addr)
		}
	}
	pool.sortedLists = newTxSortedList(pool.allTxsForLook.getAllTxsLookupByCategory(basic.TransactionCategory_Topia_Universal))

	pool.Reset(nil, pool.query.CurrentBlock().GetHead())

	pool.wg.Add(1)
	go pool.scheduleReorgLoop()
	for category, _ := range pool.allTxsForLook.getAll() {
		pool.loadLocal(category, conf.NoLocalFile, conf.PathLocal[category])
		pool.loadRemote(category, conf.NoRemoteFile, conf.PathRemote[category])
	}

	pool.loadConfig(conf.NoConfigFile, conf.PathConfig)
	pool.loopChanSelect()

	return pool
}

func (pool *transactionPool) AddTx(tx *basic.Transaction, local bool) error {
	if local {
		err := pool.AddLocal(tx)
		if err != nil {
			return err
		}
	} else {
		err := pool.AddRemote(tx)
		if err != nil {
			return err
		}
	}
	return nil
}

// AddLocals enqueues a batch of transactions into the pool if they are valid, marking the
// senders as a local ones, ensuring they go around the local pricing constraints.
//
// This method is used to add transactions from the RPC API and performs synchronous pool
// reorganization and event propagation.
func (pool *transactionPool) AddLocals(txs []*basic.Transaction) []error {

	return pool.addTxs(txs, !pool.config.NoLocalFile, true)
}

// AddLocal enqueues a single local transaction into the pool if it is valid. This is
// a convenience wrapper aroundd AddLocals.
func (pool *transactionPool) AddLocal(tx *basic.Transaction) error {
	errs := pool.AddLocals([]*basic.Transaction{tx})
	return errs[0]
}

// AddRemotes enqueues a batch of transactions into the pool if they are valid. If the
// senders are not among the locally tracked ones, full pricing constraints will apply.
//
// This method is used to add transactions from the p2p network and does not wait for pool
// reorganization and internal event propagation.
func (pool *transactionPool) AddRemotes(txs []*basic.Transaction) []error {
	return pool.addTxs(txs, false, false)
}
func (pool *transactionPool) AddRemote(tx *basic.Transaction) error {
	errs := pool.AddRemotes([]*basic.Transaction{tx})

	return errs[0]
}

//dispatch
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
	err := pool.handler.ProcessTx(msg)
	if err != nil {
		return err

	}
	return nil
}

func (pool *transactionPool) BroadCastTx(tx *basic.Transaction) error {
	if tx == nil {
		return nil
	}
	msg := &TxMessage{}
	data, err := tx.GetData().Marshal()
	if err != nil {
		return err
	}
	msg.Data = data
	_, err = msg.Marshal()
	if err != nil {
		return err
	}
	var toModuleName []string
	toModuleName = append(toModuleName, MOD_NAME)
	//comment for Test
	//pool.network.Publish(pool.ctx, toModuleName, protocol.SyncProtocolID_Msg, data)

	return nil
}

func (pool *transactionPool) PendingOfCategory(category basic.TransactionCategory) map[tpcrtypes.Address][]*basic.Transaction {
	return pool.pendings.getAddrTxsByCategory(category)
}

func (pool *transactionPool) Pending() ([]*basic.Transaction, error) {

	TxList := make([]*basic.Transaction, 0)
	for category, _ := range pool.allTxsForLook.getAll() {
		TxList = append(TxList, pool.pendings.getTxsByCategory(category)...)
	}
	if len(TxList) == 0 {
		return nil, ErrPendingIsNil
	}
	return TxList, nil
}

func (pool *transactionPool) LocalAccounts() []tpcrtypes.Address {
	return pool.locals.flatten()
}

func (pool *transactionPool) addTxs(txs []*basic.Transaction, local, sync bool) []error {
	var (
		errs     = make([]error, len(txs))
		news     = make([]*basic.Transaction, 0, len(txs))
		category basic.TransactionCategory
	)

	for i, tx := range txs {
		category = basic.TransactionCategory(tx.Head.Category)
		if txId, err := tx.HashHex(); err == nil {
			if pool.allTxsForLook.getTxFromKeyFromAllTxsLookupByCategory(category, txId) != nil {
				errs[i] = ErrAlreadyKnown
				continue
			}
			news = append(news, tx)
		}
	}
	if len(news) == 0 {
		return errs
	}

	// Process all the new transaction and merge any errors into the original slice
	newErrs, _ := pool.addTxsLocked(news, local)

	//newErrs, dirtyAddrs := pool.addTxsLocked(news, local)
	var nilSlot = 0
	for _, err := range newErrs {

		for errs[nilSlot] != nil {
			nilSlot++
		}

		errs[nilSlot] = err
		nilSlot++
	}
	//fmt.Println("addTxs 006")

	// Reorg the pool internals if needed and return
	//done := pool.requestReplaceExecutables(dirtyAddrs)
	//if sync {
	//	<-done
	//}
	return errs
}

// addTxsLocked attempts to queue a batch of transactions if they are valid.
// The transaction pool lock must be held.
func (pool *transactionPool) addTxsLocked(txs []*basic.Transaction, local bool) ([]error, *accountSet) {
	dirty := newAccountSet()
	errs := make([]error, len(txs))
	for i, tx := range txs {
		replaced, err := pool.add(tx, local)

		errs[i] = err
		if err == nil && !replaced {
			dirty.addTx(tx)
		}
	}
	return errs, dirty
}

func (pool *transactionPool) UpdateTx(tx *basic.Transaction, txKey string) error {
	category := basic.TransactionCategory(tx.Head.Category)
	// Two transactions with the same sender and receiver,
	//the later tx with higher gasPrice, can replace specified previously unPackaged tx.
	//newTxId, _ := tx.HashHex()
	err := pool.ValidateTx(tx, false)
	if err != nil {
		return err
	}

	tx2 := pool.allTxsForLook.getTxFromKeyFromAllTxsLookupByCategory(category, txKey)
	switch category {
	case basic.TransactionCategory_Topia_Universal:
		if tpcrtypes.Address(tx2.Head.FromAddr) == tpcrtypes.Address(tx.Head.FromAddr) &&
			ToAddress(tx) == ToAddress(tx2) &&
			GasPrice(tx2) <= GasPrice(tx) {
			pool.RemoveTxByKey(txKey)
			pool.add(tx, false)
			//data := "txPool update a new " + string(category) + "tx,txHash is " + newTxId + ",the replaced txHash is " + txKey
			//eventhub.GetEventHubManager().GetEventHub(pool.nodeId).Trig(pool.ctx, eventhub.EventName_TxReceived, data)

		}
		return nil
	default:
		return ErrTxUpdate
	}
}

func (pool *transactionPool) Size() int {
	return pool.allTxsForLook.getAllCount()
}

//Start register module
func (pool *transactionPool) Start(sysActor *actor.ActorSystem, network network.Network) error {
	actorPID, err := CreateTransactionPoolActor(pool.level, pool.log, sysActor, pool)
	if err != nil {
		pool.log.Panicf("CreateTransactionPoolActor error: %v", err)
		return err
	}
	network.RegisterModule(MOD_NAME, actorPID, pool.marshaler)

	ObsID, err = eventhub.GetEventHubManager().GetEventHub(pool.nodeId).
		Observe(pool.ctx, eventhub.EventName_BlockAdded, pool.handler.processBlockAddedEvent)
	if err != nil {
		pool.log.Panicf("processBlockAddedEvent error:%s", err)
	}
	pool.log.Infof("processBlockAddedEvent,obsID:%s", ObsID)

	pool.loopChanSelect()
	return nil
}

// RemoveTxByKey removes a single transaction from the queue, moving all subsequent
// transactions back to the future queue.
func (pool *transactionPool) RemoveTxByKey(key string) error {

	category := pool.TxHashCategory.getByHash(key)

	tx := pool.allTxsForLook.getTxFromKeyFromAllTxsLookupByCategory(category, key)
	if tx == nil {
		return ErrTxNotExist
	}
	addr := tpcrtypes.Address(tx.Head.FromAddr)
	// Remove it from the list of known transactions
	pool.allTxsForLook.removeTxHashFromAllTxsLookupByCategory(category, key)
	// Remove it from the list of sortedByPriced
	pool.sortedLists.removedPricedlistByCategory(category, 1)
	//data := "txPool remove a " + string(category) + "tx,txHash is " + key
	//fmt.Println("eventhub,", data)
	//eventhub.GetEventHubManager().GetEventHub(pool.nodeId).Trig(pool.ctx, eventhub.EventName_TxReceived, data)

	// Remove the transaction from the pending lists and reset the account nonce
	f1 := func(txId string, tx *basic.Transaction) {
		pool.queueAddTx(txId, tx, false, false)
	}
	pool.pendings.getTxListRemoveByAddrOfCategory(f1, tx, category, addr)

	// Transaction is in the future queue
	f2 := func(string2 string) {
		pool.ActivationIntervals.removeTxActiv(string2)
		pool.TxHashCategory.removeHashCat(string2)
	}
	pool.queues.getTxListRemoveFutureByAddrOfCategory(tx, f2, key, category, addr)

	return nil
}

func (pool *transactionPool) RemoveTxHashs(hashs []string) []error {
	errs := make([]error, 0)
	for _, txHash := range hashs {
		if err := pool.RemoveTxByKey(txHash); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

//func (pool *transactionPool) Cost(tx *basic.Transaction) *big.Int {
//	total := pool.query.EstimateTxCost(tx)
//	return total
//}
//func (pool *transactionPool) Gas(tx *basic.Transaction) uint64 {
//	total := pool.query.EstimateTxGas(tx)
//	return total
//}

func (pool *transactionPool) Get(category basic.TransactionCategory, key string) *basic.Transaction {
	return pool.allTxsForLook.getTxFromKeyFromAllTxsLookupByCategory(category, key)
}

// queueAddTx inserts a new transaction into the non-executable transaction queue.
// Note, this method assumes the pool lock is held!
func (pool *transactionPool) queueAddTx(key string, tx *basic.Transaction, local bool, addAll bool) (bool, error) {
	// Try to insert the transaction into the future queue
	f1 := func(category basic.TransactionCategory, key string) {
		pool.allTxsForLook.removeTxHashFromAllTxsLookupByCategory(category, key)
		pool.sortedLists.removedPricedlistByCategory(category, 1)

	}
	f2 := func(category basic.TransactionCategory, string2 string) *basic.Transaction {
		return pool.allTxsForLook.getTxFromKeyFromAllTxsLookupByCategory(category, key)
	}
	f3 := func(string2 string) {
		pool.log.Errorf("Missing transaction in lookup set, please report the issue ", "TxID", key)
	}
	f4 := func(category basic.TransactionCategory, tx *basic.Transaction, local bool) {
		pool.allTxsForLook.addTxToAllTxsLookupByCategory(category, tx, local)
		pool.ActivationIntervals.setTxActiv(key, time.Now())
	}
	f5 := func(key string, category basic.TransactionCategory, local bool) {
		pool.ActivationIntervals.setTxActiv(key, time.Now())
		pool.TxHashCategory.setHashCat(key, category)
		if local {
			pool.BroadCastTx(tx)
		}
		//data := "txPool add a new " + string(category) + "tx,txHash is " + key
		//eventhub.GetEventHubManager().GetEventHub(pool.nodeId).Trig(pool.ctx, eventhub.EventName_TxReceived, data)
	}
	ok, err := pool.queues.addTxByKeyOfCategory(f1, f2, f3, f4, f5, key, tx, local, addAll)
	return ok, err
}

// GetLocalTxs retrieves all currently known local transactions, grouped by origin
// account and sorted by nonce. The returned transaction set is a copy and can be
// freely modified by calling code.
func (pool *transactionPool) GetLocalTxs(category basic.TransactionCategory) map[tpcrtypes.Address][]*basic.Transaction {
	txs := make(map[tpcrtypes.Address][]*basic.Transaction)
	for addr := range pool.locals.accounts {
		if txlist := pool.pendings.getTxListByAddrOfCategory(category, addr); txlist != nil {
			txs[addr] = append(txs[addr], txlist.Flatten()...)
		}

		if queued := pool.queues.getTxListByAddrOfCategory(category, addr); queued != nil {
			txs[addr] = append(txs[addr], queued.Flatten()...)
		}
	}
	return txs
}

func (pool *transactionPool) ValidateTx(tx *basic.Transaction, local bool) error {
	//from := account.Address(tx.FromAddr[:])
	if uint64(tx.Size()) > txMaxSize {
		return ErrOversizedData
	}

	// Ensure the transaction doesn't exceed the current block limit gas.
	if pool.curMaxGasLimit < GasLimit(tx) {
		return ErrTxGasLimit
	}
	//

	switch basic.TransactionCategory(tx.Head.Category) {
	case basic.TransactionCategory_Topia_Universal:
		if !local && GasPrice(tx) < pool.config.GasPriceLimit {
			return ErrGasPriceTooLow
		}
	case basic.TransactionCategory_Eth:
		if !local && GasPrice(tx) < pool.config.GasPriceLimit {
			return ErrGasPriceTooLow
		}
	default:
		return nil
	}

	//// Transactor should have enough funds to cover the costs
	//if pool.curState.GetBalance(from).Cmp(pool.Cost(tx)) < 0 {
	//	return ErrInsufficientFunds
	//}
	//
	////Is nonce is validated
	//if !(pool.curState.GetNonce(from) > tx.Nonce) {
	//	return ErrNonceTooLow
	//}

	return nil
}

// stats retrieves the current pool stats, namely the number of pending and the
// number of queued (non-executable) transactions.
func (pool *transactionPool) stats(category basic.TransactionCategory) (int, int) {

	pending := pool.pendings.getStatsOfCategory(category)
	queued := pool.queues.getStatsOfCategory(category)

	return pending, queued
}

// add validates a transaction and inserts it into the non-executable queue for later
// pending promotion and execution. If the transaction is a replacement for an already
// pending or queued one, it overwrites the previous transaction if its price is higher.
//
// If a newly added transaction is marked as local, its sending account will be
// added to the allowlist, preventing any associated transaction from being dropped
// out of the pool due to pricing constraints.
func (pool *transactionPool) add(tx *basic.Transaction, local bool) (replaced bool, err error) {
	// Make the local flag. If it's from local source or it's from the network but
	// the sender is marked as local previously, treat it as the local transaction.
	txId, _ := tx.HashHex()
	isLocal := local || pool.locals.containsTx(tx)

	category := basic.TransactionCategory(tx.Head.Category)
	// If the transaction is already known, discard it
	if pool.allTxsForLook.getTxFromKeyFromAllTxsLookupByCategory(category, txId) != nil {
		pool.log.Tracef("Discarding already known transaction ", "hash:", txId, "category:", category)
		return false, ErrAlreadyKnown
	}

	// If the transaction pool is full, discard underpriced transactions
	if uint64(pool.allTxsForLook.getSegmentFromAllTxsLookupByCategory(category)+numSegments(tx)) > pool.config.PendingGlobalSegments+pool.config.QueueMaxTxsGlobal {
		// If the new transaction is underpriced, don't accept it
		if !isLocal && pool.sortedLists.getPricedlistByCategory(category).Underpriced(tx) {
			gasprice := GasPrice(tx)
			pool.log.Tracef("Discarding underpriced transaction ", "hash:", txId, "GasPrice:", gasprice)
			return false, ErrUnderpriced
		}

		if pool.changesSinceReorg > int(pool.config.PendingGlobalSegments/4) {
			return false, ErrTxPoolOverflow
		}

		segment := pool.allTxsForLook.getSegmentFromAllTxsLookupByCategory(category) -
			int(pool.config.PendingGlobalSegments+pool.config.QueueMaxTxsGlobal) + numSegments(tx)
		drop, success := pool.sortedLists.DiscardFromPricedlistByCategor(category, segment, isLocal)

		// Special case, we still can't make the room for the new remote one.
		if !isLocal && !success {
			pool.log.Tracef("Discarding overflown transaction ", "txId:", txId)
			return false, ErrTxPoolOverflow
		}

		// Bump the counter of rejections-since-reorg
		pool.changesSinceReorg += len(drop)
		// Kick out the underpriced remote transactions.
		for _, txi := range drop {
			txIdi, _ := txi.HashHex()
			gasprice := GasPrice(txi)
			pool.log.Tracef("Discarding freshly underpriced transaction ", "hash:", txId, "gasPrice:", gasprice)
			pool.RemoveTxByKey(txIdi)
		}
	}
	// Try to replace an existing transaction in the pending pool
	from := tpcrtypes.Address(tx.Head.FromAddr)
	f1 := func(category basic.TransactionCategory, txId string) {
		pool.allTxsForLook.removeTxHashFromAllTxsLookupByCategory(category, txId)
	}
	f2 := func(category basic.TransactionCategory) {
		pool.sortedLists.removedPricedlistByCategory(category, 1)
	}
	f3 := func(category basic.TransactionCategory, tx *basic.Transaction, isLocal bool) {
		pool.allTxsForLook.addTxToAllTxsLookupByCategory(category, tx, isLocal)
	}
	f4 := func(category basic.TransactionCategory, tx *basic.Transaction, isLocal bool) {
		pool.sortedLists.putTxToPricedlistByCategory(category, tx, isLocal)
	}
	f5 := func(txId string) {
		pool.ActivationIntervals.setTxActiv(txId, time.Now())
		pool.log.Tracef("Pooled new executable transaction ", "hash:", txId, "from:", from)
	}
	if ok, err := pool.pendings.replaceTxOfAddrOfCategory(category, from, txId, tx, isLocal, f1, f2, f3, f4, f5); !ok {
		return ok, err
	}
	// New transaction isn't replacing a pending one, push into queue
	replaced, err = pool.queueAddTx(txId, tx, isLocal, true)
	if err != nil {
		return false, err
	}

	// Mark local addresses and store local transactions
	if local && !pool.locals.contains(from) {
		pool.log.Infof("Setting new local account,", "address:", from)
		pool.locals.add(from)
		num := pool.allTxsForLook.remoteToLocalsAllTxsLookupByCategory(category, pool.locals)
		pool.sortedLists.removedPricedlistByCategory(category, num) // Migrate the remotes if it's marked as local first time.

	}
	pool.log.Tracef("Pooled new future transaction,", "txId:", txId, "from:", from, "to:", ToAddress(tx))
	return replaced, nil
}

// turnTx adds a transaction to the pending (processable) list of transactions
// and returns whether it was inserted or an older was better.
// Note, this method assumes the pool lock is held!
func (pool *transactionPool) turnTx(addr tpcrtypes.Address, txId string, tx *basic.Transaction) bool {
	category := basic.TransactionCategory(tx.Head.Category)

	// Try to insert the transaction into the pending queue
	if pool.queues.getTxListByAddrOfCategory(category, addr) == nil ||
		pool.queues.getTxByNonceFromTxlistByAddrOfCategory(category, addr, tx.Head.Nonce) != tx {
		return false
	}
	if pool.pendings.getTxListByAddrOfCategory(category, addr) == nil {
		pool.pendings.setTxListOfCategory(category, addr, newCoreList(false))
	}
	inserted, old := pool.pendings.addTxToTxListByAddrOfCategory(category, addr, tx)
	if !inserted {
		// An older transaction was existed, discard this
		pool.allTxsForLook.removeTxHashFromAllTxsLookupByCategory(category, txId)
		pool.sortedLists.removedPricedlistByCategory(category, 1)
		return false
	} else {
		pool.queues.removeTxFromTxListByAddrOfCategory(category, addr, tx)
	}

	if old != nil {
		oldkey, _ := old.HashHex()
		pool.allTxsForLook.removeTxHashFromAllTxsLookupByCategory(category, oldkey)
		pool.sortedLists.removedPricedlistByCategory(category, 1)
	}
	// Successful replace tx, bump the ActivationInterval
	pool.ActivationIntervals.setTxActiv(txId, time.Now())

	return true
}

func (pool *transactionPool) Stop() {
	// Unsubscribe subscriptions registered from blockchain
	pool.network.UnSubscribe(protocol.SyncProtocolID_Msg)
	eventhub.GetEventHubManager().GetEventHub(pool.nodeId).UnObserve(pool.ctx, ObsID, eventhub.EventName_BlockAdded)
	pool.log.Info("TransactionPool stopped")
}

// requestReplaceExecutables requests transaction promotion checks for the given addresses.
// The returned channel is closed when the promotion checks have occurred.
func (pool *transactionPool) requestReplaceExecutables(set *accountSet) chan struct{} {
	select {
	case pool.chanReqPromote <- set:
		return <-pool.chanReorgDone
	case <-pool.chanReorgShutdown:
		return pool.chanReorgShutdown
	}
}

// replaceExecutables moves transactions that have become processable from the
// future queue to the set of pending transactions. During this process, all
// invalidated transactions (low nonce, low balance) are deleted.
func (pool *transactionPool) replaceExecutables(category basic.TransactionCategory, accounts []tpcrtypes.Address) {
	// Track the promoted transactions to broadcast them at once
	var replaced []*basic.Transaction

	// Iterate over all accounts and promote any executable transactions
	for _, addr := range accounts {
		f1 := func(address tpcrtypes.Address) uint64 { return pool.curState.GetNonce(address) }
		f2 := func(transactionCategory basic.TransactionCategory, string2 string) {
			pool.allTxsForLook.removeTxHashFromAllTxsLookupByCategory(category, string2)
		}
		forwardsCnt := pool.queues.replaceExecutablesDropTooOld(category, addr, f1, f2)

		pool.log.Tracef("Removed old queued transactions", "count", forwardsCnt)

		// Gather all executable transactions and promote them
		ft0 := func(address tpcrtypes.Address) uint64 { return pool.curState.GetNonce(address) }
		ft1 := func(category basic.TransactionCategory, address tpcrtypes.Address, tx *basic.Transaction) (bool, *basic.Transaction) {
			if pool.pendings.getTxListByAddrOfCategory(category, addr) == nil {
				pool.pendings.setTxListOfCategory(category, addr, newCoreList(false))
			}
			inserted, old := pool.pendings.addTxToTxListByAddrOfCategory(category, address, tx)
			return inserted, old
		}
		ft2 := func(category basic.TransactionCategory, txId string) {
			pool.allTxsForLook.removeTxHashFromAllTxsLookupByCategory(category, txId)
			pool.sortedLists.removedPricedlistByCategory(category, 1)
		}
		ft3 := func(category basic.TransactionCategory, addr tpcrtypes.Address, tx *basic.Transaction) {
			pool.queues.removeTxFromTxListByAddrOfCategory(category, addr, tx)
		}
		ft4 := func(category basic.TransactionCategory, oldkey string) {
			pool.allTxsForLook.removeTxHashFromAllTxsLookupByCategory(category, oldkey)
			pool.sortedLists.removedPricedlistByCategory(category, 1)
		}
		ft5 := func(txId string) { pool.ActivationIntervals.setTxActiv(txId, time.Now()) }
		replacedCnt := pool.queues.replaceExecutablesTurnTx(ft0, ft1, ft2, ft3, ft4, ft5, replaced, category, addr)

		pool.log.Tracef("Promoted queued transactions", "count", replacedCnt)

		// Drop all transactions over the allowed limit
		fl1 := func(addr tpcrtypes.Address) bool { return !pool.locals.contains(addr) }
		fl2 := func(category basic.TransactionCategory, txId string) {
			pool.allTxsForLook.removeTxHashFromAllTxsLookupByCategory(category, txId)
			pool.log.Tracef("Removed cap-exceeding queued transaction", "txId", txId)
		}
		capsCnt := pool.queues.replaceExecutablesDropOverLimit(fl1, fl2, pool.config.QueueMaxTxsAccount, category, addr)

		// Mark all the items dropped as removed
		pool.sortedLists.removedPricedlistByCategory(category, forwardsCnt+capsCnt)

		// Delete the entire queue entry if it became empty.
		pool.queues.replaceExecutablesDeleteEmpty(category, addr)
	}
}

// demoteUnexecutables removes invalid and processed transactions from the pools
// executable/pending queue and any subsequent transactions that become unexecutable
// are moved back into the future queue.
//
// Note: transactions are not marked as removed in the priced list because re-heaping
// is always explicitly triggered by SetBaseFee and it would be unnecessary and wasteful
// to trigger a re-heap is this function
func (pool *transactionPool) demoteUnexecutables(category basic.TransactionCategory) {
	// Iterate over all accounts and demote any non-executable transactions
	f1 := func(address tpcrtypes.Address) uint64 { return pool.curState.GetNonce(address) }
	f2 := func(category basic.TransactionCategory, txId string) {
		pool.allTxsForLook.removeTxHashFromAllTxsLookupByCategory(category, txId)
		pool.log.Tracef("Removed old pending transaction", "txId", txId)
	}
	f3 := func(hash string, tx *basic.Transaction) {
		pool.log.Errorf("Demoting invalidated transaction", "hash", hash)
		// Internal shuffle shouldn't touch the lookup set.
		pool.queueAddTx(hash, tx, false, false)
	}
	pool.pendings.demoteUnexecutablesByCategory(category, f1, f2, f3)

}

// PickTxs if txsType is 0,pick current pending txs,if 1 pick txs sorted by price and nonce
func (pool *transactionPool) PickTxs(txType PickTxType) (txs []*basic.Transaction) {
	txs = make([]*basic.Transaction, 0)
	switch txType {
	case PickTransactionsFromPending:
		txs = pool.pendings.getAllCommitTxs()
		return txs
	case PickTransactionsSortedByGasPriceAndNonce:
		for category, _ := range pool.allTxsForLook.getAll() {
			txs = append(txs, pool.CommitTxsByPriceAndNonce(category)...)
		}
		return txs
	default:
		return nil
	}
}
func (pool *transactionPool) PickTxsOfCategory(category basic.TransactionCategory, txType PickTxType) []*basic.Transaction {
	switch txType {
	case PickTransactionsFromPending:
		return pool.CommitTxsForPending(category)

	case PickTransactionsSortedByGasPriceAndNonce:
		return pool.CommitTxsByPriceAndNonce(category)
	default:
		return nil
	}
}

// CommitTxsForPending  : Block packaged transactions for pending
func (pool *transactionPool) CommitTxsForPending(category basic.TransactionCategory) []*basic.Transaction {
	return pool.pendings.getCommitTxsCategory(category)
}

// CommitTxsByPriceAndNonce  : Block packaged transactions sorted by price and nonce
func (pool *transactionPool) CommitTxsByPriceAndNonce(category basic.TransactionCategory) []*basic.Transaction {
	txSet := NewTxsByPriceAndNonce(pool.PendingOfCategory(category))
	txs := make([]*basic.Transaction, 0)
	for {
		tx := txSet.Peek()
		if tx == nil {
			break
		}
		txs = append(txs, tx)
		txSet.Pop()
	}
	return txs
}

func ToAddress(tx *basic.Transaction) tpcrtypes.Address {
	var toAddress tpcrtypes.Address
	var targetData universal.TransactionUniversalTransfer
	if basic.TransactionCategory(tx.Head.Category) == basic.TransactionCategory_Topia_Universal {
		var txData universal.TransactionUniversal
		_ = json.Unmarshal(tx.Data.Specification, &txData)
		if txData.Head.Type == uint32(universal.TransactionUniversalType_Transfer) {
			_ = json.Unmarshal(txData.Data.Specification, &targetData)
			toAddress = targetData.TargetAddr
		}
	} else {
		return ""
	}
	return toAddress
}

func GasPrice(tx *basic.Transaction) uint64 {
	var gasPrice uint64
	switch basic.TransactionCategory(tx.Head.Category) {
	case basic.TransactionCategory_Topia_Universal:
		var txUniver universal.TransactionUniversal
		_ = json.Unmarshal(tx.Data.Specification, &txUniver)
		gasPrice = txUniver.Head.GasPrice
		return gasPrice
	default:
		return 0
	}
}

func GasLimit(tx *basic.Transaction) uint64 {
	var gasLimit uint64
	if basic.TransactionCategory(tx.Head.Category) == basic.TransactionCategory_Topia_Universal {
		var txData universal.TransactionUniversal
		_ = json.Unmarshal(tx.Data.Specification, &txData)
		gasLimit = txData.Head.GasLimit
		return gasLimit
	} else {
		return 0
	}
}
