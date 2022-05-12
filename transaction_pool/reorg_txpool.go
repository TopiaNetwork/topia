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
	)
	for {
		if currentDone == nil && launchNextRun {
			go pool.runReorgTxpool(nextDone, reset)
			currentDone, nextDone = nextDone, make(chan struct{})
			launchNextRun = false
			reset = nil
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

func (pool *transactionPool) runReorgTxpool(done chan struct{}, reset *txPoolResetHeads) {

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
	pool.txState.Purge()
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
	if oldBlockHead != nil && types.BlockHash(oldBlockHead.Hash) != types.BlockHash(newBlockHead.Hash) {

		oldBlockHeight := oldBlockHead.GetHeight()
		newBlockHeight := newBlockHead.GetHeight()

		if depth := uint64(math.Abs(float64(oldBlockHeight) - float64(newBlockHeight))); depth > 64 {
			pool.log.Debugf("Skipping deep transaction reorg,", "depth:", depth)
		} else {

			var discarded, included []*basic.Transaction
			var (
				rem = pool.query.GetBlockByHash(types.BlockHash(oldBlockHead.Hash))
				add = pool.query.GetBlockByHash(types.BlockHash(newBlockHead.Hash))
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
					if rem = pool.query.GetBlockByHash(types.BlockHash(rem.Head.ParentBlockHash)); rem == nil {
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
					if add = pool.query.GetBlockByHash(types.BlockHash(add.Head.ParentBlockHash)); add == nil {
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
					if rem = pool.query.GetBlockByHash(types.BlockHash(rem.Head.ParentBlockHash)); rem == nil {
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
					if add = pool.query.GetBlockByHash(types.BlockHash(add.Head.ParentBlockHash)); add == nil {
						pool.log.Errorf("UnRooted new chain seen by tx pool", "block", newBlockHead.Height,
							"hash", types.BlockHash(newBlockHead.Hash))
						return ErrUnRooted
					}
				}

			}
		}
	}
	if newBlockHead == nil {
		newBlockHead = pool.query.CurrentBlock().GetHead()
	}
	return nil
}
