package core

import (
	"context"
	"fmt"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
	"sync"
	"time"

	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type pendingTxList struct {
	sync          sync.RWMutex
	pendingTxChan chan TxWrapper
	sortedTxs     *sortedTxList
	heightTxs     *sortedTxList
	timeTxs       *sortedTxList
}

type accountTxManager struct {
	sync          sync.RWMutex
	pendingTxs    map[uint64]TxWrapper
	gapTxs        map[uint64]TxWrapper //nonce -> TxWrapper
	maxContNonce  uint64               //max continuously nonce
	pendingTxChan chan TxWrapper
}

func newAccountTxManager(pendingTxChan chan TxWrapper) *accountTxManager {
	return &accountTxManager{
		pendingTxs:    make(map[uint64]TxWrapper),
		gapTxs:        make(map[uint64]TxWrapper),
		pendingTxChan: pendingTxChan,
	}
}

func newPendingTxList(pendingTxChan chan TxWrapper) *pendingTxList {
	return &pendingTxList{
		pendingTxChan: pendingTxChan,
		sortedTxs:     newSortedTxList(SortWay_Misc),
		heightTxs:     newSortedTxList(SortWay_Height),
		timeTxs:       newSortedTxList(SortWay_Time),
	}
}

func (pl *pendingTxList) addTx(wTx TxWrapper) {
	pl.sync.Lock()
	defer pl.sync.Unlock()

	pl.sortedTxs.addTx(wTx)
	pl.heightTxs.addTx(wTx)
	pl.timeTxs.addTx(wTx)
}

func (pl *pendingTxList) startReceivePendingTx(ctx context.Context) {
	go func() {
		for {
			select {
			case wTx := <-pl.pendingTxChan:
				pl.addTx(wTx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (pl *pendingTxList) removeTx(txID txbasic.TxID) TxWrapper {
	pl.sync.Lock()
	defer pl.sync.Unlock()

	wTx := pl.sortedTxs.removeTx(txID)
	pl.heightTxs.removeTx(txID)
	pl.timeTxs.removeTx(txID)

	return wTx
}

func (pl *pendingTxList) pickTxs(blockMaxBytes int64, blockMaxGas int64) []*txbasic.Transaction {
	return pl.sortedTxs.pickTxs(blockMaxBytes, blockMaxGas)
}

func (pl *pendingTxList) purgeExpiredTxs(height uint64, timeOfTxLifecycle time.Duration, heightOfTxLifecycle uint64) {
	pl.sync.Lock()
	defer pl.sync.Unlock()

	now := time.Now()

	for _, wTx := range pl.heightTxs.txList {
		if height == wTx.Height()+heightOfTxLifecycle {
			rTxID := wTx.TxID()
			pl.removeTx(rTxID)
		}
	}

	for _, wTx := range pl.timeTxs.txList {
		if now.Sub(wTx.TimeStamp()) > timeOfTxLifecycle {
			rTxID := wTx.TxID()
			pl.removeTx(rTxID)
		}
	}
}

func (pl *pendingTxList) republishTxs(rePubLimit int, publish func(wTx TxWrapper) error) {
	pl.sync.Lock()
	defer pl.sync.Unlock()

	rePubCnt := 0
	for _, wTx := range pl.heightTxs.txList {
		if rePubCnt < rePubLimit && wTx.TxState() != txpooli.TxState_Republished {
			publish(wTx)
			rePubCnt++
		}
	}

}

func (a *accountTxManager) GetMaxContNonce() uint64 {
	a.sync.RLock()
	defer a.sync.RUnlock()

	return a.maxContNonce
}

func (a *accountTxManager) AddTx(wTx TxWrapper) error {
	a.sync.Lock()
	defer a.sync.Unlock()

	nonce := wTx.Nonce()

	if _, ok := a.pendingTxs[nonce]; ok {
		return fmt.Errorf("Have existed pending tx: nonce %d,  txID %s", nonce, wTx.TxID())
	}

	if _, ok := a.gapTxs[nonce]; ok {
		return fmt.Errorf("Have existed gap tx: nonce %d,  txID %s", nonce, wTx.TxID())
	}

	if nonce == a.maxContNonce+1 {
		a.pendingTxChan <- wTx
		a.pendingTxs[nonce] = wTx
		a.maxContNonce = nonce

		i := a.maxContNonce + 1
		wTxNew, ok := a.gapTxs[i]
		for ok {
			a.pendingTxChan <- wTxNew
			a.pendingTxs[i] = wTxNew
			delete(a.gapTxs, i)
			a.maxContNonce = i
			i++
			wTxNew, ok = a.gapTxs[i]
		}

		if len(a.gapTxs) == 0 {
			a.gapTxs = make(map[uint64]TxWrapper)
		}

		return nil
	} else if nonce > a.maxContNonce+1 {
		a.gapTxs[nonce] = wTx
		return nil
	} else {
		return fmt.Errorf("Invalid tx nonce: nonce %d, expected nonce %d, txID %s", nonce, a.maxContNonce+1, wTx.TxID())
	}
}

func (a *accountTxManager) removePendingTx(nonce uint64) {
	a.sync.Lock()
	defer a.sync.Unlock()

	delete(a.pendingTxs, nonce)

	if len(a.pendingTxs) == 0 {
		a.pendingTxs = make(map[uint64]TxWrapper)
	}
}

func (a *accountTxManager) getPendingTxs() []*txbasic.Transaction {
	a.sync.Lock()
	defer a.sync.Unlock()

	var rtnTxs []*txbasic.Transaction
	nonce := a.maxContNonce
	wTx, ok := a.pendingTxs[nonce]
	for ok {
		rtnTxs = append(rtnTxs, wTx.OriginTx())
		nonce++
		wTx, ok = a.pendingTxs[nonce]
	}

	return rtnTxs
}

func (a *accountTxManager) getAllTxs() []*txbasic.Transaction {
	a.sync.Lock()
	defer a.sync.Unlock()

	var rtnTxs []*txbasic.Transaction
	nonce := a.maxContNonce
	wTx, ok := a.pendingTxs[nonce]
	for ok {
		rtnTxs = append(rtnTxs, wTx.OriginTx())
		nonce++
		wTx, ok = a.pendingTxs[nonce]
	}

	for _, wTx := range a.gapTxs {
		rtnTxs = append(rtnTxs, wTx.OriginTx())
	}

	return rtnTxs
}
