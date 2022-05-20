package transactionpool

import (
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"math"
	"time"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type txPoolResetHeads struct {
	oldBlockHead, newBlockHead *tpchaintypes.BlockHead
}

func (pool *transactionPool) ReorgAndTurnTxLoop() {
	defer pool.wg.Done()
	var (
		currentDone   chan struct{} // non-nil while runReorg is active
		nextDone      = make(chan struct{})
		launchNextRun bool
		reset         *txPoolResetHeads
		accountsCache *accountCache
	)
	for {
		if currentDone == nil && launchNextRun {
			go pool.runReorgTxpool(nextDone, reset, accountsCache)
			currentDone, nextDone = nextDone, make(chan struct{})
			launchNextRun = false
			reset, accountsCache = nil, nil
		}

		select {
		case req := <-pool.chanReqReset:
			if reset == nil {
				reset = req
			} else {
				reset.newBlockHead = req.newBlockHead
			}
			launchNextRun = true
			pool.chanReorgDone <- nextDone
		case req := <-pool.chanReqTurn:
			// Promote request: update address set if request is already pending.
			if accountsCache == nil {
				accountsCache = req
			} else {
				accountsCache.merge(req)
			}
			launchNextRun = true
			pool.chanReorgDone <- nextDone

		case <-currentDone:
			currentDone = nil

		case <-pool.chanReorgShutdown:
			// Wait for current run to finish.
			if currentDone != nil {
				<-currentDone
			}
			close(nextDone)
			return
		}
	}
}

func (pool *transactionPool) runReorgTxpool(done chan struct{}, reset *txPoolResetHeads, accountCache *accountCache) {

	defer close(done)

	var addrsNeedTurn []tpcrtypes.Address
	if accountCache != nil && reset == nil {
		addrsNeedTurn = accountCache.flatten()
	}

	if reset != nil {
		pool.Reset(reset.oldBlockHead, reset.newBlockHead)
		// Reset needs turn all addresses to pending
		addrsNeedTurn = pool.queues.getAllAddress()
	}

	pool.turnAddrTxsToPending(addrsNeedTurn)

	if reset != nil {
		for category, _ := range pool.allTxsForLook.getAll() {
			pool.demoteUnexecutables(category) //demote transactions
		}
	}
	for category, _ := range pool.allTxsForLook.getAll() {
		pool.truncatePendingByCategory(category)
		pool.truncateQueueByCategory(category)
	}
	pool.changeSizeSinceReorg = 0 // Reset change counter
}

func (pool *transactionPool) requestReset(oldBlockHead *tpchaintypes.BlockHead, newBlockHead *tpchaintypes.BlockHead) chan struct{} {
	select {
	case pool.chanReqReset <- &txPoolResetHeads{oldBlockHead, newBlockHead}:
		return <-pool.chanReorgDone
	case <-pool.chanReorgShutdown:
		return pool.chanReorgShutdown
	}
}
func (pool *transactionPool) requestTurnToPending(cache *accountCache) chan struct{} {
	select {
	case pool.chanReqTurn <- cache:
		return <-pool.chanReorgDone
	case <-pool.chanReorgShutdown:
		return pool.chanReorgShutdown
	}
}

func (pool *transactionPool) Reset(oldBlockHead, newBlockHead *tpchaintypes.BlockHead) error {
	defer func(t0 time.Time) {
		pool.log.Infof("reset cost time: ", time.Since(t0))
	}(time.Now())

	var reinjectTxs []*txbasic.Transaction
	if oldBlockHead != nil && tpchaintypes.BlockHash(oldBlockHead.Hash) != tpchaintypes.BlockHash(newBlockHead.Hash) {

		oldBlockHeight := oldBlockHead.GetHeight()
		newBlockHeight := newBlockHead.GetHeight()

		if depth := uint64(math.Abs(float64(oldBlockHeight) - float64(newBlockHeight))); depth > 64 {
			pool.log.Debugf("Skipping deep transaction reorg,", "depth:", depth)
		} else {

			var curTxPoolTxs, packagedTx []*txbasic.Transaction
			var (
				rem, _ = pool.txServant.BlockByHash(tpchaintypes.BlockHash(oldBlockHead.Hash))
				add, _ = pool.txServant.BlockByHash(tpchaintypes.BlockHash(newBlockHead.Hash))
			)
			if rem == nil {

				if newBlockHeight >= oldBlockHeight {

					pool.log.Warnf("Transcation pool reset with missing oldhead",
						"oldhead hash", tpchaintypes.BlockHash(oldBlockHead.Hash),
						"newhead hash", tpchaintypes.BlockHash(newBlockHead.Hash))
					return nil
				}

				pool.log.Debugf("Skipping transaction reset caused by setHead",
					"oldhead hash", tpchaintypes.BlockHash(oldBlockHead.Hash), "oldnum", oldBlockHeight,
					"newhead hash", tpchaintypes.BlockHash(newBlockHead.Hash), "newnum", newBlockHeight)
			} else {

				for rem.Head.Height > add.Head.Height {

					for _, tx := range rem.Data.Txs {
						var txType *txbasic.Transaction
						err := pool.marshaler.Unmarshal(tx, &txType)
						if err != nil {
							curTxPoolTxs = append(curTxPoolTxs, txType)
						}
					}
					if rem, _ = pool.txServant.BlockByHash(tpchaintypes.BlockHash(rem.Head.ParentBlockHash)); rem == nil {
						pool.log.Errorf("UnRooted old chain seen by tx pool", "block", oldBlockHead.Height,
							"hash", tpchaintypes.BlockHash(oldBlockHead.Hash))
						return nil
					}
				}

				for add.Head.Height > rem.Head.Height {
					for _, tx := range add.Data.Txs {
						var txType *txbasic.Transaction
						err := pool.marshaler.Unmarshal(tx, &txType)
						if err != nil {
							packagedTx = append(packagedTx, txType)
						}
					}
					if add, _ = pool.txServant.BlockByHash(tpchaintypes.BlockHash(add.Head.ParentBlockHash)); add == nil {
						pool.log.Errorf("UnRooted new chain seen by tx pool", "block", newBlockHead.Height,
							"hash", tpchaintypes.BlockHash(newBlockHead.Hash))
						return ErrUnRooted
					}
				}

				for tpchaintypes.BlockHash(rem.Head.Hash) != tpchaintypes.BlockHash(add.Head.Hash) {
					for _, tx := range rem.Data.Txs {
						var txType *txbasic.Transaction
						err := pool.marshaler.Unmarshal(tx, &txType)
						if err != nil {
							curTxPoolTxs = append(curTxPoolTxs, txType)
						}
					}
					if rem, _ = pool.txServant.BlockByHash(tpchaintypes.BlockHash(rem.Head.ParentBlockHash)); rem == nil {
						pool.log.Errorf("UnRooted old chain seen by tx pool", "block", oldBlockHead.Height,
							"hash", tpchaintypes.BlockHash(oldBlockHead.Hash))
						return ErrUnRooted
					}
					for _, tx := range add.Data.Txs {
						var txType *txbasic.Transaction
						err := pool.marshaler.Unmarshal(tx, &txType)
						if err != nil {
							packagedTx = append(packagedTx, txType)
						}
					}
					if add, _ = pool.txServant.BlockByHash(tpchaintypes.BlockHash(add.Head.ParentBlockHash)); add == nil {
						pool.log.Errorf("UnRooted new chain seen by tx pool", "block", newBlockHead.Height,
							"hash", tpchaintypes.BlockHash(newBlockHead.Hash))
						return ErrUnRooted
					}
				}
				reinjectTxs = TxDifferenceList(curTxPoolTxs, packagedTx)
			}
		}
	}
	if newBlockHead == nil {
		curblock, _ := pool.txServant.LatestBlock()
		newBlockHead = curblock.GetHead()
	}

	pool.addTxsLocked(reinjectTxs, false)

	return nil
}

//TxDifferenceList Delete set B from set A, return the differenct transaction list
func TxDifferenceList(a, b []*txbasic.Transaction) []*txbasic.Transaction {
	keep := make([]*txbasic.Transaction, 0, len(a))

	remove := make(map[txbasic.TxID]struct{})
	for _, tx := range b {
		txid, _ := tx.TxID()
		remove[txid] = struct{}{}
	}

	for _, tx := range a {
		txid, _ := tx.TxID()
		if _, ok := remove[txid]; !ok {
			keep = append(keep, tx)
		}
	}

	return keep
}
