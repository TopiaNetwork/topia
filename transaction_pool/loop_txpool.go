package transactionpool

import (
	"github.com/TopiaNetwork/topia/transaction/basic"
	"runtime/debug"
	"sync/atomic"
	"time"
)

func (pool *transactionPool) loop() {
	pool.wg.Add(1)
	go pool.chanRemoveTxHashs()
	pool.wg.Add(1)
	go pool.resetIfNewHead()
	pool.wg.Add(1)
	go pool.reportTicks()
	pool.wg.Add(1)
	go pool.removeTxForUptoLifeTime()
	pool.wg.Add(1)
	go pool.regularSaveLocalTxs()
	pool.wg.Add(1)
	go pool.regularRepublic()
}

func (pool *transactionPool) chanRemoveTxHashs() {
	//fmt.Println("chanRemoveTxHashs")
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
		case <-pool.chanSysShutDown:
			close(pool.chanReorgShutdown)
			//local txs save
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
			return
		}
	}
}

func (pool *transactionPool) saveAllIfShutDown() (a, b, c int) {
	defer pool.wg.Done()
	defer func() {
		if err := recover(); err != nil {
			pool.log.Errorf("saveAllIfShutDown err:", err, debug.Stack())
		}
	}()
	for {
		select {
		// System shutdown.  When the system is shut down, save to the files locals/remotes/configs
		case <-pool.chanSysShutDown:
			close(pool.chanReorgShutdown)
			//local txs save
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
			return
		}

	}
}
func (pool *transactionPool) resetIfNewHead() {
	defer pool.wg.Done()
	defer func() {
		if err := recover(); err != nil {
			pool.log.Errorf("resetIfNewHead err:", err, debug.Stack())
		}
	}()
	// Track the previous head headers for transaction reorgs
	var head = pool.query.CurrentBlock()
	for {
		select {
		// Handle ChainHeadEvent
		case ev := <-pool.chanChainHead:
			if ev.Block != nil {
				pool.requestReset(head.Head, ev.Block.Head)
				head = ev.Block
			}
		case <-pool.chanSysShutDown:
			close(pool.chanReorgShutdown)
			//local txs save
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
			return
		}
	}
}
func (pool *transactionPool) reportTicks() {
	defer pool.wg.Done()
	defer func() {
		if err := recover(); err != nil {
			pool.log.Errorf("reportTicks err:", err, debug.Stack())
		}
	}()
	var (
		prevPending, prevQueued, prevStales int
		report                              = time.NewTicker(pool.config.StatsReportInterval) //500ms report queue stats
	)
	defer report.Stop()
	for {
		select {
		// Handle stats reporting ticks
		case <-report.C:
			for category, _ := range pool.pendings.pending {
				pool.reportUniversal(category, prevPending, prevQueued, prevStales)
			}
		case <-pool.chanSysShutDown:
			close(pool.chanReorgShutdown)
			//local txs save
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
			return
		}
	}
}
func (pool *transactionPool) reportUniversal(category basic.TransactionCategory, prevPending, prevQueued, prevStales int) {

	pending, queued := pool.stats(category)
	stales := int(atomic.LoadInt64(&pool.sortedLists.getPricedlistByCategory(category).stales))
	if pending != prevPending || queued != prevQueued || stales != prevStales {
		pool.log.Debugf("Transaction pool status report", "category", category, "executable", pending, "queued", queued, "stales", stales)
		prevPending, prevQueued, prevStales = pending, queued, stales
	}
}

func (pool *transactionPool) removeTxForUptoLifeTime() {
	defer pool.wg.Done()
	defer func() {
		if err := recover(); err != nil {
			pool.log.Errorf("removeTxForUptoLifeTime err:", err, debug.Stack())
		}
	}()
	var evict = time.NewTicker(pool.config.EvictionInterval) //30s report eviction
	defer evict.Stop()

	select {
	// Handle inactive account transaction eviction
	case <-evict.C:

		for category, _ := range pool.pendings.getAll() {
			for addr, _ := range pool.queues.getTxsByCategory(category).getAll() {
				// Skip local transactions from the eviction mechanism
				if pool.locals.contains(addr) {
					continue
				}
				// Any non-locals old enough should be removed
				list := pool.queues.getTxsByCategory(category).getTxListByAddr(addr).Flatten()

				for _, tx := range list {
					txId, _ := tx.HashHex()
					if time.Since(pool.ActivationIntervals.activ[txId]) > pool.config.LifetimeForTx {
						pool.RemoveTxByKey(txId)
					}
				}
			}
		}
	case <-pool.chanSysShutDown:
		close(pool.chanReorgShutdown)
		//local txs save
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
		return
	}
}
func (pool *transactionPool) regularSaveLocalTxs() {
	defer pool.wg.Done()
	defer func() {
		if err := recover(); err != nil {
			pool.log.Errorf("regularSaveLocalTxs err:", err, debug.Stack())
		}
	}()
	var stored = time.NewTicker(pool.config.ReStoredDur)
	defer stored.Stop()
	select {
	// Handle local transaction  store
	case <-stored.C:
		for category, _ := range pool.allTxsForLook.getAll() {
			if err := pool.SaveLocalTxs(category); err != nil {
				pool.log.Warnf("Failed to save local tx ", "err", err)
			}
		}
	case <-pool.chanSysShutDown:
		close(pool.chanReorgShutdown)
		//local txs save
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
		return
	}
}

func (pool *transactionPool) regularRepublic() {
	//fmt.Println("regularRepublic")
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
			for category, _ := range pool.queues.getAll() {
				for addr := range pool.queues.getTxsByCategory(category).getAll() {
					// republic transactions from the republic mechanism
					list := pool.queues.getTxsByCategory(category).getTxListByAddr(addr).Flatten()
					for _, tx := range list {
						txId, _ := tx.HashHex()
						if time.Since(pool.ActivationIntervals.activ[txId]) > pool.config.DurationForTxRePublic {
							pool.BroadCastTx(tx)
						}
					}

				}
			}
		case <-pool.chanSysShutDown:
			close(pool.chanReorgShutdown)
			//local txs save
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
			return
		}
	}
}
