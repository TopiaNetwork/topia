package transactionpool

import (
	"runtime/debug"
	"time"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/transaction/basic"
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
		if err := recover(); err != nil {
			pool.log.Errorf("chanRemoveTxHashs err:", err, debug.Stack())
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
		if err := recover(); err != nil {
			pool.log.Errorf("saveAllIfShutDown err:", err, debug.Stack())
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
			pool.log.Info("Systemshutdown stopped")
			return
		}
	}

}

func (pool *transactionPool) loopResetIfBlockAdded() {
	defer pool.wg.Done()
	defer func() {
		if err := recover(); err != nil {
			pool.log.Errorf("loopResetIfBlockAdded err:", err, debug.Stack())
		}
	}()
	// Track the previous head headers for transaction reorgs
	var head = pool.query.CurrentBlock()

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
		if err := recover(); err != nil {
			pool.log.Errorf("removeTxForUptoLifeTime err:", err, debug.Stack())
		}
	}()

	var evict = time.NewTicker(pool.config.EvictionInterval) //30s report eviction
	defer evict.Stop()

	for {
		select {
		// Handle inactive account transaction eviction
		case <-evict.C:
			f0 := func(address tpcrtypes.Address) bool { return pool.locals.contains(address) }
			f1 := func(string2 string) time.Duration {
				return time.Since(pool.ActivationIntervals.getTxActivByKey(string2))
			}
			time2 := pool.config.LifetimeForTx
			f2 := func(string2 string) {
				pool.RemoveTxByKey(string2)
			}
			for category, _ := range pool.queues.getAll() {
				pool.queues.removeTxForLifeTime(category, f0, f1, time2, f2)

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
		if err := recover(); err != nil {
			pool.log.Errorf("regularSaveLocalTxs err:", err, debug.Stack())
		}
	}()

	var stored = time.NewTicker(pool.config.ReStoredDur)
	defer stored.Stop()

	for {

		select {
		// Handle local transaction  store
		case <-stored.C:
			for category, _ := range pool.allTxsForLook.getAll() {
				if err := pool.SaveLocalTxs(category); err != nil {
					pool.log.Warnf("Failed to save local tx ", "err", err)
				}
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
		if err := recover(); err != nil {
			pool.log.Errorf("regularRepublic err:", err, debug.Stack())
		}
	}()

	var republic = time.NewTicker(pool.config.RepublicInterval) //30s check tx lifetime
	defer republic.Stop()

	for {
		select {
		case <-republic.C:
			f1 := func(string2 string) time.Duration {
				return time.Since(pool.ActivationIntervals.getTxActivByKey(string2))
			}
			time2 := pool.config.DurationForTxRePublic
			f2 := func(tx *basic.Transaction) {
				pool.BroadCastTx(tx)
			}
			for category, _ := range pool.queues.getAll() {
				pool.queues.republicTx(category, f1, time2, f2)
			}
		case <-pool.ctx.Done():
			pool.log.Info("loopRegularRepublic stopped")
			return
		}
	}
}

func (pool *transactionPool) saveAllWhenSysShutDown() {
	for category, _ := range pool.allTxsForLook.getAll() {
		if err := pool.SaveLocalTxs(category); err != nil {
			pool.log.Warnf("Failed to save local transaction", "err", err)
		}
		//remote txs save
		if err := pool.SaveRemoteTxs(category); err != nil {
			pool.log.Warnf("Failed to save remote transaction", "err", err)
		}
	}
	//txPool configs save
	if err := pool.SaveConfig(); err != nil {
		pool.log.Warnf("Failed to save transaction pool configs", "err", err)
	}
}
