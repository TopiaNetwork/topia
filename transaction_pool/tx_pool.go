package transactionpool

import (
	"encoding/hex"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"

	"github.com/TopiaNetwork/topia/account"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/common/types"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"github.com/TopiaNetwork/topia/network"
	"github.com/TopiaNetwork/topia/network/p2p"
	"github.com/TopiaNetwork/topia/network/protocol"
	"github.com/TopiaNetwork/topia/transaction"
)

const chainHeadChanSize = 10

var (
	evictionInterval    = 200 * time.Millisecond // Time interval to check for evictable transactions
	statsReportInterval = 500 * time.Millisecond // Time interval to report transaction pool stats
	republicInterval    = 30 * time.Second       //time interval to check transaction lifetime for report
)

var (
	ErrAlreadyKnown       = errors.New("transaction is already know")
	ErrNonceTooLow        = errors.New("transaction nonce is too low")
	ErrGasPriceTooLow     = errors.New("transaction gas price too low for miner")
	ErrReplaceUnderpriced = errors.New("new transaction price under priced")
	ErrUnderpriced        = errors.New("transaction underpriced")
	ErrTxGasLimit         = errors.New("exceeds block gas limit")
	ErrInsufficientFunds  = errors.New("insufficient funds for gas * price + value")
	ErrTxNotExist         = errors.New("transaction not found")
	ErrTxPoolOverflow     = errors.New("txPool is full")
	ErrOversizedData      = errors.New("transaction overSized data")
)

type TransactionPool interface {
	AddTx(tx *transaction.Transaction, local bool) error

	RemoveTxByKey(key string) error

	RemoveTxHashs(txHashs []string) []error

	Reset(oldHead, newHead *types.BlockHead) error

	UpdateTx(tx *transaction.Transaction, txKey string) error

	Pending() map[account.Address][]*transaction.Transaction

	Size() int

	Start(sysActor *actor.ActorSystem, network network.Network) error

	PickTxs(txsType int) []*transaction.Transaction
}

type transactionPool struct {
	config TransactionPoolConfig

	pubSubService p2p.P2PPubSubService

	chanChainHead     chan transaction.ChainHeadEvent
	chanReqReset      chan *txPoolResetRequest
	chanReqPromote    chan *accountSet
	chanReorgDone     chan chan struct{}
	chanReorgShutdown chan struct{}                 // requests shutdown of scheduleReorgLoop
	chanInitDone      chan struct{}                 // is closed once the pool is initialized (for tests)
	chanQueueTxEvent  chan *transaction.Transaction //check new tx insert to txpool
	chanRmTxs         chan []string
	query             TransactionPoolServant
	locals            *accountSet

	pending             *pendingTxs
	queue               *queueTxs
	allTxsForLook       *txLookup
	sortedByPriced      *txPricedList
	ActivationIntervals map[string]time.Time // ActivationInterval from each tx

	curState          StatePoolDB
	pendingNonces     uint64
	curMaxGasLimit    uint64
	log               tplog.Logger
	level             tplogcmm.LogLevel
	network           network.Network
	handler           TransactionPoolHandler
	marshaler         codec.Marshaler
	hasher            tpcmm.Hasher
	changesSinceReorg int // A counter for how many drops we've performed in-between reorg.
	wg                sync.WaitGroup
}

