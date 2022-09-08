package transactionpool

import (
	"runtime/debug"
	"time"

	txpoolcore "github.com/TopiaNetwork/topia/transaction_pool/core"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

func (pool *transactionPool) blocksRevertRoutine() {
	go func() {
		defer func() {
			err := recover()
			if err != nil {
				pool.log.Errorf("blocksRevertRoutine err:", err, debug.Stack())
			}
		}()

		for {
			select {
			case blocks := <-pool.blockRevertCh:
				if len(blocks) > 0 {
					if blocks[0] != nil {
						pool.addTxsForBlocksRevert(blocks)
					}
				}
			case <-pool.ctx.Done():
				pool.log.Info("blocksRevertRoutine stopped")
				return
			}
		}
	}()
}

func (pool *transactionPool) expiredTxsRoutine() {
	go func() {
		defer func() {
			err := recover()
			if err != nil {
				pool.log.Errorf("expiredTxsRoutine err:", err, debug.Stack())
			}
		}()

		var timer = time.NewTicker(ExpiredTxsInterval) //30s report eviction
		defer timer.Stop()

		for {
			select {
			case <-timer.C:
				curHeight, err := pool.txServant.CurrentHeight()
				if err != nil {
					continue
				}
				pool.txsCollect.PurgeExpiredTxs(curHeight, pool.config.TimeOfTxLifecycle, pool.config.HeightOfTxLifecycle)
			case <-pool.ctx.Done():
				pool.log.Info("expiredTxsRoutine stopped")
				return
			}
		}
	}()
}

func (pool *transactionPool) republishRoutine() {
	go func() {
		defer func() {
			err := recover()
			if err != nil {
				pool.log.Errorf("republishRoutine err:", err, debug.Stack())
			}
		}()

		var timer = time.NewTicker(RepublishTxInterval)
		defer timer.Stop()

		for {
			select {
			case <-timer.C:
				pool.txsCollect.RepublishTxs(RepublishTxLimit, func(wTx txpoolcore.TxWrapper) error {
					err := pool.txServant.PublishTx(pool.ctx, wTx.OriginTx())
					if err == nil {
						wTx.UpdateState(txpooli.TxState_Republished)
					}

					return err
				})
			case <-pool.ctx.Done():
				pool.log.Info("republishRoutine stopped")
				return
			}
		}
	}()
}
