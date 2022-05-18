package transactionpool

import (
	"container/heap"
	"time"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/eventhub"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

func (pool *transactionPool) truncatePendingByCategory(category txbasic.TransactionCategory) {

	pendingSize := pool.pendings.PendingSizeByCategory(category)
	if pendingSize <= pool.config.MaxSizeOfPending {
		return
	}

	// Assemble a spam order to penalize large transactors first
	greyAccounts := pool.pendings.truncatePendingByCategoryFun2(category, pool.config.MaxSizeOfEachPendingAccount)

	greyAccountsQueue := make(SizeAccountHeap, len(greyAccounts))
	i := 0
	for accAddr, size := range greyAccounts {
		greyAccountsQueue[i] = &SizeAccountItem{
			accountAddr: accAddr,
			size:        size,
			index:       i,
		}
		i++
	}
	heap.Init(&greyAccountsQueue)
	if len(greyAccounts) > 0 {
		//The accounts with the most backlogged transactions are first dumped
		f31 := func(category txbasic.TransactionCategory, txId txbasic.TxID) {
			pool.allTxsForLook.removeTxHashFromAllTxsLookupByCategory(category, txId)
			pool.log.Tracef("Removed fairness-exceeding pending transaction", "txKey", txId)
		}
		for pendingSize > pool.config.MaxSizeOfPending && len(greyAccounts) > 0 {

			bePunished := heap.Pop(&greyAccountsQueue).(*SizeAccountItem)
			sizeCaps := pool.pendings.truncatePendingByCategoryFun3(f31, category, bePunished.accountAddr)
			pool.sortedLists.removedPricedlistByCategory(category, sizeCaps)
			pendingSize -= uint64(bePunished.size)
		}

	}

}

// truncateQueue drops the older transactions in the queue if the pool is above the global queue limit.
func (pool *transactionPool) truncateQueueByCategory(category txbasic.TransactionCategory) {
	queueSize := pool.queues.getSizeOfCategory(category)
	if uint64(queueSize) <= pool.config.MaxSizeOfQueue {
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
		pool.allTxsForLook.removeTxHashFromAllTxsLookupByCategory(category, txId)
		// Remove it from the list of sortedByPriced
		pool.sortedLists.removedPricedlistByCategory(category, tx.Size())
		txRemoved := &eventhub.TxPoolEvent{
			EvType: eventhub.TxPoolEVTypee_Removed,
			Tx:     tx,
		}
		eventhub.GetEventHubManager().GetEventHub(pool.nodeId).Trig(pool.ctx, eventhub.EventName_TxPoolChanged, txRemoved)
	}
	f5 := func(f51 func(txId txbasic.TxID, tx *txbasic.Transaction), tx *txbasic.Transaction, category txbasic.TransactionCategory, addr tpcrtypes.Address) {
		pool.pendings.getTxListRemoveByAddrOfCategory(f51, tx, category, addr)
	}
	f511 := func(category txbasic.TransactionCategory, txid txbasic.TxID, tx *txbasic.Transaction) {
		pool.allTxsForLook.removeTxHashFromAllTxsLookupByCategory(category, txid)
		pool.sortedLists.removedPricedlistByCategory(category, tx.Size())
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
		queueSize, pool.config.MaxSizeOfQueue)

}

func (pool *transactionPool) TruncateTxPool() {
	for category, _ := range pool.allTxsForLook.all {
		pool.truncatePendingByCategory(category)
		pool.truncateQueueByCategory(category)
	}
	pool.log.Tracef("TransactionPool Truncated")

}