func NewTransactionPool(conf TransactionPoolConfig, level tplogcmm.LogLevel, log tplog.Logger, codecType codec.CodecType) *transactionPool {
	conf = (&conf).check()
	poolLog := tplog.CreateModuleLogger(level, "TransactionPool", log)
	pool := &transactionPool{
		config:              conf,
		log:                 poolLog,
		level:               level,
		allTxsForLook:       newTxLookup(),
		ActivationIntervals: make(map[string]time.Time),
		chanChainHead:       make(chan transaction.ChainHeadEvent, chainHeadChanSize),
		chanReqReset:        make(chan *txPoolResetRequest),
		chanReqPromote:      make(chan *accountSet),
		chanReorgDone:       make(chan chan struct{}),
		chanReorgShutdown:   make(chan struct{}),                 // requests shutdown of scheduleReorgLoop
		chanInitDone:        make(chan struct{}),                 // is closed once the pool is initialized (for tests)
		chanQueueTxEvent:    make(chan *transaction.Transaction), //check new tx insert to txpool
		chanRmTxs:           make(chan []string),

		marshaler: codec.CreateMarshaler(codecType),
		hasher:    tpcmm.NewBlake2bHasher(0),
	}
	pool.curMaxGasLimit = pool.query.GetMaxGasLimit()
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
	pool.Reset(nil, pool.query.CurrentBlock().GetHead())

	pool.wg.Add(1)
	go pool.scheduleReorgLoop()

	pool.loadLocal(conf.NoLocalFile, conf.PathLocal)
	pool.loadRemote(conf.NoRemoteFile, conf.PathRemote)
	pool.loadConfig(conf.NoConfigFile, conf.PathConfig)

	pool.pubSubService = pool.query.SubChainHeadEvent(pool.chanChainHead)

	go pool.loop()
	return pool
}

func (pool *transactionPool) AddTx(tx *transaction.Transaction, local bool) error {
	txId, err := tx.TxID()
	if err != nil {
		return err
	}
	if pool.allTxsForLook.Get(txId) != nil {
		return nil
	}
	if err := pool.ValidateTx(tx, local); err != nil {
		return err
	}
	if local {
		pool.AddLocal(tx)
	} else {
		pool.AddRemote(tx)
	}
	return nil
}

func (pool *transactionPool) Pending() map[account.Address][]*transaction.Transaction {
	pool.pending.Mu.Lock()
	defer pool.pending.Mu.Unlock()

	pending := make(map[account.Address][]*transaction.Transaction)
	for addr, list := range pool.pending.accTxs {
		txs := list.Flatten()
		if len(txs) > 0 {
			pending[addr] = txs
		}
	}
	return pending
}

func (pool *transactionPool) LocalAccounts() []account.Address {
	pool.queue.Mu.Lock()
	defer pool.queue.Mu.Unlock()
	return pool.locals.flatten()
}

