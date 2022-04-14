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
			pool.allTxsForLook.removeTxHashFromAllTxsLookupByCategory(category, txId)
			pool.log.Tracef("Removed fairness-exceeding pending transaction", "txKey", txId)
		}
		for pendingCnt > pool.config.PendingGlobalSegments && len(greyAccounts) > 0 {
			bePunished := heap.Pop(&greyAccountsQueue).(*CntAccountItem)
			lenCaps := pool.pendings.truncatePendingByCategoryFun3(f31, category, bePunished.accountAddr)
			pool.sortedLists.removedPricedlistByCategory(category, lenCaps)
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
	f1 := func(address tpcrtypes.Address) bool {
		return !pool.locals.contains(address)
	}
	f2 := func(string2 string) time.Time {
		return pool.ActivationIntervals.getTxActivByKey(string2)
	}
	f3 := func(category basic.TransactionCategory, key string) *basic.Transaction {
		return pool.allTxsForLook.getTxFromKeyFromAllTxsLookupByCategory(category, key)
	}
	f4 := func(category basic.TransactionCategory, txId string) {
		pool.allTxsForLook.removeTxHashFromAllTxsLookupByCategory(category, txId)
		// Remove it from the list of sortedByPriced
		pool.sortedLists.removedPricedlistByCategory(category, 1)
		//data := "txPool remove a " + string(category) + "tx,txHash is " + key
		//eventhub.GetEventHubManager().GetEventHub(pool.nodeId).Trig(pool.ctx, eventhub.EventName_TxReceived, data)

	}
	f5 := func(f51 func(txId string, tx *basic.Transaction), tx *basic.Transaction, category basic.TransactionCategory, addr tpcrtypes.Address) {
		pool.pendings.getTxListRemoveByAddrOfCategory(f51, tx, category, addr)
	}
	f511 := func(category basic.TransactionCategory, txid string) {
		pool.allTxsForLook.getAllTxsLookupByCategory(category).Remove(txid)
		pool.sortedLists.getPricedlistByCategory(category).Removed(1)
	}
	f512 := func(category basic.TransactionCategory, txId string) *basic.Transaction {
		return pool.allTxsForLook.getTxFromKeyFromAllTxsLookupByCategory(category, txId)
	}
	f513 := func(key string) {
		pool.log.Errorf("Missing transaction in lookup set, please report the issue", "TxID", key)
	}
	f514 := func(txId string, category basic.TransactionCategory) {
		pool.ActivationIntervals.setTxActiv(txId, time.Now())
		pool.TxHashCategory.setHashCat(txId, category)
	}
	f6 := func(txId string) {
		pool.ActivationIntervals.removeTxActiv(txId)
		pool.TxHashCategory.removeHashCat(txId)
	}
	pool.queues.removeTxsForTruncateQueue(category, f1, f2, f3, f4, f5, f511, f512, f513, f514, f6,
		queued, pool.config.QueueMaxTxsGlobal)

}
