package transactionpool

import (
	"container/heap"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"sort"
)

// truncatePending removes transactions from the pending queue if the pool is above the
// pending limit. The algorithm tries to reduce transaction counts by an approximately
// equal number for all for accounts with many pending transactions.
func (pool *transactionPool) truncatePending() {
	pending := uint64(0)
	for _, list := range pool.pending.accTxs {
		pending += uint64(list.Len())
	}
	if pending <= pool.config.PendingGlobalSlots {
		return
	}

	var greyAccounts map[tpcrtypes.Address]int
	// Assemble a spam order to penalize large transactors first

	for addr, list := range pool.pending.accTxs {
		// Only evict transactions from high rollers
		if !pool.locals.contains(addr) && uint64(list.Len()) > pool.config.PendingAccountSlots {
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
		for pending > pool.config.PendingGlobalSlots && len(greyAccounts) > 0 {
			bePunished := heap.Pop(&greyAccountsQueue).(*CntAccountItem)
			list := pool.pending.accTxs[bePunished.accountAddr]
			caps := list.Cap(list.Len() - 1)
			for _, tx := range caps {
				txId, _ := tx.HashHex()
				pool.allTxsForLook.Remove(txId)
				pool.log.Tracef("Removed fairness-exceeding pending transaction", "txKey", txId)
			}
			pool.sortedByPriced.Removed(len(caps))
		}
		pending--
	}

}

// truncateQueue drops the older transactions in the queue if the pool is above the global queue limit.
func (pool *transactionPool) truncateQueue() {
	queued := uint64(0)
	for _, list := range pool.queue.accTxs {
		queued += uint64(list.Len())
	}
	if queued <= pool.config.QueueMaxTxsGlobal {
		return
	}

	// Sort all accounts with queued transactions by heartbeat
	txs := make(txsByHeartbeat, 0, len(pool.queue.accTxs))
	for addr := range pool.queue.accTxs {
		if !pool.locals.contains(addr) { // don't drop locals
			list := pool.queue.accTxs[addr].Flatten()
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