func (pool *transactionPool) addTxs(txs []*transaction.Transaction, local, sync bool) []error {
	var (
		errs = make([]error, len(txs))
		news = make([]*transaction.Transaction, 0, len(txs))
	)
	for i, tx := range txs {
		if txId, err := tx.TxID(); err == nil {
			if pool.allTxsForLook.Get(txId) != nil {
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
	pool.pending.Mu.Lock()
	newErrs, _ := pool.addTxsLocked(news, local)
	//newErrs, dirtyAddrs := pool.addTxsLocked(news, local)
	pool.pending.Mu.Unlock()
	var nilSlot = 0
	for _, err := range newErrs {
		for errs[nilSlot] != nil {
			nilSlot++
		}
		errs[nilSlot] = err
		nilSlot++
	}
	// Reorg the pool internals if needed and return
	//done := pool.requestReplaceExecutables(dirtyAddrs)
	//if sync {
	//	<-done
	//}
	//
	return errs
}

// addTxsLocked attempts to queue a batch of transactions if they are valid.
// The transaction pool lock must be held.
func (pool *transactionPool) addTxsLocked(txs []*transaction.Transaction, local bool) ([]error, *accountSet) {
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

func (pool *transactionPool) UpdateTx(tx *transaction.Transaction, txKey string) error {
	// Two transactions with the same sender and receiver,
	//the later tx with higher gasPrice, can replace specified previously unPackaged tx.
	err := pool.ValidateTx(tx, false)
	if err != nil {
		return err
	}
	tx2 := pool.allTxsForLook.Get(txKey)

	if account.Address(tx2.FromAddr) == account.Address(tx.FromAddr) &&
		account.Address(tx2.TargetAddr) == account.Address(tx.TargetAddr) &&
		tx2.GasPrice <= tx.GasPrice {

		pool.RemoveTxByKey(txKey)
		pool.add(tx, false)
	}
	return nil
}
func (pool *transactionPool) Size() int {
	return pool.allTxsForLook.Count()
}

//Start register module
func (pool *transactionPool) Start(sysActor *actor.ActorSystem, network network.Network) error {
	actorPID, err := CreateTransactionPoolActor(pool.level, pool.log, sysActor, pool)
	if err != nil {
		pool.log.Panicf("CreateTransactionPoolActor error: %v", err)
		return err
	}
	network.RegisterModule("TransactionPool", actorPID, pool.marshaler)
	return nil
}

// RemoveTxByKey removes a single transaction from the queue, moving all subsequent
// transactions back to the future queue.
func (pool *transactionPool) RemoveTxByKey(key string) error {
	tx := pool.allTxsForLook.Get(key)
	if tx == nil {
		return ErrTxNotExist
	}
	addr := account.Address(hex.EncodeToString(tx.FromAddr))
	// Remove it from the list of known transactions
	pool.allTxsForLook.Remove(key)
	// Remove it from the list of sortedByPriced
	pool.sortedByPriced.Removed(1)

	// Remove the transaction from the pending lists and reset the account nonce
	pool.pending.Mu.Lock()
	if pending := pool.pending.accTxs[addr]; pending != nil {
		if removed, invalids := pending.Remove(tx); removed {
			if pending.Empty() {
				delete(pool.pending.accTxs, addr)
			}
			// Postpone any invalidated transactions
			for _, tx := range invalids {
				txId, _ := tx.TxID()
				// Internal shuffle shouldn't touch the lookup set.
				pool.queueAddTx(txId, tx, false, false)
			}
		}
	}

	pool.pending.Mu.Unlock()
	// Transaction is in the future queue
	pool.queue.Mu.Lock()
	if future := pool.queue.accTxs[addr]; future != nil {
		future.Remove(tx)
		if future.Empty() {
			delete(pool.queue.accTxs, addr)
			delete(pool.ActivationIntervals, key)
		}
	}
	pool.queue.Mu.Unlock()
	return nil
}

func (pool *transactionPool) Cost(tx *transaction.Transaction) *big.Int {
	total := pool.query.EstimateTxCost(tx)
	return total
}
func (pool *transactionPool) Gas(tx *transaction.Transaction) uint64 {
	total := pool.query.EstimateTxGas(tx)
	return total
}

func (pool *transactionPool) Get(key string) *transaction.Transaction {
	return pool.allTxsForLook.Get(key)
}

// queueAddTx inserts a new transaction into the non-executable transaction queue.
// Note, this method assumes the pool lock is held!
func (pool *transactionPool) queueAddTx(key string, tx *transaction.Transaction, local bool, addAll bool) (bool, error) {
	// Try to insert the transaction into the future queue
	pool.queue.Mu.Lock()
	from := account.Address(hex.EncodeToString(tx.FromAddr))
	if pool.queue.accTxs[from] == nil {
		pool.queue.accTxs[from] = newTxList(false)
	}

	inserted, old := pool.queue.accTxs[from].Add(tx)
	pool.queue.Mu.Unlock()
	if !inserted {
		// An older transaction was existed
		return false, ErrReplaceUnderpriced
	}
	// Discard any previous transaction and mark this
	if old != nil {
		oldTxId, _ := old.TxID()
		pool.allTxsForLook.Remove(oldTxId)
		pool.sortedByPriced.Removed(1)
	}
	// If the transaction isn't in lookup set but it's expected to be there,
	// show the error log.
	if pool.allTxsForLook.Get(key) == nil && !addAll {
		pool.log.Errorf("Missing transaction in lookup set, please report the issue", "TxID", key)
	}
	if addAll {

		pool.allTxsForLook.Add(tx, local)
		pool.sortedByPriced.Put(tx, local)

	}
	// If we never record the ActivationInterval, do it right now.
	if _, exist := pool.ActivationIntervals[key]; !exist {
		pool.ActivationIntervals[key] = time.Now()
	}
	return old != nil, nil
}

// GetLocalTxs retrieves all currently known local transactions, grouped by origin
// account and sorted by nonce. The returned transaction set is a copy and can be
// freely modified by calling code.
func (pool *transactionPool) GetLocalTxs() map[account.Address][]*transaction.Transaction {
	txs := make(map[account.Address][]*transaction.Transaction)
	for addr := range pool.locals.accounts {
		if pending := pool.pending.accTxs[addr]; pending != nil {
			txs[addr] = append(txs[addr], pending.Flatten()...)
		}
		if queued := pool.queue.accTxs[addr]; queued != nil {
			txs[addr] = append(txs[addr], queued.Flatten()...)
		}
	}
	return txs
}

func (pool *transactionPool) ValidateTx(tx *transaction.Transaction, local bool) error {
	//from := account.Address(tx.FromAddr[:])
	if uint64(tx.Size()) > txMaxSize {
		return ErrOversizedData
	}
	// Ensure the transaction doesn't exceed the current block limit gas.
	if pool.curMaxGasLimit < tx.GasLimit {
		return ErrTxGasLimit
	}
	//
	if !local && tx.GasPrice < pool.config.GasPriceLimit {
		return ErrGasPriceTooLow
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
func (pool *transactionPool) stats() (int, int) {
	pending := 0
	pool.pending.Mu.RLock()
	for _, list := range pool.pending.accTxs {
		pending += list.Len()
	}
	queued := 0
	pool.pending.Mu.RUnlock()
	pool.queue.Mu.RLock()
	for _, list := range pool.queue.accTxs {
		queued += list.Len()
	}
	pool.queue.Mu.RUnlock()
	return pending, queued
}

// add validates a transaction and inserts it into the non-executable queue for later
// pending promotion and execution. If the transaction is a replacement for an already
// pending or queued one, it overwrites the previous transaction if its price is higher.
//
// If a newly added transaction is marked as local, its sending account will be
// added to the allowlist, preventing any associated transaction from being dropped
// out of the pool due to pricing constraints.
func (pool *transactionPool) add(tx *transaction.Transaction, local bool) (replaced bool, err error) {
	// If the transaction is already known, discard it
	txId, _ := tx.TxID()
	if pool.allTxsForLook.Get(txId) != nil {
		pool.log.Tracef("Discarding already known transaction", "hash", txId)
		return false, ErrAlreadyKnown

	}
	// Make the local flag. If it's from local source or it's from the network but
	// the sender is marked as local previously, treat it as the local transaction.
	isLocal := local || pool.locals.containsTx(tx)
	//If the transaction fails basic validation, discard it
	if err := pool.ValidateTx(tx, isLocal); err != nil {
		pool.log.Tracef("Discarding invalid transaction", "txId", txId, "err", err)
		return false, err
	}

	// If the transaction pool is full, discard underpriced transactions
	if uint64(pool.allTxsForLook.Slots()+numSlots(tx)) > pool.config.PendingGlobalSlots+pool.config.QueueMaxTxsGlobal {
		// If the new transaction is underpriced, don't accept it

		if !isLocal && pool.sortedByPriced.Underpriced(tx) {
			pool.log.Tracef("Discarding underpriced transaction", "hash", txId, "GasPrice", tx.GasPrice)
			return false, ErrUnderpriced
		}
		if pool.changesSinceReorg > int(pool.config.PendingGlobalSlots/4) {
			return false, ErrTxPoolOverflow
		}
		drop, success := pool.sortedByPriced.Discard(pool.allTxsForLook.Slots()-int(pool.config.PendingGlobalSlots+pool.config.QueueMaxTxsGlobal)+numSlots(tx), isLocal)
		// Special case, we still can't make the room for the new remote one.
		if !isLocal && !success {
			pool.log.Tracef("Discarding overflown transaction", "txId", txId)
			return false, ErrTxPoolOverflow
		}
		// Bump the counter of rejections-since-reorg
		pool.changesSinceReorg += len(drop)
		// Kick out the underpriced remote transactions.
		for _, tx := range drop {
			txId, _ := tx.TxID()
			pool.log.Tracef("Discarding freshly underpriced transaction", "hash", txId, "gasPrice", tx.GasPrice)
			pool.RemoveTxByKey(txId)
		}
	}

	// Try to replace an existing transaction in the pending pool
	from := account.Address(hex.EncodeToString(tx.FromAddr))

	if list := pool.pending.accTxs[from]; list != nil && list.Overlaps(tx) {
		inserted, old := list.Add(tx)
		if !inserted {
			return false, ErrReplaceUnderpriced
		}
		if old != nil {
			pool.allTxsForLook.Remove(txId)
			pool.sortedByPriced.Removed(1)
		}
		pool.allTxsForLook.Add(tx, isLocal)
		pool.sortedByPriced.Put(tx, isLocal)
		pool.ActivationIntervals[txId] = time.Now()
	}
	// New transaction isn't replacing a pending one, push into queue
	replaced, err = pool.queueAddTx(txId, tx, isLocal, true)
	if err != nil {
		return false, err
	}

	// Mark local addresses and store local transactions
	if local && !pool.locals.contains(from) {

		//pool.log.Infof("Setting new local account", "address", from)

		pool.locals.add(from)

		pool.sortedByPriced.Removed(pool.allTxsForLook.RemoteToLocals(pool.locals)) // Migrate the remotes if it's marked as local first time.

	}

	//pool.log.Tracef("Pooled new future transaction", "txId", txId, "from", from, "to", tx.TargetAddr)
	return replaced, nil
}

// replaceTx adds a transaction to the pending (processable) list of transactions
// and returns whether it was inserted or an older was better.
// Note, this method assumes the pool lock is held!
func (pool *transactionPool) turnTx(addr account.Address, txId string, tx *transaction.Transaction) bool {
	pool.pending.Mu.Lock()
	defer pool.pending.Mu.Unlock()
	// Try to insert the transaction into the pending queue
	if pool.pending.accTxs[addr] == nil {
		pool.pending.accTxs[addr] = newTxList(true)
	}

	list := pool.pending.accTxs[addr]

	inserted, old := list.Add(tx)
	if !inserted {
		// An older transaction was existed, discard this
		pool.allTxsForLook.Remove(txId)
		pool.sortedByPriced.Removed(1)
		return false
	} else {
		pool.queue.Mu.Lock()
		queuelist := pool.queue.accTxs[addr]
		queuelist.Remove(tx)
		pool.queue.Mu.Unlock()
	}

	if old != nil {
		oldkey, _ := old.TxID()
		pool.allTxsForLook.Remove(oldkey)
		pool.sortedByPriced.Removed(1)

	}
	// Successful replace tx, bump the ActivationInterval
	pool.ActivationIntervals[txId] = time.Now()
	return true
}

func (pool *transactionPool) Stop() {
	// Unsubscribe subscriptions registered from blockchain
	pool.pubSubService.UnSubscribe(protocol.SyncProtocolID_Msg)
	//pool.log.Info("Transaction pool stopped")
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

// queueTxEvent enqueues a transaction event to be sent in the next reorg run.
func (pool *transactionPool) queueTxEvent(tx *transaction.Transaction) {
	select {
	case pool.chanQueueTxEvent <- tx:
	case <-pool.chanReorgShutdown:
	}
}

// replaceExecutables moves transactions that have become processable from the
// future queue to the set of pending transactions. During this process, all
// invalidated transactions (low nonce, low balance) are deleted.
func (pool *transactionPool) replaceExecutables(accounts []account.Address) []*transaction.Transaction {
	// Track the promoted transactions to broadcast them at once
	var replaced []*transaction.Transaction

	// Iterate over all accounts and promote any executable transactions
	for _, addr := range accounts {
		list := pool.queue.accTxs[addr]
		if list == nil {
			continue // Just in case someone calls with a non existing account
		}
		// Drop all transactions that are deemed too old (low nonce)
		forwards := list.Forward(pool.curState.GetNonce(addr))

		for _, tx := range forwards {
			if txId, err := tx.TxID(); err != nil {
				//print err
			} else {
				pool.allTxsForLook.Remove(txId)
			}
		}
		pool.log.Tracef("Removed old queued transactions", "count", len(forwards))
		// Drop all transactions that are too costly (low balance or out of gas)
		drops, _ := list.Filter(pool.curState.GetBalance(addr), pool.curMaxGasLimit)
		for _, tx := range drops {
			txId, err := tx.TxID()
			if err == nil {
				pool.allTxsForLook.Remove(txId)
				delete(pool.ActivationIntervals, txId)

			}
		}
		pool.log.Tracef("Removed unpayable queued transactions", "count", len(drops))

		// Gather all executable transactions and promote them
		readies := list.Ready(pool.curState.GetNonce(addr))
		for _, tx := range readies {
			txId, _ := tx.TxID()
			if pool.turnTx(addr, txId, tx) {
				replaced = append(replaced, tx)
			}
		}
		pool.log.Tracef("Promoted queued transactions", "count", len(replaced))

		// Drop all transactions over the allowed limit
		var caps []*transaction.Transaction
		if !pool.locals.contains(addr) {
			caps = list.Cap(int(pool.config.QueueMaxTxsAccount))
			for _, tx := range caps {
				txId, _ := tx.TxID()
				pool.allTxsForLook.Remove(txId)
				pool.log.Tracef("Removed cap-exceeding queued transaction", "txId", txId)
			}
		}
		// Mark all the items dropped as removed
		pool.sortedByPriced.Removed(len(forwards) + len(drops) + len(caps))

		// Delete the entire queue entry if it became empty.
		if list.Empty() {
			delete(pool.queue.accTxs, addr)
		}
	}
	return replaced
}

// demoteUnexecutables removes invalid and processed transactions from the pools
// executable/pending queue and any subsequent transactions that become unexecutable
// are moved back into the future queue.
//
// Note: transactions are not marked as removed in the priced list because re-heaping
// is always explicitly triggered by SetBaseFee and it would be unnecessary and wasteful
// to trigger a re-heap is this function
func (pool *transactionPool) demoteUnexecutables() {
	// Iterate over all accounts and demote any non-executable transactions
	for addr, list := range pool.pending.accTxs {
		nonce := pool.curState.GetNonce(addr)

		// Drop all transactions that are deemed too old (low nonce)
		olds := list.Forward(nonce)
		for _, tx := range olds {
			txId, _ := tx.TxID()
			pool.allTxsForLook.Remove(txId)
			pool.log.Tracef("Removed old pending transaction", "txId", txId)
		}
		// Drop all transactions that are too costly (low balance or out of gas), and queue any invalids back for later
		drops, invalids := list.Filter(pool.curState.GetBalance(addr), pool.curMaxGasLimit)
		for _, tx := range drops {
			txId, _ := tx.TxID()
			pool.log.Tracef("Removed unpayable pending transaction", "txId", txId)
			pool.allTxsForLook.Remove(txId)
		}

		for _, tx := range invalids {
			hash, _ := tx.TxID()
			pool.log.Tracef("Demoting pending transaction", "hash", hash)

			// Internal shuffle shouldn't touch the lookup set.
			pool.queueAddTx(hash, tx, false, false)
		}

		// If there's a gap in front, alert (should never happen) and postpone all transactions
		if list.Len() > 0 && list.txs.Get(nonce) == nil {
			gaped := list.Cap(0)
			for _, tx := range gaped {
				hash, _ := tx.TxID()
				pool.log.Errorf("Demoting invalidated transaction", "hash", hash)

				// Internal shuffle shouldn't touch the lookup set.
				pool.queueAddTx(hash, tx, false, false)
			}

		}
		// Delete the entire pending entry if it became empty.
		if list.Empty() {
			delete(pool.pending.accTxs, addr)
		}
	}
}

// PickTxs if txsType is 0,pick current pending txs,if 1 pick txs sorted by price and nonce
func (pool *transactionPool) PickTxs(txsType int) []*transaction.Transaction {
	switch txsType {
	case 0:
		return pool.CommitTxsForPending()

	case 1:
		return pool.CommitTxsByPriceAndNonce()
	}
	return nil
}

// CommitTxsForPending  : Block packaged transactions for pending
func (pool *transactionPool) CommitTxsForPending() []*transaction.Transaction {
	txls := pool.pending.accTxs
	var txs = make([]*transaction.Transaction, 0)
	for _, txlist := range txls {
		for _, tx := range txlist.txs.cache {
			txs = append(txs, tx)
		}
	}
	return txs
}

// CommitTxsByPriceAndNonce  : Block packaged transactions sorted by price and nonce
func (pool *transactionPool) CommitTxsByPriceAndNonce() []*transaction.Transaction {
	txSet := transaction.NewTxsByPriceAndNonce(pool.Pending())
	txs := make([]*transaction.Transaction, 0)
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
