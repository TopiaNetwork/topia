package transactionpool

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/TopiaNetwork/topia/account"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/common/types"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"github.com/TopiaNetwork/topia/network"
	"github.com/TopiaNetwork/topia/transaction"
	"github.com/ethereum/go-ethereum/common/prque"
	"github.com/ethereum/go-ethereum/log"
	"io/ioutil"
	"math"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)
// chainHeadChanSize is the size of channel listening to ChainHeadEvent.
const (
	chainHeadChanSize = 10

	// txSlotSize is used to calculate how many data slots a single transaction
	// takes up based on its size. The slots are used as DoS protection, ensuring
	// that validating a new transaction remains a constant operation (in reality
	// O(maxslots), where max slots are 4 currently).
	txSlotSize = 32 * 1024
	txMaxSize  = 4 * txSlotSize
)
var(
	evictionInterval    = 200 * time.Millisecond     // Time interval to check for evictable transactions
	statsReportInterval = 500 * time.Millisecond // Time interval to report transaction pool stats

)


type TransactionPool interface {
	AddTx(tx *transaction.Transaction,local bool) error

	RemoveTxByKey(key transaction.TxKey,outofbound bool) error

	Reset(oldHead,newHead *types.BlockHead) error

	UpdateTx(tx *transaction.Transaction,txKey transaction.TxKey) error

	Pending() map[account.Address][]*transaction.Transaction

	Size() int

	Start(sysActor *actor.ActorSystem, network network.Network) error
	TxsPackaged(txs map[account.Address][]*transaction.Transaction,txsType int)
	CommitTxsForPending(map[account.Address][]*transaction.Transaction) map[account.Address][]*transaction.Transaction
	CommitTxsByPriceAndNonce(map[account.Address][]*transaction.Transaction) map[account.Address][]*transaction.Transaction
}

type TransactionPoolConfig struct {
	chain          			blockChain
	Locals                  []account.Address
	NoLocalFile             bool
	NoRemoteFile			bool
	NoConfigFile			bool

	PathLocal            	string
	PathRemote           	string
	PathConfig         	  	string
	ReStoredDur             time.Duration

	GasPriceLimit           uint64

	PendingAccountSlots 	uint64 // Number of executable transaction slots guaranteed per account
	PendingGlobalSlots 		uint64 // Maximum number of executable transaction slots for all accounts
	QueueMaxTxsAccount 		uint64 // Maximum number of non-executable transaction slots permitted per account
	QueueMaxTxsGlobal 		uint64 // Maximum number of non-executable transaction slots for all accounts

	LifetimeForAccount      time.Duration
}

var (
	ErrAlreadyKnown   			= errors.New("transaction is already know")
	ErrInvalidSender   		    = errors.New("transaction has invalid sender")
	ErrNonceTooLow              = errors.New("transaction nonce is too low")
	ErrGasPriceTooLow           = errors.New("transaction gas price too low for miner")
	ErrReplaceUnderpriced       = errors.New("new transaction price under priced")
	ErrUnderpriced 				= errors.New("transaction underpriced")
	ErrTxGasLimit 				= errors.New("exceeds block gas limit")
	ErrInsufficientFunds 		= errors.New("insufficient funds for gas * price + value")
	ErrTxNotExist					= errors.New("transaction not found")
	// ErrTxPoolOverflow is returned if the transaction pool is full and can't accpet
	// another remote transaction.
	ErrTxPoolOverflow 			= errors.New("txPool is full")
	ErrOversizedData 			= errors.New("transaction overSized data")
)

var DefaultTransactionPoolConfig = TransactionPoolConfig{
	PathLocal:           	"localTransactions.rlp",
	PathRemote:          	"remoteTransactions.rlp",
	PathConfig:          	"txPoolConfigs.json",
	ReStoredDur:              30 * time.Minute,

	GasPriceLimit:          1,
	PendingAccountSlots:	16,
	PendingGlobalSlots:     8192, // queue capacity
	QueueMaxTxsAccount:		64,
	QueueMaxTxsGlobal:		8192*2,

	LifetimeForAccount:		30 * time.Minute,
}

func (config *TransactionPoolConfig) check() TransactionPoolConfig {
	conf := *config
	if conf.GasPriceLimit < 1 {
		//tplog.Logger.Warnf("Invalid gasPriceLimit,updated to default value:",string(conf.gasLimit),string(DefaultTransactionPoolConfig.gasLimit))
		conf.GasPriceLimit = DefaultTransactionPoolConfig.GasPriceLimit
	}
	if conf.PendingAccountSlots < 1 {
		//tplog.Logger.Warnf("Invalid MaxTxForQueuePerAccount,updated to default value:",string(conf.MaxTxForQueuePerAccount),string(DefaultTransactionPoolConfig.MaxTxForQueuePerAccount))
		conf.PendingAccountSlots = DefaultTransactionPoolConfig.PendingAccountSlots
	}
	if conf.PendingGlobalSlots < 1 {
		//tplog.Logger.Warnf("Invalid MaxTxForQueue,updated to default value:",string(conf.MaxTxForQueue),string(DefaultTransactionPoolConfig.MaxTxForQueue))
		conf.PendingGlobalSlots = DefaultTransactionPoolConfig.PendingGlobalSlots
	}
	if conf.QueueMaxTxsAccount < 1 {
		//tplog.Logger.Warnf("Invalid MaxTxForQueue,updated to default value:",string(conf.MaxTxForQueue),string(DefaultTransactionPoolConfig.MaxTxForQueue))
		conf.QueueMaxTxsAccount = DefaultTransactionPoolConfig.QueueMaxTxsAccount
	}
	if conf.QueueMaxTxsGlobal < 1 {
		//tplog.Logger.Warnf("Invalid MaxTxForQueue,updated to default value:",string(conf.MaxTxForQueue),string(DefaultTransactionPoolConfig.MaxTxForQueue))
		conf.QueueMaxTxsGlobal = DefaultTransactionPoolConfig.QueueMaxTxsGlobal
	}

	return conf
}

type blockChain interface {
	CurrentBlock() *types.Block
	GetBlock(hash types.BlockHash,num uint64) *types.Block
	StateAt(root types.BlockHash)(*StatePoolDB,error)
	SubChainHeadEvent(ch chan<- transaction.ChainHeadEvent) Subscription
}

