package core

import (
	"context"
	"fmt"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"sync"
	"time"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type CollectTxs struct {
	syncOfAccount sync.RWMutex
	fullTxs       *tpcmm.ShrinkableMap
	accountTxs    map[tpcrtypes.Address]*accountTxManager
	pendingTxChan chan TxWrapper
	pendingTxs    *pendingTxList
}

func NewCollectTxs() *CollectTxs {
	pendingTxChan := make(chan TxWrapper, 100)
	return &CollectTxs{
		fullTxs:       tpcmm.NewShrinkMap(),
		accountTxs:    make(map[tpcrtypes.Address]*accountTxManager),
		pendingTxChan: pendingTxChan,
		pendingTxs:    newPendingTxList(pendingTxChan),
	}
}

func (c *CollectTxs) Start(ctx context.Context) {
	c.pendingTxs.startReceivePendingTx(ctx)
}

func (c *CollectTxs) AddTx(wTx TxWrapper) error {
	txID := wTx.TxID()
	wTxE, ok := c.fullTxs.Get(txID)
	if ok && wTxE != nil {
		return fmt.Errorf("Existed tx %s", txID)
	}
	c.fullTxs.Set(txID, wTx)

	fromAddr := wTx.FromAddr()

	c.syncOfAccount.Lock()
	accTxMng, ok := c.accountTxs[fromAddr]
	if ok {
		c.syncOfAccount.Unlock()

		return accTxMng.AddTx(wTx)
	} else {
		accTxMngNew := newAccountTxManager(c.pendingTxChan)
		c.accountTxs[fromAddr] = accTxMngNew
		c.syncOfAccount.Unlock()

		return accTxMngNew.AddTx(wTx)
	}
}

func (c *CollectTxs) RemoveTx(txID txbasic.TxID) TxWrapper {
	c.fullTxs.Del(txID)
	wTx := c.pendingTxs.removeTx(txID)
	if wTx != nil {
		fromAddr := wTx.FromAddr()
		c.syncOfAccount.Lock()
		accTxMng, ok := c.accountTxs[fromAddr]
		c.syncOfAccount.Unlock()
		if ok {
			accTxMng.removePendingTx(wTx.Nonce())
		}

		return wTx
	}

	return nil
}

func (c *CollectTxs) PickTxs(blockMaxBytes int64, blockMaxGas int64) []*txbasic.Transaction {
	return c.pendingTxs.pickTxs(blockMaxBytes, blockMaxGas)
}

func (c *CollectTxs) PurgeExpiredTxs(height uint64, timeOfTxLifecycle time.Duration, heightOfTxLifecycle uint64) {
	c.pendingTxs.purgeExpiredTxs(height, timeOfTxLifecycle, heightOfTxLifecycle)
}

func (c *CollectTxs) RepublishTxs(rePubLimit int, publish func(wTx TxWrapper) error) {
	c.pendingTxs.republishTxs(rePubLimit, publish)
}

func (c *CollectTxs) PendingOfAddress(addr tpcrtypes.Address) ([]*txbasic.Transaction, error) {
	c.syncOfAccount.Lock()
	accTxMng, ok := c.accountTxs[addr]
	if ok {
		c.syncOfAccount.Unlock()
		return accTxMng.getPendingTxs(), nil
	} else {
		c.syncOfAccount.Unlock()
		return nil, fmt.Errorf("Can't get pending txs of address %s", addr)
	}
}

func (c *CollectTxs) AllOfAddress(addr tpcrtypes.Address) ([]*txbasic.Transaction, error) {
	c.syncOfAccount.Lock()
	accTxMng, ok := c.accountTxs[addr]
	if ok {
		c.syncOfAccount.Unlock()
		return accTxMng.getAllTxs(), nil
	} else {
		c.syncOfAccount.Unlock()
		return nil, fmt.Errorf("Can't get all txs of address %s", addr)
	}
}

func (c *CollectTxs) AllOfAddressFunc(addrFunc func(tpcrtypes.Address) bool) []*txbasic.Transaction {
	c.syncOfAccount.RLock()
	defer c.syncOfAccount.RUnlock()

	var rtnTxs []*txbasic.Transaction
	for accAddr, accTxMng := range c.accountTxs {
		if addrFunc(accAddr) {
			rtnTxs = append(rtnTxs, accTxMng.getAllTxs()...)
		}
	}

	return rtnTxs
}

func (c *CollectTxs) GetTx(txID txbasic.TxID) TxWrapper {
	wTxI, ok := c.fullTxs.Get(txID)
	if ok {
		return wTxI.(TxWrapper)
	}

	return nil
}
