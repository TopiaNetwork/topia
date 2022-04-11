package transactionpool

import (
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"math"

	"github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/transaction/basic"
)

type txPoolResetRequest struct {
	oldHead, newHead *types.BlockHead
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
		queuedEvents  = make(map[tpcrtypes.Address]*txSortedMap)
	)
	for {
		// Launch next background reorg if needed
		if curDone == nil && launchNextRun {
			// Run the background reorg and announcements
			go pool.runReorg(nextDone, reset, dirtyAccounts, queuedEvents)

			// Prepare everything for the next round of reorg
			curDone, nextDone = nextDone, make(chan struct{})
			launchNextRun = false

			reset, dirtyAccounts = nil, nil
			queuedEvents = make(map[tpcrtypes.Address]*txSortedMap)
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
			addr := tpcrtypes.Address(tx.Head.FromAddr)
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
func (pool *transactionPool) runReorg(done chan struct{}, reset *txPoolResetRequest, dirtyAccounts *accountSet, events map[tpcrtypes.Address]*txSortedMap) {
	defer close(done)
	for category, _ := range pool.pendings.pending {
		var replaceAddrs []tpcrtypes.Address
		if dirtyAccounts != nil && reset == nil {
			// Only dirty accounts need to be promoted, unless we're resetting.
			// For resets, all addresses in the tx queue will be promoted and
			// the flatten operation can be avoided.
			replaceAddrs = dirtyAccounts.flatten()
		}
		if reset != nil {
			// Reset from the old head to the new, rescheduling any reorged transactions
			pool.Reset(reset.oldHead, reset.newHead)

			// Nonces were reset, discard any events that became stale
			for addr := range events {
				events[addr].Forward(pool.curState.GetNonce(addr))
				if events[addr].Len() == 0 {
					delete(events, addr)
				}
			}
			// Reset needs promote for all addresses
			replaceAddrs = make([]tpcrtypes.Address, 0, len(pool.queues.getQueueTxsByCategory(category).addrTxList))
			for addr, _ := range pool.queues.getQueueTxsByCategory(category).addrTxList {
				replaceAddrs = append(replaceAddrs, addr)
			}
		}
		// Check for pending transactions for every account that sent new ones
		promoted := pool.replaceExecutables(category, replaceAddrs)

		// If a new block appeared, validate the pool of pending transactions. This will
		// remove any transaction that has been included in the block or was invalidated
		// because of another transaction (e.g. higher gas price).
		if reset != nil {
			pool.demoteUnexecutables(category) //demote transactions
			if reset.newHead != nil {
				pool.sortedLists.getPricedlistByCategory(category).Reheap()
			}
			// Update all accounts to the latest known pending nonce
			nonces := make(map[tpcrtypes.Address]uint64, len(pool.pendings.getPendingTxsByCategory(category).addrTxList))
			for addr, list := range pool.pendings.getAddrTxListOfCategory(category) {
				highestPending := list.LastElement()
				Noncei := highestPending.Head.Nonce
				nonces[addr] = Noncei + 1
			}
		}
		// Ensure pool.queue and pool.pending sizes stay within the configured limits.
		pool.truncatePending(category)
		pool.truncateQueue(category)

		pool.changesSinceReorg = 0 // Reset change counter

		// Notify subsystems for newly added transactions
		for _, tx := range promoted {
			addr := tpcrtypes.Address(tx.Head.FromAddr)
			if _, ok := events[addr]; !ok {
				events[addr] = newTxSortedMap()
			}
			events[addr].Put(tx)
		}
		if len(events) > 0 {
			var txs []*basic.Transaction
			for _, set := range events {
				txs = append(txs, set.Flatten()...)
			}
			for _, tx := range txs {
				pool.BroadCastTx(tx)
			}
		}
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

func (pool *transactionPool) Reset(oldHead, newHead *types.BlockHead) error {
	//If the old header and the new header do not meet certain conditions,
	//part of the transaction needs to be injected back into the transaction pool
	var reInject []*basic.Transaction
	if oldHead != nil && types.BlockHash(oldHead.Hash) != types.BlockHash(newHead.Hash) {

		oldNum := oldHead.GetHeight()
		newNum := newHead.GetHeight()

		//If the difference between the old block and the new block is greater than 64
		//then no recombination is carried out
		if depth := uint64(math.Abs(float64(oldNum) - float64(newNum))); depth > 64 {

			pool.log.Debugf("Skipping deep transaction reorg", "depth", depth)
		} else {

			//The reorganization looks shallow enough to put all the transactions into memory
			var discarded, included []*basic.Transaction
			var (
				rem = pool.query.GetBlock(types.BlockHash(oldHead.Hash), oldHead.Height)
				add = pool.query.GetBlock(types.BlockHash(newHead.Hash), newHead.Height)
			)
			if rem == nil {

				if newNum >= oldNum {

					pool.log.Warnf("Transcation pool reset with missing oldhead",
						"old", types.BlockHash(oldHead.Hash),
						"new", types.BlockHash(newHead.Hash))
					return nil
				}

				pool.log.Debugf("Skipping transaction reset caused by setHead",
					"old", types.BlockHash(oldHead.Hash), "oldnum", oldNum,
					"new", types.BlockHash(newHead.Hash), "newnum", newNum)
			} else {

				for rem.Head.Height > add.Head.Height {

					for _, tx := range rem.Data.Txs {
						var txType *basic.Transaction
						err := pool.marshaler.Unmarshal(tx, &txType)
						if err != nil {
							discarded = append(discarded, txType)
						}
					}
					if rem = pool.query.GetBlock(types.BlockHash(rem.Head.ParentBlockHash), rem.Head.Height-1); rem == nil {
						pool.log.Errorf("UnRooted old chain seen by tx pool", "block", oldHead.Height,
							"hash", types.BlockHash(oldHead.Hash))
						return nil
					}
				}

				for add.Head.Height > rem.Head.Height {
					for _, tx := range add.Data.Txs {
						var txType *basic.Transaction
						err := pool.marshaler.Unmarshal(tx, &txType)
						if err != nil {
							included = append(included, txType)
						}
					}
					if add = pool.query.GetBlock(types.BlockHash(add.Head.ParentBlockHash), add.Head.Height-1); add == nil {
						pool.log.Errorf("UnRooted new chain seen by tx pool", "block", newHead.Height,
							"hash", types.BlockHash(newHead.Hash))
						return nil
					}
				}

				for types.BlockHash(rem.Head.Hash) != types.BlockHash(add.Head.Hash) {
					for _, tx := range rem.Data.Txs {
						var txType *basic.Transaction
						err := pool.marshaler.Unmarshal(tx, &txType)
						if err != nil {
							discarded = append(discarded, txType)
						}
					}
					if rem = pool.query.GetBlock(types.BlockHash(rem.Head.ParentBlockHash), rem.Head.Height-1); rem == nil {
						pool.log.Errorf("UnRooted old chain seen by tx pool", "block", oldHead.Height,
							"hash", types.BlockHash(oldHead.Hash))
						return nil
					}
					for _, tx := range add.Data.Txs {
						var txType *basic.Transaction
						err := pool.marshaler.Unmarshal(tx, &txType)
						if err != nil {
							included = append(included, txType)
						}
					}
					if add = pool.query.GetBlock(types.BlockHash(add.Head.ParentBlockHash), add.Head.Height-1); add == nil {
						pool.log.Errorf("UnRooted new chain seen by tx pool", "block", newHead.Height,
							"hash", types.BlockHash(newHead.Hash))
						return nil
					}
				}

				reInject = basic.TxDifference(discarded, included)
			}
		}
	}

	// Initialize the internal state to the current head
	if newHead == nil {
		newHead = pool.query.CurrentBlock().GetHead()
	}

	stateDb, err := pool.query.StateAt(types.BlockHash(newHead.Hash))
	if err != nil {
		pool.log.Errorf("Failed to reset txPool state", "err", err)
		return nil
	}

	pool.curState = *stateDb

	pool.log.Debugf("ReInjecting stale transactions", "count", len(reInject))

	return nil
}
