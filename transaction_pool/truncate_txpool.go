package transactionpool

import (
	"container/heap"
	"time"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/eventhub"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

func (pool *transactionPool) truncatePendingByCategory(category txbasic.TransactionCategory) {

	pendingCnt := pool.pendings.truncatePendingByCategoryFun1(category)
	if pendingCnt <= pool.config.PendingGlobalSegments {
		return
	}

	// Assemble a spam order to penalize large transactors first
	greyAccounts := pool.pendings.truncatePendingByCategoryFun2(category, pool.config.PendingAccountSegments)
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
		f31 := func(category txbasic.TransactionCategory, txId txbasic.TxID) {
			pool.allTxsForLook.removeTxHashFromAllTxsLookupByCategory(category, txId, pool.config.TxSegmentSize)
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
func (pool *transactionPool) truncateQueueByCategory(category txbasic.TransactionCategory) {
	queued := pool.queues.getStatsOfCategory(category)
	if uint64(queued) <= pool.config.QueueMaxTxsGlobal {
		return
	}
	// Sort all accounts with queued transactions by heartbeat
	f2 := func(string2 txbasic.TxID) time.Time {
		return pool.ActivationIntervals.getTxActivByKey(string2)
	}
	f3 := func(category txbasic.TransactionCategory, key txbasic.TxID) *txbasic.Transaction {
		return pool.allTxsForLook.getTxFromKeyFromAllTxsLookupByCategory(category, key)
	}
	f4 := func(category txbasic.TransactionCategory, txId txbasic.TxID, tx *txbasic.Transaction) {
		pool.allTxsForLook.removeTxHashFromAllTxsLookupByCategory(category, txId, pool.config.TxSegmentSize)
		// Remove it from the list of sortedByPriced
		pool.sortedLists.removedPricedlistByCategory(category, 1)
		txRemoved := &eventhub.TxPoolEvent{
			EvType: eventhub.TxPoolEVTypee_Removed,
			Tx:     tx,
		}
		eventhub.GetEventHubManager().GetEventHub(pool.nodeId).Trig(pool.ctx, eventhub.EventName_TxPoolChanged, txRemoved)
	}
	f5 := func(f51 func(txId txbasic.TxID, tx *txbasic.Transaction), tx *txbasic.Transaction, category txbasic.TransactionCategory, addr tpcrtypes.Address) {
		pool.pendings.getTxListRemoveByAddrOfCategory(f51, tx, category, addr)
	}
	f511 := func(category txbasic.TransactionCategory, txid txbasic.TxID) {
		pool.allTxsForLook.removeTxHashFromAllTxsLookupByCategory(category, txid, pool.config.TxSegmentSize)
		pool.sortedLists.removedPricedlistByCategory(category, 1)
	}
	f512 := func(category txbasic.TransactionCategory, txId txbasic.TxID) *txbasic.Transaction {
		return pool.allTxsForLook.getTxFromKeyFromAllTxsLookupByCategory(category, txId)
	}
	f513 := func(key txbasic.TxID) {
		pool.log.Errorf("Missing transaction in lookup set, please report the issue", "TxID", key)
	}
	f514 := func(txId txbasic.TxID, category txbasic.TransactionCategory) {

		pool.ActivationIntervals.setTxActiv(txId, time.Now())
		pool.TxHashCategory.setHashCat(txId, category)
	}
	f6 := func(txId txbasic.TxID) {
		pool.ActivationIntervals.removeTxActiv(txId)
		pool.TxHashCategory.removeHashCat(txId)
	}
	pool.queues.removeTxsForTruncateQueue(category, f2, f3, f4, f5, f511, f512, f513, f514, f6,
		queued, pool.config.QueueMaxTxsGlobal)

}

func (pool *transactionPool) TruncateTxPool() {
	for category, _ := range pool.allTxsForLook.all {
		pool.truncatePendingByCategory(category)
		pool.truncateQueueByCategory(category)
	}
	pool.log.Tracef("TransactionPool Truncated")

}