type transactionPool struct {
	config				 TransactionPoolConfig
	txFeed               Feed
	chainHeadSub  		 Subscription
	scope       		 SubscriptionScope

	chanChainHead 		 chan transaction.ChainHeadEvent
	chanReqReset     	 chan *txPoolResetRequest
	chanReqPromote   	 chan *accountSet
	chanReorgDone    	 chan chan struct{}
	chanReorgShutdown	 chan struct{}  // requests shutdown of scheduleReorgLoop
	chanInitDone     	 chan struct{}  // is closed once the pool is initialized (for tests)
	chanQueueTxEvent  	 chan *transaction.Transaction  //check new tx insert to txpool

	query                 TxPoolQuery

	locals       		 *accountSet
	txStored       		 *txStored
	Signer        		 transaction.BaseSigner  //no achieved
	mu           		 sync.RWMutex
	wg            		 sync.WaitGroup // tracks loop, scheduleReorgLoop

	pending       		 map[account.Address]*txList
	queue        	  	 map[account.Address]*txList
	allTxsForLook        *txLookup
	sortedByPriced       *txPricedList
	heartbeats           map[account.Address]time.Time // Last heartbeat from each known account

	curState       		 *StatePoolDB
	pendingNonces  		 *txNoncer      // Pending state tracking virtual nonces
	curMaxGasLimit       uint64
	log            		 tplog.Logger
	level          		 tplogcmm.LogLevel
	network        		 network.Network
	handler        		 TransactionPoolHandler
	marshaler      		 codec.Marshaler
	hasher         		 tpcmm.Hasher
	changesSinceReorg 	 int // A counter for how many drops we've performed in-between reorg.
}

func (pool *transactionPool) AddTx(tx *transaction.Transaction,local bool) error {
	txId ,err := tx.TxID()
	if err != nil {
		return err
	}
	if pool.allTxsForLook.Get(txId) != nil{ return nil}
	if err := pool.ValidateTx(tx,local);err != nil {return err }
	if local {
		pool.AddLocal(tx)
	} else {
		pool.AddRemote(tx)
	}
	return nil
}


type txPoolResetRequest struct {
	oldHead, newHead *types.BlockHead
}
func NewTransactionPool(conf TransactionPoolConfig, level tplogcmm.LogLevel, log tplog.Logger, codecType codec.CodecType) *transactionPool {
	conf = (&conf).check()
	poolLog := tplog.CreateModuleLogger(level, "TransactionPool", log)
	pool := &transactionPool{
		config:           conf,
		log:              poolLog,
		level:            level,
		Signer:           transaction.MakeSigner(),//no achieved for signer!!!!!!!!

		pending:          make(map[account.Address]*txList),
		queue:            make(map[account.Address]*txList),
		allTxsForLook:    newTxLookup(),

		chanChainHead:    make(chan transaction.ChainHeadEvent,chainHeadChanSize),
		chanReqReset:  	  make(chan *txPoolResetRequest),
		chanReqPromote:   make(chan *accountSet),
		chanReorgDone:    make(chan chan struct{}),
		chanReorgShutdown:make(chan struct{}),  // requests shutdown of scheduleReorgLoop
		chanInitDone:     make(chan struct{}),  // is closed once the pool is initialized (for tests)
		chanQueueTxEvent: make(chan *transaction.Transaction),  //check new tx insert to txpool

		marshaler:        codec.CreateMarshaler(codecType),
		hasher:           tpcmm.NewBlake2bHasher(0),
	}

	poolHandler := NewTransactionPoolHandler(poolLog, pool)

	pool.handler = poolHandler

	pool.locals =newAccountSet(pool.Signer)
	for _, addr := range conf.Locals {
		//log.Info("Setting new local account", "address", addr)
		pool.locals.add(addr)
	}
	pool.sortedByPriced = newTxPricedList(pool.allTxsForLook)  //done
	//reset func to refresh the pool.
	pool.Reset(nil,conf.chain.CurrentBlock().GetHead())

	// Start the reorg loop early, so it can handle requests generated during stored tx loading.
	pool.wg.Add(1)
	go pool.scheduleReorgLoop()


	// If local transactions and storing is enabled, load from disk
	if !conf.NoLocalFile && conf.PathLocal != "" {
		pool.txStored = newTxStored(conf.PathLocal, conf.PathRemote)

		if err := pool.txStored.loadLocal(pool.AddLocals); err != nil {
			log.Warn("Failed to load local transaction from stored file")
		}
	}
	//load remote txs from disk
	if !conf.NoRemoteFile && conf.PathRemote != "" {
		if err := pool.txStored.loadRemote(pool.AddLocals); err != nil {
			log.Warn("Failed to load remote transactions")
		}
	}
	//load txPool configs from disk
	if !conf.NoConfigFile && conf.PathConfig !="" {
		if con,err := pool.Loadconfig();err != nil{
			log.Warn("Failed to load txPool configs")
		}else {
			conf = *con
		}

	}

	//Subscribe events from blockchain
	pool.chainHeadSub = pool.config.chain.SubChainHeadEvent(pool.chanChainHead)

	//start the main event loop
	pool.wg.Add(1)
	go pool.loop()
	return pool
}

func (pool *transactionPool) processTx(msg *TxMessage) error {

	err := pool.handler.ProcessTx(msg)
	if err != nil{
		return err
	}
	return nil
}

// loop is the transaction pool's main event loop, waiting for and reacting to
// outside blockchain events as well as for various reporting and transaction
// eviction events.
func (pool *transactionPool) loop() {
	defer pool.wg.Done()

	var (
		prevPending, prevQueued, prevStales int
		// Start the stats reporting and transaction eviction tickers
		report  = time.NewTicker(statsReportInterval)
		evict   = time.NewTicker(evictionInterval)
		stored = time.NewTicker(pool.config.ReStoredDur)
		// Track the previous head headers for transaction reorgs
		head = pool.config.chain.CurrentBlock()
	)
	defer report.Stop()
	defer evict.Stop()
	defer stored.Stop()

	// Notify tests that the init phase is done
	close(pool.chanInitDone)
	for {
		select {
		// Handle ChainHeadEvent
		case ev := <-pool.chanChainHead:
			if ev.Block != nil {
				pool.requestReset(head.Head, ev.Block.Head)
				head = ev.Block
			}

		// System shutdown.  When the system is shut down, save to the files locals/remotes/configs
		case <-pool.chainHeadSub.Err():
			close(pool.chanReorgShutdown)
			//local txs save
			if err := pool.txStored.saveLocal(pool.local()); err != nil {
				log.Warn("Failed to save local transaction", "err", err)
			}
			//remote txs save
			if err := pool.txStored.saveRemote(pool.remote()); err != nil {
				log.Warn("Failed to save remote transaction", "err", err)
			}
			//txPool configs save
			if err := pool.SaveConfig(); err != nil {
				log.Warn("Failed to save transaction pool configs","err",err)
			}
			return

		// Handle stats reporting ticks
		case <-report.C:
			pool.mu.RLock()
			pending, queued := pool.stats() //
			pool.mu.RUnlock()
			stales := int(atomic.LoadInt64(&pool.sortedByPriced.stales))

			if pending != prevPending || queued != prevQueued || stales != prevStales {
				log.Debug("Transaction pool status report", "executable", pending, "queued", queued, "stales", stales)
				prevPending, prevQueued, prevStales = pending, queued, stales
			}

		// Handle inactive account transaction eviction
		case <-evict.C:
			pool.mu.Lock()
			for addr := range pool.queue {
				// Skip local transactions from the eviction mechanism
				if pool.locals.contains(addr) {
					continue
				}
				// Any non-locals old enough should be removed
				if time.Since(pool.heartbeats[addr]) > pool.config.LifetimeForAccount {
					list := pool.queue[addr].Flatten()
					for _, tx := range list {
						txId,_ := tx.TxID()
						pool.RemoveTxByKey(txId, true)
					}
				}
			}
			pool.mu.Unlock()

		// Handle local transaction  store
		case <-stored.C:
			if pool.txStored != nil {
				pool.mu.Lock()
				if err := pool.txStored.saveLocal(pool.local()); err != nil {
					log.Warn("Failed to save local tx ", "err", err)
				}
				pool.mu.Unlock()
			}
		}
	}
}

