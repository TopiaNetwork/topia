package transactionpool

import (
	"runtime/debug"
	"sync/atomic"
	"time"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

func (pool *transactionPool) loopChanSelect() {

	pool.wg.Add(1)
	go pool.loopAddTxs()

	pool.wg.Add(1)
	go pool.loopAddIntoSorted()

	pool.wg.Add(1)
	go pool.loopChanSaveTxsStorage()
	pool.wg.Add(1)
	go pool.loopChanDelTxsStorage()

	pool.wg.Add(1)
	go pool.LoopAddTxsForBlocksRevert()
	pool.wg.Add(1)
	go pool.loopDropTxsIfBlockAdded()

	pool.wg.Add(1)
	go pool.loopRemoveTxForUptoLifeTime()
	pool.wg.Add(1)
	go pool.loopRegularSaveLocalTxs()
	pool.wg.Add(1)
	go pool.loopRegularRepublish()
	pool.wg.Add(1)
	go pool.loopSaveAllIfShutDown()

}

func (pool *transactionPool) loopAddTxs() {
	defer pool.wg.Done()
	for {
		if atomic.LoadInt32(&pool.isInRemove) == int32(0) && atomic.LoadInt32(&pool.isPicking) == int32(0) {
			select {
			case txsItem := <-pool.chanAddTxs:
				pool.loopAddTxsFromChan(txsItem)
				//if err != nil {
				//	pool.log.Errorf("loopAddTxs error:", err)
				//}
			case <-pool.ctx.Done():
				pool.log.Info("loopAddTxs stopped")
				return
			}
		}

	}

}
func (pool *transactionPool) loopAddIntoSorted() {
	defer pool.wg.Done()
	for {
		if atomic.LoadInt32(&pool.isPicking) == int32(0) && atomic.LoadInt32(&pool.isInRemove) == int32(0) {
			select {
			case sortItem := <-pool.chanSortedItem:
				pool.sortedTxs.addAccTx(sortItem.account, sortItem.maxPrice, sortItem.isMaxPriceChanged, sortItem.txs)
			case <-pool.ctx.Done():
				pool.log.Info("loopAddIntoSorted stopped")
			}
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
	var saveTxToStorageTimer = time.NewTicker(SaveTxStorageInterval) //0.15s
	defer saveTxToStorageTimer.Stop()
	for {
		select {
		case wTs := <-pool.chanSaveTxsStorage:
			wrappedTxs = append(wrappedTxs, wTs...)

		case <-saveTxToStorageTimer.C:
			if len(wrappedTxs) > 0 {
				pool.SaveLocalTxsData(pool.config.PathTxsStorage, wrappedTxs)
				saveTxToStorageTimer.Reset(SaveTxStorageInterval)
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
				delTxFromStorageTimer.Reset(delTxFromStorageInterval)
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

	defer func(t0 time.Time) {
		pool.log.Infof(pool.nodeId, "DropTxsIfBlockAdded cost time:", time.Since(t0))
	}(time.Now())

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
		case blocks := <-pool.chanBlocksRevert:
			if len(blocks) > 0 {
				if blocks[0] != nil {
					pool.addTxsForBlocksRevert(blocks)
				}
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

	var expiredRemove = time.NewTicker(RemoveTxInterval) //30s report eviction
	defer expiredRemove.Stop()

	for {
		select {
		// Handle inactive account transaction eviction
		case <-expiredRemove.C:

			pool.removeTxForExpired(TxExpiredTime)

		case <-pool.ctx.Done():
			pool.log.Info("loopRemoveTxForUptoLifeTime stopped")
			return
		}
	}
}

func (pool *transactionPool) removeTxForExpired(expiredPolicy TxExpiredPolicy) {

	var localDropTxIDs []txbasic.TxID

	dropLocalsForExpired := func(k interface{}, v interface{}) {
		switch expiredPolicy {
		case TxExpiredTime:
			if time.Since(v.(*wrappedTx).LastTime) > pool.config.LifetimeForTx {
				localDropTxIDs = append(localDropTxIDs, k.(txbasic.TxID))
			}
		case TxExpiredHeight:
			if v.(*wrappedTx).LastHeight > pool.config.LifeHeight {
				localDropTxIDs = append(localDropTxIDs, k.(txbasic.TxID))
			}
		case TxExpiredTimeAndHeight:
			if time.Since(v.(*wrappedTx).LastTime) > pool.config.LifetimeForTx && v.(*wrappedTx).LastHeight > pool.config.LifeHeight {
				localDropTxIDs = append(localDropTxIDs, k.(txbasic.TxID))
			}
		case TxExpiredTimeOrHeight:
			if time.Since(v.(*wrappedTx).LastTime) > pool.config.LifetimeForTx || v.(*wrappedTx).LastHeight > pool.config.LifeHeight {
				localDropTxIDs = append(localDropTxIDs, k.(txbasic.TxID))
			}
		}
	}
	pool.allWrappedTxs.localTxs.IterateCallback(dropLocalsForExpired)

	if len(localDropTxIDs) > 0 {
		pool.RemoveTxHashes(localDropTxIDs)
	}

	var remoteDropTxIDs []txbasic.TxID
	dropRemotesForExpired := func(k interface{}, v interface{}) {
		switch expiredPolicy {
		case TxExpiredTime:
			if time.Since(v.(*wrappedTx).LastTime) > pool.config.LifetimeForTx {
				remoteDropTxIDs = append(remoteDropTxIDs, k.(txbasic.TxID))
			}
		case TxExpiredHeight:
			if v.(*wrappedTx).LastHeight > pool.config.LifeHeight {
				remoteDropTxIDs = append(remoteDropTxIDs, k.(txbasic.TxID))
			}
		case TxExpiredTimeAndHeight:
			if time.Since(v.(*wrappedTx).LastTime) > pool.config.LifetimeForTx && v.(*wrappedTx).LastHeight > pool.config.LifeHeight {
				remoteDropTxIDs = append(remoteDropTxIDs, k.(txbasic.TxID))
			}
		case TxExpiredTimeOrHeight:
			if time.Since(v.(*wrappedTx).LastTime) > pool.config.LifetimeForTx || v.(*wrappedTx).LastHeight > pool.config.LifeHeight {
				remoteDropTxIDs = append(remoteDropTxIDs, k.(txbasic.TxID))
			}
		}
	}
	pool.allWrappedTxs.remoteTxs.IterateCallback(dropRemotesForExpired)

	if len(remoteDropTxIDs) > 0 {
		pool.RemoveTxHashes(remoteDropTxIDs)
	}
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
			if atomic.LoadInt32(&pool.isInRemove) == 0 && atomic.LoadInt32(&pool.isPicking) == 0 {
				pool.republishTxForExpired(TxRepublishTime)
			}
			republish.Reset(RepublishTxInterval)
		case <-pool.ctx.Done():
			pool.log.Info("loopRegularRepublish stopped")
			return
		}
	}
}

func (pool *transactionPool) loopAddTxsFromChan(txsItem *addTxsItem) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	if pool.Count()+int64(len(txsItem.txs)) > pool.config.TxPoolMaxCnt {
		for _, tx := range txsItem.txs {
			id, _ := tx.TxID()
			pool.txCache.Add(id, txpooli.StateDroppedForTxPoolFull)
		}
		return ErrTxPoolFull
	}
	curHeight, err := pool.txServant.CurrentHeight()
	if err != nil {
		return err
	}

	var newTxs []*txbasic.Transaction
	for _, tx := range txsItem.txs {
		chainNonce, _ := pool.txServant.GetNonce(tpcrtypes.Address(tx.Head.FromAddr))
		if tx.Head.Nonce <= chainNonce {
			continue
		}
		txID, _ := tx.TxID()

		if _, ok := pool.allWrappedTxs.Get(txID); ok {
			continue
		}
		newTxs = append(newTxs, tx)
	}
	if len(newTxs) == 0 {
		return ErrNoTxAdded
	}
	getPendingNonce := func(addr tpcrtypes.Address) (uint64, error) {
		return pool.pendingNonces.get(addr)
	}
	setPendingNonce := func(addr tpcrtypes.Address, nonce uint64) {
		pool.pendingNonces.set(addr, nonce)
	}
	isPackaged := func(id txbasic.TxID) bool {
		_, ok := pool.packagedTxIDs.Get(id)
		return ok
	}
	dropOldTx := func(id txbasic.TxID) {
		pool.allWrappedTxs.Del(id)
		pool.txCache.Remove(id)
		pool.chanDelTxsStorage <- []txbasic.TxID{id}
	}
	addTxInfo := func(isLocal bool, tx *txbasic.Transaction) {
		txID, _ := tx.TxID()
		txInfo := &wrappedTx{
			TxID:          txID,
			IsLocal:       isLocal,
			Category:      txbasic.TransactionCategory(tx.Head.Category),
			LastTime:      time.Now(),
			LastHeight:    curHeight,
			TxState:       txpooli.StateTxAdded,
			IsRepublished: false,
			FromAddr:      tpcrtypes.Address(tx.Head.FromAddr),
			Nonce:         tx.Head.Nonce,
			Tx:            tx,
		}
		pool.allWrappedTxs.Set(txID, txInfo)
		pool.txCache.Add(txID, txpooli.StateTxAdded)
		pool.chanSaveTxsStorage <- []*wrappedTx{txInfo}
	}
	addSize := func(pendingCnt, pendingSize, poolCnt, poolSize int64) {
		if pendingCnt != int64(0) {
			atomic.AddInt64(&pool.pendingCount, pendingCnt)
		}
		if pendingSize != int64(0) {
			atomic.AddInt64(&pool.pendingSize, pendingSize)
		}
		if poolCnt != int64(0) {
			for {
				old := atomic.LoadInt64(&pool.poolCount)
				if atomic.CompareAndSwapInt64(&pool.poolCount, old, old+poolCnt) {
					break
				}
			}
		}
		if poolSize != int64(0) {
			atomic.AddInt64(&pool.poolSize, poolSize)
		}
	}
	addIntoSorted := func(address tpcrtypes.Address, maxPrice uint64, isMaxPriceChanged bool, txs []*txbasic.Transaction) bool {

		if atomic.LoadInt32(&pool.isPicking) == 1 {
			sortItem := &sortedItem{
				account:           address,
				maxPrice:          maxPrice,
				isMaxPriceChanged: isMaxPriceChanged,
				txs:               txs,
			}
			pool.chanSortedItem <- sortItem
			return false
		}

		pool.sortedTxs.addAccTx(address, maxPrice, isMaxPriceChanged, txs)
		return true
	}
	fetchTxsPrepared := func(address tpcrtypes.Address, nonce uint64) []*txbasic.Transaction {
		return pool.prepareTxs.fetchTxsToPending(address, nonce)
	}
	insertToPrepared := func(address tpcrtypes.Address, isLocal bool, tx *txbasic.Transaction,
		dropOldTx func(id txbasic.TxID), addTxInfo func(isLocal bool, tx *txbasic.Transaction), addSize func(pendingCnt, pendingSize, poolCnt, poolSize int64)) {
		pool.prepareTxs.addTxToPrepared(address, isLocal, tx, dropOldTx, addTxInfo, addSize)
	}

	pool.pending.addTxsToPending(newTxs, txsItem.isLocal, getPendingNonce, setPendingNonce, isPackaged, dropOldTx, addTxInfo,
		addSize, addIntoSorted, fetchTxsPrepared, insertToPrepared)

	return nil
}

func (pool *transactionPool) republishTxForExpired(policy TxRepublishPolicy) {

	var republishLocals []*wrappedTx
	republishExpiredTx := func(k interface{}, v interface{}) {
		switch policy {
		case TxRepublishTime:
			if time.Since(v.(*wrappedTx).LastTime) > pool.config.TxTTLTimeOfRepublish {
				if v.(*wrappedTx).IsRepublished == false {
					republishLocals = append(republishLocals, v.(*wrappedTx))
				}
			}
		case TxRepublishHeight:
			if v.(*wrappedTx).LastHeight > pool.config.TxTTLHeightOfRepublish {
				if v.(*wrappedTx).IsRepublished == false {
					republishLocals = append(republishLocals, v.(*wrappedTx))
				}
			}
		case TxRepublishTimeAndHeight:
			if time.Since(v.(*wrappedTx).LastTime) > pool.config.TxTTLTimeOfRepublish &&
				v.(*wrappedTx).LastHeight > pool.config.TxTTLHeightOfRepublish {
				if v.(*wrappedTx).IsRepublished == false {
					republishLocals = append(republishLocals, v.(*wrappedTx))
				}
			}
		case TxRepublishTimeOrHeight:
			if time.Since(v.(*wrappedTx).LastTime) > pool.config.TxTTLTimeOfRepublish ||
				v.(*wrappedTx).LastHeight > pool.config.TxTTLHeightOfRepublish {
				if v.(*wrappedTx).IsRepublished == false {
					republishLocals = append(republishLocals, v.(*wrappedTx))
				}
			}
		}
	}
	pool.allWrappedTxs.localTxs.IterateCallback(republishExpiredTx)
	if len(republishLocals) > 0 {
		for _, wTx := range republishLocals {
			pool.txServant.PublishTx(pool.ctx, wTx.Tx)
			pool.txCache.Add(wTx.TxID, txpooli.StateTxRepublished)
			wTx.IsRepublished = true
			pool.allWrappedTxs.localTxs.Set(wTx.TxID, wTx)
		}
	}

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
