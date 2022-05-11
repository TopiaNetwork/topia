package transactionpool

import (
	"container/heap"
	"time"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/eventhub"
	"github.com/TopiaNetwork/topia/transaction/basic"
)

func (pool *transactionPool) truncatePendingByCategory(category basic.TransactionCategory) {

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
		f31 := func(category basic.TransactionCategory, txId basic.TxID) {
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
func (pool *transactionPool) truncateQueueByCategory(category basic.TransactionCategory) {
	queued := pool.queues.getStatsOfCategory(category)
	if uint64(queued) <= pool.config.QueueMaxTxsGlobal {
		return
	}
	// Sort all accounts with queued transactions by heartbeat
	f2 := func(string2 basic.TxID) *timeAndHeight {
		return pool.ActivationIntervals.getTxActivByKey(string2)
	}
	f3 := func(category basic.TransactionCategory, key basic.TxID) *basic.Transaction {
		return pool.allTxsForLook.getTxFromKeyFromAllTxsLookupByCategory(category, key)
	}
	f4 := func(category basic.TransactionCategory, txId basic.TxID, tx *basic.Transaction) {
		pool.allTxsForLook.removeTxHashFromAllTxsLookupByCategory(category, txId, pool.config.TxSegmentSize)
		// Remove it from the list of sortedByPriced
		pool.sortedLists.removedPricedlistByCategory(category, 1)
		txRemoved := &eventhub.TxPoolEvent{
			EvType: eventhub.TxPoolEVTypee_Removed,
			Tx:     tx,
		}
		eventhub.GetEventHubManager().GetEventHub(pool.nodeId).Trig(pool.ctx, eventhub.EventName_TxPoolChanged, txRemoved)
	}
	f5 := func(f51 func(txId basic.TxID, tx *basic.Transaction), tx *basic.Transaction, category basic.TransactionCategory, addr tpcrtypes.Address) {
		pool.pendings.getTxListRemoveByAddrOfCategory(f51, tx, category, addr)
	}
	f511 := func(category basic.TransactionCategory, txid basic.TxID) {
		pool.allTxsForLook.removeTxHashFromAllTxsLookupByCategory(category, txid, pool.config.TxSegmentSize)
		pool.sortedLists.removedPricedlistByCategory(category, 1)
	}
	f512 := func(category basic.TransactionCategory, txId basic.TxID) *basic.Transaction {
		return pool.allTxsForLook.getTxFromKeyFromAllTxsLookupByCategory(category, txId)
	}
	f513 := func(key basic.TxID) {
		pool.log.Errorf("Missing transaction in lookup set, please report the issue", "TxID", key)
	}
	f514 := func(txId basic.TxID, category basic.TransactionCategory) {
		timeandheight := &timeAndHeight{
			time:   time.Now(),
			height: pool.query.CurrentHeight(),
		}
		pool.ActivationIntervals.setTxActiv(txId, timeandheight)
		pool.TxHashCategory.setHashCat(txId, category)
	}
	f6 := func(txId basic.TxID) {
		pool.ActivationIntervals.removeTxActiv(txId)
		pool.TxHashCategory.removeHashCat(txId)
	}
	pool.queues.removeTxsForTruncateQueue(category, f2, f3, f4, f5, f511, f512, f513, f514, f6,
		queued, pool.config.QueueMaxTxsGlobal)

}