// RemoveTxs : after commitTxs you need removeTxs from txPool
func (pool *transactionPool) RemoveTxs(txl map[account.Address][]*transaction.Transaction) {
	for _,txs := range txl {
		for _,tx := range txs{
			txId,_ := tx.TxID()
			pool.RemoveTxByKey(txId,true)
		}
	}
}



func (pool *transactionPool) Pending() map[account.Address][]*transaction.Transaction {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	pending := make(map[account.Address][]*transaction.Transaction)
	for addr, list := range pool.pending {
		txs := list.Flatten()
		if len(txs) > 0 {
			pending[addr] = txs
		}
	}
	return pending
}

func (pool *transactionPool)Locals() []account.Address {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	return pool.locals.flatten()
}


func (pool *transactionPool) Loadconfig()(conf *TransactionPoolConfig,error error) {
	data,err := ioutil.ReadFile(pool.config.PathConfig)
	if err != nil {return nil,err}
	config := &conf
	err = json.Unmarshal(data,&config)
	if err != nil {
		return nil,err
	}
	return *config,nil
}

func (pool *transactionPool) UpdateTxPoolConfig( conf TransactionPoolConfig){
	conf = (conf).check()
	pool.config = conf
	return
}



func (pool *transactionPool) SaveConfig() error {
	fmt.Printf("%v",pool.config)
	conf,err := json.Marshal(pool.config)
	if err!= nil {
		return err
	}

	err = ioutil.WriteFile(pool.config.PathConfig,conf,0664)
	if err != nil{
		return err
	}
	return nil
}


//dispatch 收到交易，定期（）广播出去，local交易优先广播，优先广播交易池中时间最长的。
func (pool *transactionPool) dispatch(context actor.Context, data []byte) {
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
		if err != nil {return}
	default:
		pool.log.Errorf("TransactionPool receive invalid msg %d", txPoolMsg.MsgType)
		return
	}
}


func (pool *transactionPool) addTxs(txs []*transaction.Transaction,local,sync bool) []error {
	var (
		errs = make([]error,len(txs))
		news = make([]*transaction.Transaction,0,len(txs))
	)
	for i,tx := range txs{
		if txId,err:= tx.TxID(); err == nil{
			if pool.allTxsForLook.Get(txId) != nil{
				errs[i] = ErrAlreadyKnown
				continue
			}
			_,er := transaction.Sender(pool.Signer,tx)
			if er != nil{
				errs[i] = ErrInvalidSender
				continue
			}
			news = append(news,tx)
		}
	}
	if len(news) == 0 {
		return errs
	}
	// Process all the new transaction and merge any errors into the original slice
	pool.mu.Lock()
	newErrs,dirtyAddrs := pool.addTxsLocked(news,local)
	pool.mu.Unlock()

	var nilSlot = 0
	for _,err := range newErrs{
		for errs[nilSlot] != nil {
			nilSlot++
		}
		errs[nilSlot] = err
		nilSlot ++
	}
	// Reorg the pool internals if needed and return
	done := pool.requestPromoteExecutables(dirtyAddrs)
	if sync {
		<-done
	}
	return errs
}


