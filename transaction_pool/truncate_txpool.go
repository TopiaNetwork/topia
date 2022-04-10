package transactionpool

import (
	"container/heap"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/transaction/basic"
	"sort"
)

// truncatePending removes transactions from the pending queue if the pool is above the
// pending limit. The algorithm tries to reduce transaction counts by an approximately
// equal number for all for accounts with many pending transactions.
func (pool *transactionPool) truncatePending(category basic.TransactionCategory) {
	pool.pendings.Mu.RLock()
	defer pool.pendings.Mu.RUnlock()
	pending := uint64(0)
	for _, list := range pool.pendings.pending[category] {
		pending += uint64(list.Len())
	}
	if pending <= pool.config.PendingGlobalSegments {
		return
	}

	var greyAccounts map[tpcrtypes.Address]int
	// Assemble a spam order to penalize large transactors first

	for addr, list := range pool.pendings.pending[category] {
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
			list := pool.pendings.pending[category][bePunished.accountAddr]
			caps := list.Cap(list.Len() - 1)
			for _, tx := range caps {
				txId, _ := tx.HashHex()
				pool.allTxsForLook.all[category].Remove(txId)
				pool.log.Tracef("Removed fairness-exceeding pending transaction", "txKey", txId)
			}
			pool.sortedLists.Pricedlist[category].Removed(len(caps))
		}
		pending--
	}

}

// truncateQueue drops the older transactions in the queue if the pool is above the global queue limit.
func (pool *transactionPool) truncateQueue(category basic.TransactionCategory) {
	pool.queues.Mu.RLock()
	defer pool.queues.Mu.RUnlock()
	queued := uint64(0)
	for _, list := range pool.queues.queue[category] {
		queued += uint64(list.Len())
	}
	if queued <= pool.config.QueueMaxTxsGlobal {
		return
	}

	// Sort all accounts with queued transactions by heartbeat
	txs := make(txsByHeartbeat, 0, len(pool.queues.queue[category]))
	for addr := range pool.queues.queue[category] {
		if !pool.locals.contains(addr) { // don't drop locals
			list := pool.queues.queue[category][addr].Flatten()
			for _, tx := range list {
				txId, _ := tx.HashHex()
				txs = append(txs, txByHeartbeat{txId, pool.ActivationIntervals.activ[txId]})
			}
		}
	}
	sort.Sort(txs)

	// Drop transactions until the total is below the limit or only locals remain
	for drop := queued - pool.config.QueueMaxTxsGlobal; drop > 0 && len(txs) > 0; {
		tx := txs[len(txs)-1]
		txs = txs[:len(txs)-1]
		txId := tx.tx
		pool.RemoveTxByKey(txId)
		drop -= 1
		continue
	}
}
