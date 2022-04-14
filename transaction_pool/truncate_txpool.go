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

	pendingCnt := pool.pendings.truncatePendingByCategoryFun1(category)
	if pendingCnt <= pool.config.PendingGlobalSegments {
		return
	}

	// Assemble a spam order to penalize large transactors first
	f21 := func(address tpcrtypes.Address) bool { return !pool.locals.contains(address) }
	greyAccounts := pool.pendings.truncatePendingByCategoryFun2(category, f21, pool.config.PendingAccountSegments)

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
		f31 := func(category basic.TransactionCategory, txId string) {
			pool.allTxsForLook.getAllTxsLookupByCategory(category).Remove(txId)
			pool.log.Tracef("Removed fairness-exceeding pending transaction", "txKey", txId)
		}
		for pendingCnt > pool.config.PendingGlobalSegments && len(greyAccounts) > 0 {
			bePunished := heap.Pop(&greyAccountsQueue).(*CntAccountItem)
			lenCaps := pool.pendings.truncatePendingByCategoryFun3(f31, category, bePunished.accountAddr)
			pool.sortedLists.getPricedlistByCategory(category).Removed(lenCaps)
		}
		pendingCnt--
	}

}

// truncateQueue drops the older transactions in the queue if the pool is above the global queue limit.
func (pool *transactionPool) truncateQueue(category basic.TransactionCategory) {
	queued := pool.queues.getStatsOfCategory(category)
	if uint64(queued) <= pool.config.QueueMaxTxsGlobal {
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
		for cnt := uint64(queued); cnt > pool.config.QueueMaxTxsGlobal && len(txs) > 0; {
			tx := txs[len(txs)-1]
			txs = txs[:len(txs)-1]
			txId := tx.txId
			pool.RemoveTxByKey(txId)
			cnt -= 1
			continue
		}
	}

}