// addTxsLocked attempts to queue a batch of transactions if they are valid.
// The transaction pool lock must be held.
func (pool *transactionPool) addTxsLocked(txs []*transaction.Transaction, local bool) ([]error, *accountSet) {
	dirty := newAccountSet(pool.Signer)
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


func (pool *transactionPool)UpdateTx(tx *transaction.Transaction,txKey transaction.TxKey) error {
	// chinese:如果交易的from和to都是同一个，并且，gas费用更高的后来者交易，可以替换未打包的交易。
	err := pool.ValidateTx(tx,false)
	if err != nil {return err}
	 tx2:= pool.allTxsForLook.Get(txKey)
	 if account.Address(tx2.FromAddr)==account.Address(tx.FromAddr) &&
		 account.Address(tx2.TargetAddr)==account.Address(tx.TargetAddr) &&
		 tx2.GasLimit <= tx.GasLimit {
		 pool.RemoveTxByKey(txKey,true)
		 pool.add(tx,false)
	 }
	return nil
}
func (pool *transactionPool)Size() int {
	return pool.allTxsForLook.Count()
}

// AddLocals enqueues a batch of transactions into the pool if they are valid, marking the
// senders as a local ones, ensuring they go around the local pricing constraints.
//
// This method is used to add transactions from the RPC API and performs synchronous pool
// reorganization and event propagation.
func (pool *transactionPool) AddLocals(txs []*transaction.Transaction) []error {
	return pool.addTxs(txs, !pool.config.NoLocalFile, true)
}

// AddLocal enqueues a single local transaction into the pool if it is valid. This is
// a convenience wrapper aroundd AddLocals.
func (pool *transactionPool) AddLocal(tx *transaction.Transaction) error {
	errs := pool.AddLocals([]*transaction.Transaction{tx})
	return errs[0]
}

// AddRemotes enqueues a batch of transactions into the pool if they are valid. If the
// senders are not among the locally tracked ones, full pricing constraints will apply.
//
// This method is used to add transactions from the p2p network and does not wait for pool
// reorganization and internal event propagation.
func (pool *transactionPool) AddRemotes(txs []*transaction.Transaction) []error {
	return pool.addTxs(txs, false, false)
}
func (pool *transactionPool) AddRemote(tx *transaction.Transaction) error {
	errs := pool.AddRemotes([]*transaction.Transaction{tx})
	return errs[0]
}

//Start chinese:网络收到消息，把消息转发给交易池，转换数据结构。事件触发模型。
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
func (pool *transactionPool) RemoveTxByKey(key transaction.TxKey,outofbound bool) error{
	tx := pool.allTxsForLook.Get(key)
	if tx == nil {return ErrTxNotExist }
	addr, _ := transaction.Sender(pool.Signer,tx)
	// Remove it from the list of known transactions
	pool.allTxsForLook.Remove(key)
	if outofbound {
		pool.sortedByPriced.Removed(1)
	}

	// Remove the transaction from the pending lists and reset the account nonce
	if pending := pool.pending[addr]; pending != nil {
		if removed, invalids := pending.Remove(tx); removed {
			if pending.Empty() {
				delete(pool.pending, addr)
			}
			// Postpone any invalidated transactions
			for _,tx := range invalids{
				txId,_ := tx.TxID()
				// Internal shuffle shouldn't touch the lookup set.
				pool.queueAddTx(txId, tx, false, false)
			}
			// Update the account nonce if needed
			pool.pendingNonces.setIfLower(addr,tx.Nonce)
			// Reduce the pending counter

			return nil
		}
	}
	// Transaction is in the future queue
	if future := pool.queue[addr];future != nil {

		if future.Empty() {
			delete(pool.queue,addr)
			delete(pool.heartbeats,addr)
		}
	}
	return nil
}

// scheduleReorgLoop schedules runs of reset and promoteExecutables. Code above should not
// call those methods directly, but request them being run using requestReset and
// requestPromoteExecutables instead.
func (pool *transactionPool) scheduleReorgLoop() {
	defer pool.wg.Done()

	var (
		curDone       chan struct{} // non-nil while runReorg is active
		nextDone      = make(chan struct{})
		launchNextRun bool
		reset         *txPoolResetRequest
		dirtyAccounts *accountSet
		queuedEvents  = make(map[account.Address]*txSortedMap)
	)
	for {
		// Launch next bckground reorg if neededa
		if curDone == nil && launchNextRun {
			// Run the background reorg and announcements
			go pool.runReorg(nextDone, reset, dirtyAccounts, queuedEvents)

			// Prepare everything for the next round of reorg
			curDone, nextDone = nextDone, make(chan struct{})
			launchNextRun = false

			reset, dirtyAccounts = nil, nil
			queuedEvents = make(map[account.Address]*txSortedMap)
		}

		select {
		case req := <-pool.chanReqReset:
			// Reset request: update head if request is already pending.
			if reset == nil {
				reset = req
			} else {
				reset.newHead = req.newHead
			}
			launchNextRun = true
			pool.chanReorgDone <- nextDone

		case req := <-pool.chanReqPromote:
			// Promote request: update address set if request is already pending.
			if dirtyAccounts == nil {
				dirtyAccounts = req
			} else {
				dirtyAccounts.merge(req)
			}
			launchNextRun = true
			pool.chanReorgDone <- nextDone

		case tx := <-pool.chanQueueTxEvent:
			// Queue up the event, but don't schedule a reorg. It's up to the caller to
			// request one later if they want the events sent.
			addr, _ := transaction.Sender(pool.Signer, tx)
			if _, ok := queuedEvents[addr]; !ok {
				queuedEvents[addr] = newTxSortedMap()
			}
			queuedEvents[addr].Put(tx)

		case <-curDone:
			curDone = nil

		case <-pool.chanReorgShutdown:
			// Wait for current run to finish.
			if curDone != nil {
				<-curDone
			}
			close(nextDone)
			return
		}
	}
}

// runReorg runs reset and promoteExecutables on behalf of scheduleReorgLoop.
func (pool *transactionPool) runReorg(done chan struct{}, reset *txPoolResetRequest, dirtyAccounts *accountSet, events map[account.Address]*txSortedMap) {
	defer close(done)
	var promoteAddrs []account.Address
	if dirtyAccounts != nil && reset == nil {
		// Only dirty accounts need to be promoted, unless we're resetting.
		// For resets, all addresses in the tx queue will be promoted and
		// the flatten operation can be avoided.
		promoteAddrs = dirtyAccounts.flatten()
	}
	pool.mu.Lock()
	if reset != nil {
		// Reset from the old head to the new, rescheduling any reorged transactions
		pool.Reset(reset.oldHead, reset.newHead)

		// Nonces were reset, discard any events that became stale
		for addr := range events {
			events[addr].Forward(pool.pendingNonces.get(addr))
			if events[addr].Len() == 0 {
				delete(events, addr)
			}
		}
		// Reset needs promote for all addresses
		promoteAddrs = make([]account.Address, 0, len(pool.queue))
		for addr := range pool.queue {
			promoteAddrs = append(promoteAddrs, addr)
		}
	}
	// Check for pending transactions for every account that sent new ones
	promoted := pool.promoteExecutables(promoteAddrs)

	// If a new block appeared, validate the pool of pending transactions. This will
	// remove any transaction that has been included in the block or was invalidated
	// because of another transaction (e.g. higher gas price).
	if reset != nil {
		pool.demoteUnexecutables() //demote transactions
		if reset.newHead != nil  {
			pool.sortedByPriced.Reheap()
		}
		// Update all accounts to the latest known pending nonce
		nonces := make(map[account.Address]uint64, len(pool.pending))
		for addr, list := range pool.pending {
			highestPending := list.LastElement()
			nonces[addr] = highestPending.Nonce + 1
		}
		pool.pendingNonces.setAll(nonces)
	}
	// Ensure pool.queue and pool.pending sizes stay within the configured limits.
	pool.truncatePending()
	pool.truncateQueue()

	pool.changesSinceReorg = 0 // Reset change counter
	pool.mu.Unlock()

	// Notify subsystems for newly added transactions
	for _, tx := range promoted {
		addr, _ := transaction.Sender(pool.Signer, tx)
		if _, ok := events[addr]; !ok {
			events[addr] = newTxSortedMap()
		}
		events[addr].Put(tx)
	}
	if len(events) > 0 {
		var txs []*transaction.Transaction
		for _, set := range events {
			txs = append(txs, set.Flatten()...)
		}
		pool.txFeed.Send(NewTxsEvent{txs})
	}
}



// requestReset requests a pool reset to the new head block.
// The returned channel is closed when the reset has occurred.
func (pool *transactionPool) requestReset(oldHead *types.BlockHead, newHead *types.BlockHead) chan struct{} {
	select {
	case pool.chanReqReset <- &txPoolResetRequest{oldHead, newHead}:
		return <-pool.chanReorgDone
	case <-pool.chanReorgShutdown:
		return pool.chanReorgShutdown
	}
}


func (pool *transactionPool) Cost(tx *transaction.Transaction) *big.Int {
	total := pool.query.EstimateTxCost(tx)
	return total
}
func (pool *transactionPool) Gas(tx *transaction.Transaction) uint64 {
	total := pool.query.EstimateTxGas(tx)
	return total
}

func (pool *transactionPool) Get(key transaction.TxKey) *transaction.Transaction {
	return pool.allTxsForLook.Get(key)
}

// queueAddTx inserts a new transaction into the non-executable transaction queue.
// Note, this method assumes the pool lock is held!
func (pool *transactionPool) queueAddTx(key transaction.TxKey, tx *transaction.Transaction, local bool, addAll bool) (bool, error) {
	// Try to insert the transaction into the future queue
	from, _ := transaction.Sender(pool.Signer, tx) // already validated
	if pool.queue[from] == nil {
		pool.queue[from] = newTxList(false)
	}
	inserted, old := pool.queue[from].Add(tx)
	if !inserted {
		// An older transaction was existed
		return false, ErrReplaceUnderpriced
	}
	// Discard any previous transaction and mark this
	if old != nil {
		oldTxId,_ := old.TxID()
		pool.allTxsForLook.Remove(oldTxId)
		pool.sortedByPriced.Removed(1)
	}
	// If the transaction isn't in lookup set but it's expected to be there,
	// show the error log.
	if pool.allTxsForLook.Get(key) == nil && !addAll {
		log.Error("Missing transaction in lookup set, please report the issue", "TxID", key)
	}
	if addAll {
		pool.allTxsForLook.Add(tx, local)
		pool.sortedByPriced.Put(tx, local)
	}
	// If we never record the heartbeat, do it right now.
	if _, exist := pool.heartbeats[from]; !exist {
		pool.heartbeats[from] = time.Now()
	}
	return old != nil, nil
}


// local retrieves all currently known local transactions, grouped by origin
// account and sorted by nonce. The returned transaction set is a copy and can be
// freely modified by calling code.
func (pool *transactionPool) local() map[account.Address][]*transaction.Transaction {
	txs := make(map[account.Address][]*transaction.Transaction)
	for addr := range pool.locals.accounts {
		if pending := pool.pending[addr]; pending != nil {
			txs[addr] = append(txs[addr], pending.Flatten()...)
		}
		if queued := pool.queue[addr]; queued != nil {
			txs[addr] = append(txs[addr], queued.Flatten()...)
		}
	}
	return txs
}
// local retrieves all currently known local transactions, grouped by origin
// account and sorted by nonce. The returned transaction set is a copy and can be
// freely modified by calling code.
func (pool *transactionPool) remote() map[account.Address][]*transaction.Transaction {
	txs := make(map[account.Address][]*transaction.Transaction)
	for addr := range pool.locals.accounts {
		if pending := pool.pending[addr]; pending == nil {
			txs[addr] = append(txs[addr], pending.Flatten()...)
		}
		if queued := pool.queue[addr]; queued == nil {
			txs[addr] = append(txs[addr], queued.Flatten()...)
		}
	}
	return txs
}

func (pool *transactionPool) ValidateTx(tx *transaction.Transaction,local bool) error {
	from := account.Address(string(tx.FromAddr[:]))

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
	// Transactor should have enough funds to cover the costs
	if pool.curState.GetBalance(from).Cmp(pool.Cost(tx)) < 0 {
		return ErrInsufficientFunds
	}

	//Is nonce is validated
	if !(pool.curState.GetNonce(from) > tx.Nonce){
		return ErrNonceTooLow
	}

	return nil
}


// stats retrieves the current pool stats, namely the number of pending and the
// number of queued (non-executable) transactions.
func (pool *transactionPool) stats() (int, int) {
	pending := 0
	for _, list := range pool.pending {
		pending += list.Len()
	}
	queued := 0
	for _, list := range pool.queue {
		queued += list.Len()
	}
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

	txId,_ := tx.TxID()
	if pool.allTxsForLook.Get(txId) != nil {
		log.Trace("Discarding already known transaction", "hash", txId)
		return false, ErrAlreadyKnown
	}
	// Make the local flag. If it's from local source or it's from the network but
	// the sender is marked as local previously, treat it as the local transaction.
	isLocal := local || pool.locals.containsTx(tx)

	// If the transaction fails basic validation, discard it
	if err := pool.ValidateTx(tx, isLocal); err != nil {
		log.Trace("Discarding invalid transaction", "txId", txId, "err", err)
		return false, err
	}
	// If the transaction pool is full, discard underpriced transactions
	if uint64(pool.allTxsForLook.Slots()+numSlots(tx)) > pool.config.PendingGlobalSlots+pool.config.QueueMaxTxsGlobal {
		// If the new transaction is underpriced, don't accept it
		if !isLocal && pool.sortedByPriced.Underpriced(tx) {
			log.Trace("Discarding underpriced transaction", "hash", txId, "GasPrice", tx.GasPrice)
			return false, ErrUnderpriced
		}
		// We're about to replace a transaction. The reorg does a more thorough
		// analysis of what to remove and how, but it runs async. We don't want to
		// do too many replacements between reorg-runs, so we cap the number of
		// replacements to 25% of the slots
		if pool.changesSinceReorg > int(pool.config.PendingGlobalSlots/4) {
			return false, ErrTxPoolOverflow
		}

		// New transaction is better than our worse ones, make room for it.
		// If it's a local transaction, forcibly discard all available transactions.
		// Otherwise if we can't make enough room for new one, abort the operation.
		drop, success := pool.sortedByPriced.Discard(pool.allTxsForLook.Slots()-int(pool.config.PendingGlobalSlots+pool.config.QueueMaxTxsGlobal)+numSlots(tx), isLocal)

		// Special case, we still can't make the room for the new remote one.
		if !isLocal && !success {
			log.Trace("Discarding overflown transaction", "txId", txId)
			return false, ErrTxPoolOverflow
		}
		// Bump the counter of rejections-since-reorg
		pool.changesSinceReorg += len(drop)
		// Kick out the underpriced remote transactions.
		for _, tx := range drop {
			txId,_ := tx.TxID()
			log.Trace("Discarding freshly underpriced transaction", "hash", txId, "gasPrice", tx.GasPrice)
			pool.RemoveTxByKey(txId, false)
		}
	}
	// Try to replace an existing transaction in the pending pool
	from, _ := transaction.Sender(pool.Signer, tx) // already validated

	pool.queueTxEvent(tx)   //send new tx to chan queueTxEventChan

	// New transaction isn't replacing a pending one, push into queue
	replaced, err = pool.queueAddTx(txId, tx, isLocal, true)
	if err != nil {
		return false, err
	}
	// Mark local addresses and store local transactions
	if local && !pool.locals.contains(from) {
		//log.Info("Setting new local account", "address", from)
		pool.locals.add(from)
		pool.sortedByPriced.Removed(pool.allTxsForLook.RemoteToLocals(pool.locals)) // Migrate the remotes if it's marked as local first time.
	}
	pool.storeTx(from, tx)

	log.Trace("Pooled new future transaction", "txId", txId, "from", from, "to", tx.TargetAddr)
	return replaced, nil
}



func (pool *transactionPool)GetAccForTxID(key transaction.TxKey) account.Address{
	return account.Address(string(pool.allTxsForLook.Get(key).FromAddr[:]))
}
// storeTx adds the specified transaction to the local disk  if it is
// deemed to have been sent from a local account.
func (pool *transactionPool) storeTx(from account.Address, tx *transaction.Transaction) {
	if pool.txStored == nil || !pool.locals.contains(from) {
		return
	}
	if err := pool.txStored.insert(tx); err != nil {
		//log.Warn("Failed to store local transaction", "err", err)
	}
}


// promoteTx adds a transaction to the pending (processable) list of transactions
// and returns whether it was inserted or an older was better.
//
// Note, this method assumes the pool lock is held!
func (pool *transactionPool) promoteTx(addr account.Address, txId transaction.TxKey, tx *transaction.Transaction) bool {
	// Try to insert the transaction into the pending queue
	if pool.pending[addr] == nil {
		pool.pending[addr] = newTxList(true)
	}
	list := pool.pending[addr]

	inserted, _ := list.Add(tx)
	if !inserted {
		// An older transaction was existed, discard this
		pool.allTxsForLook.Remove(txId)
		pool.sortedByPriced.Removed(1)
		return false
	}

	// Set the potentially new pending nonce and notify any subsystems of the new tx
	pool.pendingNonces.set(addr, tx.Nonce+1)
	// Successful promotion, bump the heartbeat
	pool.heartbeats[addr] = time.Now()
	return true
}


// SubscribeNewTxsEvent registers a subscription of NewTxsEvent and
// starts sending event to the given channel.
func (pool *transactionPool) SubscribeNewTxsEvent(ch chan<- NewTxsEvent) Subscription {
	return pool.scope.Track(pool.txFeed.Subscribe(ch))
}
// requestPromoteExecutables requests transaction promotion checks for the given addresses.
// The returned channel is closed when the promotion checks have occurred.
func (pool *transactionPool) requestPromoteExecutables(set *accountSet) chan struct{} {
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


// promoteExecutables moves transactions that have become processable from the
// future queue to the set of pending transactions. During this process, all
// invalidated transactions (low nonce, low balance) are deleted.
func (pool *transactionPool) promoteExecutables(accounts []account.Address) []*transaction.Transaction {
	// Track the promoted transactions to broadcast them at once
	var promoted []*transaction.Transaction

	// Iterate over all accounts and promote any executable transactions
	for _, addr := range accounts {
		list := pool.queue[addr]
		if list == nil {
			continue // Just in case someone calls with a non existing account
		}
		// Drop all transactions that are deemed too old (low nonce)
		forwards := list.Forward(pool.curState.GetNonce(addr))

		for _, tx := range forwards {
			if txId,err := tx.TxID(); err != nil{
				//print err
			} else {
				pool.allTxsForLook.Remove(txId)
			}
		}
		log.Trace("Removed old queued transactions", "count", len(forwards))
		// Drop all transactions that are too costly (low balance or out of gas)
		drops, _ := list.Filter(pool.curState.GetBalance(addr), pool.curMaxGasLimit)
		for _, tx := range drops {
			txId,err := tx.TxID()
			if err == nil{
				pool.allTxsForLook.Remove(txId)
			}
		}
		log.Trace("Removed unpayable queued transactions", "count", len(drops))

		// Gather all executable transactions and promote them
		readies := list.Ready(pool.pendingNonces.get(addr))
		for _, tx := range readies {
			txId,_ := tx.TxID()
			if pool.promoteTx(addr, txId, tx) {
				promoted = append(promoted, tx)
			}
		}
		log.Trace("Promoted queued transactions", "count", len(promoted))

		// Drop all transactions over the allowed limit
		var caps []*transaction.Transaction
		if !pool.locals.contains(addr) {
			caps = list.Cap(int(pool.config.QueueMaxTxsAccount))
			for _, tx := range caps {
				txId,_ := tx.TxID()
				pool.allTxsForLook.Remove(txId)
				log.Trace("Removed cap-exceeding queued transaction", "txId", txId)
			}
		}
		// Mark all the items dropped as removed
		pool.sortedByPriced.Removed(len(forwards) + len(drops) + len(caps))

		// Delete the entire queue entry if it became empty.
		if list.Empty() {
			delete(pool.queue, addr)
			delete(pool.heartbeats, addr)
		}
	}
	return promoted
}


// truncatePending removes transactions from the pending queue if the pool is above the
// pending limit. The algorithm tries to reduce transaction counts by an approximately
// equal number for all for accounts with many pending transactions.
func (pool *transactionPool) truncatePending() {
	pending := uint64(0)
	for _, list := range pool.pending {
		pending += uint64(list.Len())
	}
	if pending <= pool.config.PendingGlobalSlots {
		return
	}

	// Assemble a spam order to penalize large transactors first
	spammers := prque.New(nil)
	for addr, list := range pool.pending {
		// Only evict transactions from high rollers
		if !pool.locals.contains(addr) && uint64(list.Len()) > pool.config.PendingAccountSlots {
			spammers.Push(addr, int64(list.Len()))
		}
	}
	// Gradually drop transactions from offenders
	offenders := []account.Address{}
	for pending > pool.config.PendingGlobalSlots && !spammers.Empty() {
		// Retrieve the next offender if not local address
		offender, _ := spammers.Pop()
		offenders = append(offenders, offender.(account.Address))

		// Equalize balances until all the same or below threshold
		if len(offenders) > 1 {
			// Calculate the equalization threshold for all current offenders
			threshold := pool.pending[offender.(account.Address)].Len()

			// Iteratively reduce all offenders until below limit or threshold reached
			for pending > pool.config.PendingGlobalSlots && pool.pending[offenders[len(offenders)-2]].Len() > threshold {
				for i := 0; i < len(offenders)-1; i++ {
					list := pool.pending[offenders[i]]

					caps := list.Cap(list.Len() - 1)
					for _, tx := range caps {
						// Drop the transaction from the global pools too
						txId,_ := tx.TxID()
						pool.allTxsForLook.Remove(txId)

						// Update the account nonce to the dropped transaction
						pool.pendingNonces.setIfLower(offenders[i], tx.Nonce)
						log.Trace("Removed fairness-exceeding pending transaction", "txId", txId)
					}
					pool.sortedByPriced.Removed(len(caps))
					pending--
				}
			}
		}
	}

	// If still above threshold, reduce to limit or min allowance
	if pending > pool.config.PendingGlobalSlots && len(offenders) > 0 {
		for pending > pool.config.PendingGlobalSlots && uint64(pool.pending[offenders[len(offenders)-1]].Len()) > pool.config.PendingAccountSlots {
			for _, addr := range offenders {
				list := pool.pending[addr]

				caps := list.Cap(list.Len() - 1)
				for _, tx := range caps {
					// Drop the transaction from the global pools too
					txId,_ := tx.TxID()
					pool.allTxsForLook.Remove(txId)

					// Update the account nonce to the dropped transaction
					pool.pendingNonces.setIfLower(addr, tx.Nonce)
					log.Trace("Removed fairness-exceeding pending transaction", "txId", txId)
				}
				pool.sortedByPriced.Removed(len(caps))

				pending--
			}
		}
	}
}

// truncateQueue drops the oldes transactions in the queue if the pool is above the global queue limit.
func (pool *transactionPool) truncateQueue() {
	queued := uint64(0)
	for _, list := range pool.queue {
		queued += uint64(list.Len())
	}
	if queued <= pool.config.QueueMaxTxsGlobal {
		return
	}

	// Sort all accounts with queued transactions by heartbeat
	addresses := make(addressesByHeartbeat, 0, len(pool.queue))
	for addr := range pool.queue {
		if !pool.locals.contains(addr) { // don't drop locals
			addresses = append(addresses, addressByHeartbeat{addr, pool.heartbeats[addr]})
		}
	}
	sort.Sort(addresses)

	// Drop transactions until the total is below the limit or only locals remain
	for drop := queued - pool.config.QueueMaxTxsGlobal; drop > 0 && len(addresses) > 0; {
		addr := addresses[len(addresses)-1]
		list := pool.queue[addr.address]

		addresses = addresses[:len(addresses)-1]

		// Drop all transactions if they are less than the overflow
		if size := uint64(list.Len()); size <= drop {
			for _, tx := range list.Flatten() {
				txId,_ := tx.TxID()
				pool.RemoveTxByKey(txId, true)
			}
			drop -= size
			continue
		}
		// Otherwise drop only last few transactions
		txs := list.Flatten()
		for i := len(txs) - 1; i >= 0 && drop > 0; i-- {
			txId,_ := txs[i].TxID()
			pool.RemoveTxByKey(txId, true)
			drop--
		}
	}
}


func(pool *transactionPool)Reset(oldHead,newHead *types.BlockHead) error{
	//If the old header and the new header do not meet certain conditions,
	//part of the transaction needs to be injected back into the transaction pool
	var reInject []transaction.Transaction

	if oldHead != nil && types.BlockHash(oldHead.TxHashRoot) != types.BlockHash(newHead.TxHashRoot){
		oldNum := oldHead.GetHeight()
		newNum := newHead.GetHeight()
		//If the difference between the old block and the new block is greater than 64
		//then no recombination is carried out
		if depth := uint64(math.Abs(float64(oldNum)-float64(newNum)));depth > 64 {
			//fmt.Printf("Skipping deep transaction reorg", "depth", depth)
		} else {
			//The reorganization looks shallow enough to put all the transactions into memory
			var discarded, included []transaction.Transaction
			var (
				rem = pool.config.chain.GetBlock(types.BlockHash(oldHead.TxHashRoot),oldHead.Height)
				add = pool.config.chain.GetBlock(types.BlockHash(newHead.TxHashRoot),newHead.Height)
			)
			if rem == nil {
				if newNum >= oldNum {
					log.Warn("Transcation pool reset with missing oldhead",
					"old", types.BlockHash(oldHead.TxHashRoot),
						"new", types.BlockHash(newHead.TxHashRoot))
					return nil
				}
				log.Debug("Skipping transaction reset caused by setHead",
					"old", types.BlockHash(oldHead.TxHashRoot), "oldnum", oldNum,
					"new", types.BlockHash(newHead.TxHashRoot), "newnum", newNum)
			} else {
				for rem.Head.Height > add.Head.Height {
					for _, tx := range rem.Data.Txs {
						var txType transaction.Transaction
						err := pool.marshaler.Unmarshal(tx, &txType)
						if err != nil {
							discarded = append(discarded, txType)
						}
					}
					if rem = pool.config.chain.GetBlock(types.BlockHash(rem.Head.ParentBlockHash), rem.Head.Height-1);
						rem == nil {
						log.Error("UnRooted old chain seen by tx pool", "block", oldHead.Height,
							"hash", types.BlockHash(oldHead.TxHashRoot))
						return nil
					}
				}
				for add.Head.Height > rem.Head.Height {
					for _,tx := range add.Data.Txs {
						var txType transaction.Transaction
						err := pool.marshaler.Unmarshal(tx,&txType)
						if err != nil {
							included = append(included,txType)
						}
					}
					if add = pool.config.chain.GetBlock(types.BlockHash(add.Head.ParentBlockHash),add.Head.Height - 1);add == nil {
						log.Error("UnRooted new chain seen by tx pool", "block", newHead.Height,
							"hash", types.BlockHash(newHead.TxHashRoot))
						return nil
					}
				}
				for types.BlockHash(rem.Head.TxHashRoot) != types.BlockHash(add.Head.TxHashRoot) {
					for _,tx := range rem.Data.Txs{
						var txType transaction.Transaction
						err := pool.marshaler.Unmarshal(tx,&txType)
						if err != nil {
							discarded = append(discarded,txType)
						}
					}
					if rem = pool.config.chain.GetBlock(types.BlockHash(rem.Head.ParentBlockHash),rem.Head.Height-1);
						rem == nil {
						log.Error("UnRooted old chain seen by tx pool", "block", oldHead.Height,
							"hash", types.BlockHash(oldHead.TxHashRoot))
						return nil
					}
					for _,tx := range add.Data.Txs{
						var txType transaction.Transaction
						err := pool.marshaler.Unmarshal(tx,&txType)
						if err != nil{
							included = append(included,txType)
						}
					}
					if add = pool.config.chain.GetBlock(types.BlockHash(add.Head.ParentBlockHash),add.Head.Height-1);
						add ==nil{
						log.Error("UnRooted new chain seen by tx pool", "block", newHead.Height,
							"hash", types.BlockHash(newHead.TxHashRoot))
						return nil
					}
				}
				reInject = transaction.TxDifference(discarded,included)
			}
		}
	}
	// Initialize the internal state to the current head
	if newHead == nil {
		newHead = pool.config.chain.CurrentBlock().GetHead()
	}
	stateDb,err := pool.config.chain.StateAt(types.BlockHash(newHead.TxHashRoot))
	if err != nil{
		log.Error("Failed to reset txPool state", "err", err)
		return nil
	}
	pool.curState      = stateDb
	pool.pendingNonces = newTxNoncer(stateDb)
	//pool.curMaxGasLimit     = newHead.GasLimit //no achieve for newhead no gaslimit
	log.Debug("ReInjecting stale transactions", "count", len(reInject))
	return nil
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
	for addr, list := range pool.pending {
		nonce := pool.curState.GetNonce(addr)

		// Drop all transactions that are deemed too old (low nonce)
		olds := list.Forward(nonce)
		for _, tx := range olds {
			txId,_ := tx.TxID()
			pool.allTxsForLook.Remove(txId)
			log.Trace("Removed old pending transaction", "txId", txId)
		}
		// Drop all transactions that are too costly (low balance or out of gas), and queue any invalids back for later
		drops, invalids := list.Filter(pool.curState.GetBalance(addr), pool.curMaxGasLimit)
		for _, tx := range drops {
			txId,_ := tx.TxID()
			log.Trace("Removed unpayable pending transaction", "txId", txId)
			pool.allTxsForLook.Remove(txId)
		}

		for _, tx := range invalids {
			hash,_ := tx.TxID()
			log.Trace("Demoting pending transaction", "hash", hash)

			// Internal shuffle shouldn't touch the lookup set.
			pool.queueAddTx(hash, tx, false, false)
		}

		// If there's a gap in front, alert (should never happen) and postpone all transactions
		if list.Len() > 0 && list.txs.Get(nonce) == nil {
			gaped := list.Cap(0)
			for _, tx := range gaped {
				hash,_ := tx.TxID()
				log.Error("Demoting invalidated transaction", "hash", hash)

				// Internal shuffle shouldn't touch the lookup set.
				pool.queueAddTx(hash, tx, false, false)
			}

		}
		// Delete the entire pending entry if it became empty.
		if list.Empty() {
			delete(pool.pending, addr)
		}
	}
}


func (pool *transactionPool)TxsPackaged(txs map[account.Address][]*transaction.Transaction,txsType int){
	switch txsType {
	case 0:
		pool.CommitTxsForPending(txs)
		pool.RemoveTxs(txs)
	case 1:
		pool.CommitTxsByPriceAndNonce(txs)
		pool.RemoveTxs(txs)
	}
}


// CommitTxsForPending  : Block packaged transactions for pending
func (pool *transactionPool) CommitTxsForPending(txs map[account.Address][]*transaction.Transaction) map[account.Address][]*transaction.Transaction {

	return txs
}
// CommitTxsByPriceAndNonce  : Block packaged transactions sorted by price and nonce
func (pool *transactionPool) CommitTxsByPriceAndNonce(txs map[account.Address][]*transaction.Transaction) map[account.Address][]*transaction.Transaction{

	txset := transaction.NewTxsByPriceAndNonce(pool.Signer,txs)
	txs = make(map[account.Address][]*transaction.Transaction,0)
	for {
		tx := txset.Peek()
		if tx == nil {
			break
		}
		from,_ := transaction.Sender(pool.Signer,tx)
		txs[from] = append(txs[from],tx)
	}
	return txs
}



// addressByHeartbeat is an account address tagged with its last activity timestamp.
type addressByHeartbeat struct {
	address   account.Address
	heartbeat time.Time
}

type addressesByHeartbeat []addressByHeartbeat

func (a addressesByHeartbeat) Len() int           { return len(a) }
func (a addressesByHeartbeat) Less(i, j int) bool { return a[i].heartbeat.Before(a[j].heartbeat) }
func (a addressesByHeartbeat) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }


type accountSet struct {
	accounts map[account.Address]struct{}
	signer   transaction.BaseSigner
	cache    *[]account.Address
}
func newAccountSet(signer transaction.BaseSigner, addrs ...account.Address) *accountSet {
	as := &accountSet{
		accounts: make(map[account.Address]struct{}),
		signer:   signer,
	}
	for _, addr := range addrs {
		as.add(addr)
	}
	return as
}

func (accSet *accountSet) contains(addr account.Address) bool {
	_, exist := accSet.accounts[addr]
	return exist
}
func (accSet *accountSet) containsTx(tx *transaction.Transaction) bool {
	if addr, err := transaction.Sender(accSet.signer,tx);err == nil {
		return accSet.contains(addr)
	}
	//no achieved
	return false
}

func (accSet *accountSet) empty() bool {
	return len(accSet.accounts) == 0
}
func (accSet *accountSet) len() int {
	return len(accSet.accounts)
}

func (accSet *accountSet) add(addr account.Address) {
	accSet.accounts[addr] = struct{}{}
	accSet.cache = nil
}
func (accSet *accountSet) addTx(tx *transaction.Transaction){
	if addr,err := transaction.Sender(accSet.signer,tx);err ==nil{
		accSet.add(addr)
	}
}
func(accSet *accountSet) merge(other *accountSet) {
	for addr := range other.accounts {
		accSet.accounts[addr] = struct{}{}
	}
	accSet.cache = nil
}

func (accSet *accountSet) RemoveAccount(key account.Address) {
	delete(accSet.accounts,key)
}

// flatten returns the list of addresses within this set, also caching it for later
// reuse. The returned slice should not be changed!
func (as *accountSet) flatten() []account.Address {
	if as.cache == nil {
		accounts := make([]account.Address, 0, len(as.accounts))
		for acc := range as.accounts {
			accounts = append(accounts, acc)
		}
		as.cache = &accounts
	}
	return *as.cache
}


type txLookup struct{
	slots       int
	lock        sync.RWMutex
	locals      map[transaction.TxKey]*transaction.Transaction
	remotes     map[transaction.TxKey]*transaction.Transaction
}
func (t *txLookup) Range(f func(key transaction.TxKey, tx *transaction.Transaction, local bool) bool, local bool, remote bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	if local {
		for k, v := range t.locals {
			if !f(k, v, true) {
				return
			}
		}
	}
	if remote {
		for k, v := range t.remotes {
			if !f(k, v, false) {
				return
			}
		}
	}
}

func newTxLookup() *txLookup {
	return &txLookup{
		locals:  make(map[transaction.TxKey]*transaction.Transaction),
		remotes: make(map[transaction.TxKey]*transaction.Transaction),
	}
}
func(t *txLookup)Get(key transaction.TxKey) *transaction.Transaction{
	t.lock.RLock()
	defer t.lock.RUnlock()
	if tx := t.locals[key];tx != nil {
		return tx
	}
	return t.remotes[key]
}

// GetLocal returns a transaction if it exists in the lookup, or nil if not found.
func (t *txLookup) GetLocal(key transaction.TxKey) *transaction.Transaction {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.locals[key]
}

// GetRemote returns a transaction if it exists in the lookup, or nil if not found.
func (t *txLookup) GetRemote(key transaction.TxKey) *transaction.Transaction {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.remotes[key]
}

func(t *txLookup)Count() int{
	t.lock.RLock()
	defer t.lock.RUnlock()
	return len(t.locals)+len(t.remotes)
}

// LocalCount returns the current number of local transactions in the lookup.
func (t *txLookup) LocalCount() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return len(t.locals)
}

