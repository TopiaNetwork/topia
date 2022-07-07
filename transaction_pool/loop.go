package transactionpool

import (
	"runtime/debug"
	"time"

	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

func (pool *transactionPool) loopChanSelect() {
	pool.wg.Add(1)
	go pool.loopChanRemoveTxHashes()
	pool.wg.Add(1)
	go pool.loopChanDelTxsStorage()
	pool.wg.Add(1)
	go pool.loopChanSaveTxsStorage()
	pool.wg.Add(1)
	go pool.loopDropTxsIfBlockAdded()
	pool.wg.Add(1)
	go pool.LoopAddTxsForBlocksRevert()
	pool.wg.Add(1)
	go pool.loopRemoveTxForUptoLifeTime()
	pool.wg.Add(1)
	go pool.loopRegularSaveLocalTxs()
	pool.wg.Add(1)
	go pool.loopRegularRepublish()
	pool.wg.Add(1)
	go pool.loopSaveAllIfShutDown()
}

func (pool *transactionPool) loopChanRemoveTxHashes() {
	defer pool.wg.Done()
	defer func() {
		err := recover()
		if err != nil {
			pool.log.Errorf("loopChanRemoveTxHashes err:", err, debug.Stack())
		}
	}()

	for {
		select {
		case Hashes := <-pool.chanRmTxs:
			pool.RemoveTxHashes(Hashes, true)
		case <-pool.ctx.Done():
			pool.log.Info("loopChanRemoveTxHashes stopped")
			return
		}
	}
}

func (pool *transactionPool) loopChanSaveTxsStorage() {
	defer pool.wg.Done()
	defer func() {
		err := recover()
		if err != nil {
			pool.log.Errorf("loopChanSaveTxsStorage err:", err, debug.Stack())
		}
	}()
	var wrappedTxs []*wrappedTx
	var saveTxToStorageTimer = time.NewTicker(SaveTxStorageInterval) //0.05s
	defer saveTxToStorageTimer.Stop()
	for {
		select {
		case wTs := <-pool.chanSaveTxsStorage:
			wrappedTxs = append(wrappedTxs, wTs...)

		case <-saveTxToStorageTimer.C:
			if len(wrappedTxs) > 0 {
				pool.SaveLocalTxsData(pool.config.PathTxsStorage, wrappedTxs)
			}

		case <-pool.ctx.Done():
			pool.log.Info("loopChanSaveTxsStorage stopped")
			return
		}
	}
}

func (pool *transactionPool) loopChanDelTxsStorage() {
	defer pool.wg.Done()
	defer func() {
		err := recover()
		if err != nil {
			pool.log.Errorf("loopChanDelTxsStorage err:", err, debug.Stack())
		}
	}()
	var delHashes []txbasic.TxID
	var delTxFromStorageTimer = time.NewTicker(delTxFromStorageInterval) //0.1s
	defer delTxFromStorageTimer.Stop()

	for {
		select {
		case Hashes := <-pool.chanDelTxsStorage:
			delHashes = append(delHashes, Hashes...)
		case <-delTxFromStorageTimer.C:
			if len(delHashes) > 0 {
				pool.DelLocalTxsData(pool.config.PathTxsStorage, delHashes)
			}
		case <-pool.ctx.Done():
			pool.log.Info("loopChanDelTxsStorage stopped")
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
			pool.saveAllWhenSysShutDown()

		case <-pool.ctx.Done():
			pool.log.Info("loopSaveAllIfShutDown stopped")
			return
		}
	}

}

func (pool *transactionPool) loopDropTxsIfBlockAdded() {
	defer pool.wg.Done()
	defer func() {
		err := recover()
		if err != nil {
			pool.log.Errorf("loopDropTxsIfBlockAdded err:", err, debug.Stack())
		}
	}()

	for {
		select {
		// Handle ChainHeadEvent
		case block := <-pool.chanBlockAdded:
			if block != nil {
				pool.dropTxsForBlockAdded(block)
			}
		case <-pool.ctx.Done():
			pool.log.Info("loopDropTxsIfBlockAdded stopped")
			return
		}
	}

}

func (pool *transactionPool) LoopAddTxsForBlocksRevert() {
	defer pool.wg.Done()
	defer func() {
		err := recover()
		if err != nil {
			pool.log.Errorf("LoopAddTxsForBlocksRevert err:", err, debug.Stack())
		}
	}()

	for {
		select {
		// Handle ChainHeadEvent
		case blocks := <-pool.chanBlocksRevert:
			if blocks != nil {
				pool.addTxsForBlocksRevert(blocks)
			}
		case <-pool.ctx.Done():
			pool.log.Info("LoopAddTxsForBlocksRevert stopped")
			return
		}
	}

}

func (pool *transactionPool) loopRemoveTxForUptoLifeTime() {
	defer pool.wg.Done()
	defer func() {
		err := recover()
		if err != nil {
			pool.log.Errorf("loopRemoveTxForUptoLifeTime err:", err, debug.Stack())
		}
	}()

	var evict = time.NewTicker(RemoveTxInterval) //30s report eviction
	defer evict.Stop()

	for {
		select {
		// Handle inactive account transaction eviction
		case <-evict.C:
			pool.removeTxForExpired(TxExpiredTime)

		case <-pool.ctx.Done():
			pool.log.Info("loopRemoveTxForUptoLifeTime stopped")
			return
		}
	}
}

func (pool *transactionPool) removeTxForExpired(expiredPolicy TxExpiredPolicy) {
	dropLocalsForExpired := func(k interface{}, v interface{}) {
		switch expiredPolicy {
		case TxExpiredTime:
			if time.Since(v.(*wrappedTx).LastTime) > pool.config.LifetimeForTx {
				pool.callBackRemoveTxByKey(v.(*wrappedTx).TxID, true, true)
			}
		case TxExpiredHeight:
			if v.(*wrappedTx).LastHeight > pool.config.LifeHeight {
				pool.callBackRemoveTxByKey(v.(*wrappedTx).TxID, true, true)
			}
		case TxExpiredTimeAndHeight:
			if time.Since(v.(*wrappedTx).LastTime) > pool.config.LifetimeForTx && v.(*wrappedTx).LastHeight > pool.config.LifeHeight {
				pool.callBackRemoveTxByKey(v.(*wrappedTx).TxID, true, true)
			}
		case TxExpiredTimeOrHeight:
			if time.Since(v.(*wrappedTx).LastTime) > pool.config.LifetimeForTx || v.(*wrappedTx).LastHeight > pool.config.LifeHeight {
				pool.callBackRemoveTxByKey(v.(*wrappedTx).TxID, true, true)
			}
		}
	}
	pool.allWrappedTxs.localTxs.IterateCallback(dropLocalsForExpired)

	dropRemotesForExpired := func(k interface{}, v interface{}) {
		switch expiredPolicy {
		case TxExpiredTime:
			if time.Since(v.(*wrappedTx).LastTime) > pool.config.LifetimeForTx {
				pool.callBackRemoveTxByKey(v.(*wrappedTx).TxID, true, false)
			}
		case TxExpiredHeight:
			if v.(*wrappedTx).LastHeight > pool.config.LifeHeight {
				pool.callBackRemoveTxByKey(v.(*wrappedTx).TxID, true, false)
			}
		case TxExpiredTimeAndHeight:
			if time.Since(v.(*wrappedTx).LastTime) > pool.config.LifetimeForTx && v.(*wrappedTx).LastHeight > pool.config.LifeHeight {
				pool.callBackRemoveTxByKey(v.(*wrappedTx).TxID, true, false)
			}
		case TxExpiredTimeOrHeight:
			if time.Since(v.(*wrappedTx).LastTime) > pool.config.LifetimeForTx || v.(*wrappedTx).LastHeight > pool.config.LifeHeight {
				pool.callBackRemoveTxByKey(v.(*wrappedTx).TxID, true, false)
			}
		}
	}
	pool.allWrappedTxs.remoteTxs.IterateCallback(dropRemotesForExpired)
}

func (pool *transactionPool) loopRegularSaveLocalTxs() {
	defer pool.wg.Done()
	defer func() {
		err := recover()
		if err != nil {
			pool.log.Errorf("loopRegularSaveLocalTxs err:", err, debug.Stack())
		}
	}()

	var stored = time.NewTicker(pool.config.ReStoredDur)
	defer stored.Stop()

	for {

		select {
		// Handle local transaction  store
		case <-stored.C:
			pool.SaveAllLocalTxsData(pool.config.PathTxsStorage)
		case <-pool.ctx.Done():
			pool.log.Info("loopRegularSaveLocalTxs stopped")
			return
		}

	}

}

func (pool *transactionPool) loopRegularRepublish() {
	defer pool.wg.Done()
	defer func() {
		err := recover()
		if err != nil {
			pool.log.Errorf("loopRegularRepublish err:", err, debug.Stack())
		}
	}()

	var republish = time.NewTicker(RepublishTxInterval) //30s check tx lifetime
	defer republish.Stop()

	for {
		select {
		case <-republish.C:
			pool.republishTxForExpired(TxRepublishTime)
		case <-pool.ctx.Done():
			pool.log.Info("loopRegularRepublish stopped")
			return
		}
	}
}

func (pool *transactionPool) republishTxForExpired(policy TxRepublishPolicy) {
	republishExpiredTx := func(k interface{}, v interface{}) {
		switch policy {
		case TxRepublishTime:
			if time.Since(v.(*wrappedTx).LastTime) > pool.config.TxTTLTimeOfRepublish {
				ok, err := pool.allWrappedTxs.isRepublished(v.(*wrappedTx).TxID)
				if err == nil && !ok {
					pool.txServant.PublishTx(pool.ctx, v.(*wrappedTx).Tx)
					pool.txCache.Add(v.(*wrappedTx).TxID, txpooli.StateTxRepublished)
					pool.allWrappedTxs.setPublishTag(v.(*wrappedTx).TxID)
				}
			}
		case TxRepublishHeight:
			if v.(*wrappedTx).LastHeight > pool.config.TxTTLHeightOfRepublish {
				ok, err := pool.allWrappedTxs.isRepublished(v.(*wrappedTx).TxID)
				if err == nil && !ok {
					pool.txServant.PublishTx(pool.ctx, v.(*wrappedTx).Tx)
					pool.txCache.Add(v.(*wrappedTx).TxID, txpooli.StateTxRepublished)
					pool.allWrappedTxs.setPublishTag(v.(*wrappedTx).TxID)
				}
			}
		case TxRepublishTimeAndHeight:
			if time.Since(v.(*wrappedTx).LastTime) > pool.config.TxTTLTimeOfRepublish &&
				v.(*wrappedTx).LastHeight > pool.config.TxTTLHeightOfRepublish {
				ok, err := pool.allWrappedTxs.isRepublished(v.(*wrappedTx).TxID)
				if err == nil && !ok {
					pool.txServant.PublishTx(pool.ctx, v.(*wrappedTx).Tx)
					pool.txCache.Add(v.(*wrappedTx).TxID, txpooli.StateTxRepublished)
					pool.allWrappedTxs.setPublishTag(v.(*wrappedTx).TxID)
				}
			}
		case TxRepublishTimeOrHeight:
			if time.Since(v.(*wrappedTx).LastTime) > pool.config.TxTTLTimeOfRepublish ||
				v.(*wrappedTx).LastHeight > pool.config.TxTTLHeightOfRepublish {
				ok, err := pool.allWrappedTxs.isRepublished(v.(*wrappedTx).TxID)
				if err == nil && !ok {
					pool.txServant.PublishTx(pool.ctx, v.(*wrappedTx).Tx)
					pool.txCache.Add(v.(*wrappedTx).TxID, txpooli.StateTxRepublished)
					pool.allWrappedTxs.setPublishTag(v.(*wrappedTx).TxID)
				}
			}
		}
	}
	pool.allWrappedTxs.localTxs.IterateCallback(republishExpiredTx)
}

func (pool *transactionPool) saveAllWhenSysShutDown() {
	if pool.config.PathTxsStorage != "" {
		if err := pool.SaveAllLocalTxsData(pool.config.PathTxsStorage); err != nil {
			pool.log.Errorf("Failed to save transactions info", "err", err)
		}
	} else {
		pool.log.Errorf("Failed to save transactions info: config.PathTxsStorage is nil")
	}
	if pool.config.PathConf != "" {
		if err := pool.savePoolConfig(pool.config.PathConf); err != nil {
			pool.log.Errorf("Failed to save txPool config", "err", err)
		}
	} else {
		pool.log.Errorf("Failed to save save txPool config: config.PathConf is nil")
	}

}
