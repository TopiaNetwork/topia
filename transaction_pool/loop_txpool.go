package transactionpool

import (
	"runtime/debug"
	"time"

	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

func (pool *transactionPool) loopChanSelect() {
	pool.wg.Add(1)
	go pool.loopChanRemoveTxHashs()
	pool.wg.Add(1)
	go pool.loopResetIfBlockAdded()
	pool.wg.Add(1)
	go pool.loopRemoveTxForUptoLifeTime()
	pool.wg.Add(1)
	go pool.loopRegularSaveLocalTxs()
	pool.wg.Add(1)
	go pool.loopRegularRepublic()
	pool.wg.Add(1)
	go pool.loopSaveAllIfShutDown()
}

func (pool *transactionPool) loopChanRemoveTxHashs() {
	defer pool.wg.Done()
	defer func() {
		err := recover()
		if err != nil {
			pool.log.Errorf("loopChanRemoveTxHashs err:", err, debug.Stack())
		}
	}()

	for {
		select {
		case Hashs := <-pool.chanRmTxs:
			pool.RemoveTxHashs(Hashs)
		case <-pool.ctx.Done():
			pool.log.Info("loopChanRemoveTxHashs stopped")
			return
		}
	}

}

func (pool *transactionPool) loopSaveAllIfShutDown() {
	defer pool.wg.Done()
	defer func() {
		err := recover()
		if err != nil {
			pool.log.Errorf("loopSaveAllIfShutDown err:", err, debug.Stack())
		}
	}()

	for {
		select {
		// System shutdown.  When the system is shut down, save to the files locals/remotes/configs
		case <-pool.chanSysShutdown:
			close(pool.chanReorgShutdown)
			close(pool.chanRmTxs)
			close(pool.chanBlockAdded)
			pool.saveAllWhenSysShutDown()
			return
		case <-pool.ctx.Done():
			pool.log.Info("loopSaveAllIfShutDown stopped")
			return
		}
	}

}

func (pool *transactionPool) loopResetIfBlockAdded() {
	defer pool.wg.Done()
	defer func() {
		err := recover()
		if err != nil {
			pool.log.Errorf("loopResetIfBlockAdded err:", err, debug.Stack())
		}
	}()
	// Track the previous head headers for transaction reorgs
	head, err := pool.txServant.GetLatestBlock()
	if err != nil {
		pool.log.Errorf("loopResetIfBlockAdded get current block err:", err)
	}
	for {
		select {
		// Handle ChainHeadEvent
		case ev := <-pool.chanBlockAdded:
			if ev.Block != nil {
				pool.requestReset(head.Head, ev.Block.Head)
				head = ev.Block
			}
		case <-pool.ctx.Done():
			pool.log.Info("loopResetIfNewHead stopped")
			return
		}
	}

}

func (pool *transactionPool) loopRemoveTxForUptoLifeTime() {
	defer pool.wg.Done()
	defer func() {
		err := recover()
		if err != nil {
			pool.log.Errorf("removeTxForUptoLifeTime err:", err, debug.Stack())
		}
	}()

	var evict = time.NewTicker(pool.config.EvictionInterval) //30s report eviction
	defer evict.Stop()

	for {
		select {
		// Handle inactive account transaction eviction
		case <-evict.C:
			f1 := func(string2 txbasic.TxID) time.Duration {
				return time.Since(pool.ActivationIntervals.getTxActivByKey(string2))
			}
			time2 := pool.config.LifetimeForTx
			f2 := func(string2 txbasic.TxID) {
				pool.RemoveTxByKey(string2)
			}
			f3 := func(string2 txbasic.TxID) uint64 {
				curheight, err := pool.txServant.CurrentHeight()
				if err != nil {
					pool.log.Errorf("get current height error:", err)
				}
				txheight := pool.HeightIntervals.HI[string2]
				if curheight > txheight {
					return curheight - txheight
				}
				return 0
			}
			diffHeight := pool.config.LifeHeight
			for category := range pool.config.PathMapRemoteTxsByCategory {
				pool.queues.removeTxForLifeTime(category, pool.config.TxExpiredPolicy, f1, time2, f2, f3, diffHeight)

			}

		case <-pool.ctx.Done():
			pool.log.Info("loopRemoveTxForUptoLifeTime stopped")
			return
		}
	}
}

func (pool *transactionPool) loopRegularSaveLocalTxs() {
	defer pool.wg.Done()
	defer func() {
		err := recover()
		if err != nil {
			pool.log.Errorf("regularSaveLocalTxs err:", err, debug.Stack())
		}
	}()

	var stored = time.NewTicker(pool.config.ReStoredDur)
	defer stored.Stop()

	for {

		select {
		// Handle local transaction  store
		case <-stored.C:
			for category := range pool.config.PathMapRemoteTxsByCategory {
				err := pool.SaveRemoteTxs(category)
				if err != nil {
					pool.log.Warnf("Failed to save local tx ", "error:", err)
				}
			}
			err := pool.SaveTxsInfo()
			if err != nil {
				pool.log.Errorf("Failed to save Txs info ", "error:", err)
			}
		case <-pool.ctx.Done():
			pool.log.Info("loopRegularSaveLocalTxs stopped")
			return
		}

	}

}

func (pool *transactionPool) loopRegularRepublic() {
	defer pool.wg.Done()
	defer func() {
		err := recover()
		if err != nil {
			pool.log.Errorf("regularRepublic err:", err, debug.Stack())
		}
	}()

	var republic = time.NewTicker(pool.config.RepublicInterval) //30s check tx lifetime
	defer republic.Stop()

	for {
		select {
		case <-republic.C:
			f1 := func(string2 txbasic.TxID) time.Duration {
				return time.Since(pool.ActivationIntervals.getTxActivByKey(string2))
			}
			time2 := pool.config.TxTTLTimeOfRepublic
			f2 := func(tx *txbasic.Transaction) {
				pool.txServant.PublishTx(pool.ctx, tx)
			}
			f3 := func(string2 txbasic.TxID) uint64 {
				curHeight, err := pool.txServant.CurrentHeight()
				if err != nil {
					pool.log.Errorf("get current height error:", err)
				}
				txHeight := pool.HeightIntervals.getTxHeightByKey(string2)
				if curHeight > txHeight {
					return curHeight - txHeight
				}
				return 0
			}
			diffHeight := pool.config.TxTTLHeightOfRepublic
			for category := range pool.config.PathMapRemoteTxsByCategory {
				pool.queues.republicTx(category, TxRepublicTime, f1, time2, f2, f3, diffHeight)
			}
		case <-pool.ctx.Done():
			pool.log.Info("loopRegularRepublic stopped")
			return
		}
	}
}

func (pool *transactionPool) saveAllWhenSysShutDown() {
	if pool.config.PathMapRemoteTxsByCategory != nil {
		for category := range pool.config.PathMapRemoteTxsByCategory {
			err := pool.SaveRemoteTxs(category)
			if err != nil {
				pool.log.Errorf("Failed to save remote transaction", "err", err)
			}
		}
	}
	if pool.config.PathTxsInfoFile != "" {
		err := pool.SaveTxsInfo()
		if err != nil {
			pool.log.Errorf("Failed to save txs info", "err", err)
		}
	}
	//txPool configs save
	if pool.config.PathConfigFile != "" {
		err := pool.SaveConfig()
		if err != nil {
			pool.log.Errorf("Failed to save transaction pool configs", "err", err)
		}
	}

}