// RemoteCount returns the current number of remote transactions in the lookup.
func (t *txLookup) RemoteCount() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return len(t.remotes)
}

// Slots returns the current number of Quota used in the lookup.
func (t *txLookup)Slots() int{
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.slots
}

func(t *txLookup) Add(tx *transaction.Transaction,local bool){
	t.lock.Lock()
	defer t.lock.Unlock()
	t.slots += numSlots(tx)
	if txId,err := tx.TxID();err != nil {
		if local {
			t.locals[txId] = tx
		}else {
			t.remotes[txId] = tx
		}
	}
}
func(t *txLookup)Remove(key transaction.TxKey){
	t.lock.Lock()
	defer t.lock.Unlock()
	tx,ok := t.locals[key]
	if !ok{
		tx,ok = t.remotes[key]
	}
	if !ok {
		//log.Error("No transaction found to be deleted", "hash", hash)
		//no achieved for log
		return
	}
	t.slots -= numSlots(tx)
	delete(t.locals, key)
	delete(t.remotes, key)
}


// RemoteToLocals migrates the transactions belongs to the given locals to locals
// set. The assumption is held the locals set is thread-safe to be used.
func (t *txLookup) RemoteToLocals(locals *accountSet) int {
	t.lock.Lock()
	defer t.lock.Unlock()

	var migrated int
	for key, tx := range t.remotes {
		if locals.containsTx(tx) {
			t.locals[key] = tx
			delete(t.remotes, key)
			migrated += 1
		}
	}
	return migrated
}



// numSlots calculates the number of slots needed for a single transaction.
func numSlots(tx *transaction.Transaction) int {
	return int((tx.Size() + txSlotSize - 1) / txSlotSize)
}
