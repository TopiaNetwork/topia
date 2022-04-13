package transactionpool

import (
	"container/heap"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/transaction/basic"
	"time"
)

// truncatePending removes transactions from the pending queue if the pool is above the
// pending limit. The algorithm tries to reduce transaction counts by an approximately
// equal number for all for accounts with many pending transactions.
func (pool *transactionPool) truncatePending(category basic.TransactionCategory) {

	pending := uint64(0)
	for _, list := range pool.pendings.getAddrTxListOfCategory(category) {
		pending += uint64(list.Len())
	}
	if pending <= pool.config.PendingGlobalSegments {
		return
	}

	var greyAccounts map[tpcrtypes.Address]int
	// Assemble a spam order to penalize large transactors first

	for addr, list := range pool.pendings.getAddrTxListOfCategory(category) {
		// Only evict transactions from high rollers
		if !pool.locals.contains(addr) && uint64(list.Len()) > pool.config.PendingAccountSegments {
			greyAccounts[addr] = list.Len()
		}
	}
	if len(greyAccounts) > 0 {
		greyAccountsQueue := make(CntAccountHeap, len(greyAccounts))
		i := 0
		for accAddr, cnt := range greyAccounts {
			greyAccountsQueue[i] = &CntAccountItem{
				accountAddr: accAddr,
				cnt:         cnt,
				index:       i,
			}
			i++
		}
		heap.Init(&greyAccountsQueue)

		//The accounts with the most backlogged transactions are first dumped
		for pending > pool.config.PendingGlobalSegments && len(greyAccounts) > 0 {
			bePunished := heap.Pop(&greyAccountsQueue).(*CntAccountItem)
			caps := pool.pendings.getTxListCapsByAddrOfCategory(category, bePunished.accountAddr)
			for _, tx := range caps {
				txId, _ := tx.HashHex()
				pool.allTxsForLook.getAllTxsLookupByCategory(category).Remove(txId)
				pool.log.Tracef("Removed fairness-exceeding pending transaction", "txKey", txId)
			}
			pool.sortedLists.getPricedlistByCategory(category).Removed(len(caps))
		}
		pending--
	}

}

// truncateQueue drops the older transactions in the queue if the pool is above the global queue limit.
func (pool *transactionPool) truncateQueue(category basic.TransactionCategory) {

	queued := uint64(0)
	for _, list := range pool.queues.getAddrTxListOfCategory(category) {
		queued += uint64(list.Len())
	}
	if queued <= pool.config.QueueMaxTxsGlobal {
		return
	}

	// Sort all accounts with queued transactions by heartbeat
	txs := make(txsByHeartbeat, 0, len(pool.queues.getAddrTxListOfCategory(category)))
	f1 := func(address tpcrtypes.Address) bool {
		return !pool.locals.contains(address)
	}
	f2 := func(string2 string) time.Time {
		return pool.ActivationIntervals.getTxActivByKey(string2)
	}
	if txs = pool.queues.addTxsForTruncateQueue(category, f1, f2); txs != nil {
		// Drop transactions until the total is below the limit or only locals remain
		for cnt := queued; cnt > pool.config.QueueMaxTxsGlobal && len(txs) > 0; {
			tx := txs[len(txs)-1]
			txs = txs[:len(txs)-1]
			txId := tx.txId
			pool.RemoveTxByKey(txId)
			cnt -= 1
			continue
		}
	}

}
