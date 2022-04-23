package transactionpool

import (
	"math"
	"time"

	"github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/transaction/basic"
)

type txPoolResetHeads struct {
	oldBlockHead, newBlockHead *types.BlockHead
}

func (pool *transactionPool) ReorgTxpoolLoop() {
	defer pool.wg.Done()
	var (
		currentDone   chan struct{} // non-nil while runReorg is active
		nextDone      = make(chan struct{})
		launchNextRun bool
		reset         *txPoolResetHeads
		dirtyAccounts *accountSet
	)
	for {
		if currentDone == nil && launchNextRun {
			go pool.runReorgTxpool(nextDone, reset, dirtyAccounts)
			currentDone, nextDone = nextDone, make(chan struct{})
			launchNextRun = false
			reset, dirtyAccounts = nil, nil
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

		case req := <-pool.chanReqPromote:
			if dirtyAccounts == nil {
				dirtyAccounts = req
			} else {
				dirtyAccounts.merge(req)
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

func (pool *transactionPool) runReorgTxpool(done chan struct{}, reset *txPoolResetHeads, dirtyAccounts *accountSet) {

	defer close(done)
	if reset != nil {
		pool.Reset(reset.oldBlockHead, reset.newBlockHead)
	}

	if reset != nil {
		for category, _ := range pool.allTxsForLook.getAll() {
			pool.demoteUnexecutables(category) //demote transactions
			if reset.newBlockHead != nil {
				pool.sortedLists.ReheapForPricedlistByCategory(category)
			}
			pool.pendings.noncesForAddrTxListOfCategory(category)
		}
	}
	for category, _ := range pool.allTxsForLook.getAll() {
		pool.truncatePendingByCategory(category)
		pool.truncateQueueByCategory(category)
	}
	pool.changesSinceReorg = 0 // Reset change counter
}

func (pool *transactionPool) requestReset(oldBlockHead *types.BlockHead, newBlockHead *types.BlockHead) chan struct{} {
	select {
	case pool.chanReqReset <- &txPoolResetHeads{oldBlockHead, newBlockHead}:
		return <-pool.chanReorgDone
	case <-pool.chanReorgShutdown:
		return pool.chanReorgShutdown
	}
}

func (pool *transactionPool) Reset(oldBlockHead, newBlockHead *types.BlockHead) error {
	defer func(t0 time.Time) {
		pool.log.Infof("reset cost time: ", time.Since(t0))
	}(time.Now())
	var reInject []*basic.Transaction
	if oldBlockHead != nil && types.BlockHash(oldBlockHead.Hash) != types.BlockHash(newBlockHead.Hash) {

		oldBlockHeight := oldBlockHead.GetHeight()
		newBlockHeight := newBlockHead.GetHeight()

		if depth := uint64(math.Abs(float64(oldBlockHeight) - float64(newBlockHeight))); depth > 64 {
			pool.log.Debugf("Skipping deep transaction reorg,", "depth:", depth)
		} else {

			var discarded, included []*basic.Transaction
			var (
				rem = pool.query.GetBlock(types.BlockHash(oldBlockHead.Hash), oldBlockHead.Height)
				add = pool.query.GetBlock(types.BlockHash(newBlockHead.Hash), newBlockHead.Height)
			)
			if rem == nil {

				if newBlockHeight >= oldBlockHeight {

					pool.log.Warnf("Transcation pool reset with missing oldhead",
						"oldhead hash", types.BlockHash(oldBlockHead.Hash),
						"newhead hash", types.BlockHash(newBlockHead.Hash))
					return nil
				}

				pool.log.Debugf("Skipping transaction reset caused by setHead",
					"oldhead hash", types.BlockHash(oldBlockHead.Hash), "oldnum", oldBlockHeight,
					"newhead hash", types.BlockHash(newBlockHead.Hash), "newnum", newBlockHeight)
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
						pool.log.Errorf("UnRooted old chain seen by tx pool", "block", oldBlockHead.Height,
							"hash", types.BlockHash(oldBlockHead.Hash))
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
						pool.log.Errorf("UnRooted new chain seen by tx pool", "block", newBlockHead.Height,
							"hash", types.BlockHash(newBlockHead.Hash))
						return ErrUnRooted
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
						pool.log.Errorf("UnRooted old chain seen by tx pool", "block", oldBlockHead.Height,
							"hash", types.BlockHash(oldBlockHead.Hash))
						return ErrUnRooted
					}
					for _, tx := range add.Data.Txs {
						var txType *basic.Transaction
						err := pool.marshaler.Unmarshal(tx, &txType)
						if err != nil {
							included = append(included, txType)
						}
					}
					if add = pool.query.GetBlock(types.BlockHash(add.Head.ParentBlockHash), add.Head.Height-1); add == nil {
						pool.log.Errorf("UnRooted new chain seen by tx pool", "block", newBlockHead.Height,
							"hash", types.BlockHash(newBlockHead.Hash))
						return ErrUnRooted
					}
				}

				reInject = basic.TxDifference(discarded, included)
			}
		}
	}
	if newBlockHead == nil {
		newBlockHead = pool.query.CurrentBlock().GetHead()
	}
	stateDb, err := pool.query.StateAt(types.BlockHash(newBlockHead.Hash))
	if err != nil {
		pool.log.Errorf("Failed to reset txPool state", "err", err)
		return err
	}
	pool.curState = *stateDb
	pool.log.Debugf("ReInjecting stale transactions", "count", len(reInject))
	return nil
}
