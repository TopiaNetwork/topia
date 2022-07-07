package transactionpool

import (
	"container/heap"
	"encoding/json"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	tpcmm "github.com/TopiaNetwork/topia/common"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txuni "github.com/TopiaNetwork/topia/transaction/universal"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

type TxByGasAndTime []*txbasic.Transaction

func (s TxByGasAndTime) Len() int { return len(s) }
func (s TxByGasAndTime) Less(i, j int) bool {
	if GasPrice(s[i]) < GasPrice(s[j]) {
		return true
	} else if GasPrice(s[i]) > GasPrice(s[j]) {
		return false
	} else {
		return s[i].Head.TimeStamp < s[i].Head.TimeStamp
	}
}
func (s TxByGasAndTime) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type TxByNonce []*txbasic.Transaction

func (s TxByNonce) Len() int { return len(s) }
func (s TxByNonce) Less(i, j int) bool {
	return s[i].Head.Nonce < s[j].Head.Nonce
}
func (s TxByNonce) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type nonceHeap []uint64

func (h nonceHeap) Len() int           { return len(h) }
func (h nonceHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h nonceHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *nonceHeap) Push(x interface{}) {
	*h = append(*h, x.(uint64))
}

func (h *nonceHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

type mapNonceTxs struct {
	items map[uint64]*txbasic.Transaction
	index *nonceHeap
	cache []*txbasic.Transaction
	cnt   int64
}

func newMapNonceTxs() *mapNonceTxs {
	return &mapNonceTxs{
		items: make(map[uint64]*txbasic.Transaction),
		index: new(nonceHeap),
		cnt:   0,
	}
}
func (m *mapNonceTxs) Get(nonce uint64) *txbasic.Transaction {
	return m.items[nonce]
}

func (m *mapNonceTxs) Put(tx *txbasic.Transaction) {
	nonce := tx.Head.Nonce
	if m.items[nonce] == nil {
		heap.Push(m.index, nonce)
		atomic.AddInt64(&m.cnt, 1)
	}
	m.items[nonce], m.cache = tx, nil
}

func (m *mapNonceTxs) Remove(nonce uint64) bool {
	if _, ok := m.items[nonce]; !ok {
		return false
	}
	for i := 0; i < m.index.Len(); i++ {
		if (*m.index)[i] == nonce {

			atomic.AddInt64(&m.cnt, -1)
			heap.Remove(m.index, i)
			break
		}
	}
	delete(m.items, nonce)
	m.cache = nil

	return true
}

func (m *mapNonceTxs) dropLowNonceTx(thresholdNonce uint64) []*txbasic.Transaction {
	var removed []*txbasic.Transaction

	// Pop off heap items until the threshold is reached
	for m.index.Len() > 0 && (*m.index)[0] < thresholdNonce {
		nonce := heap.Pop(m.index).(uint64)
		removed = append(removed, m.items[nonce])
		atomic.AddInt64(&m.cnt, -1)
		delete(m.items, nonce)
	}
	// If we had a cached order, shift the front
	if m.cache != nil {
		m.cache = m.cache[len(removed):]
	}
	return removed
}

func (m *mapNonceTxs) Len() int {
	return len(m.items)
}

func (m *mapNonceTxs) flatten() []*txbasic.Transaction {
	if m.cache == nil {
		m.cache = make([]*txbasic.Transaction, 0, len(m.items))
		for _, tx := range m.items {
			m.cache = append(m.cache, tx)
		}
		sort.Sort(TxByNonce(m.cache))
	}
	return m.cache
}

type pendingTxList struct {
	mu      sync.RWMutex
	addrTxs map[tpcrtypes.Address]*mapNonceTxs
}

func newPending() *pendingTxList {

	pd := &pendingTxList{
		addrTxs: make(map[tpcrtypes.Address]*mapNonceTxs, 0),
	}
	return pd
}
func (pd *pendingTxList) addTxsToPending(txs []*txbasic.Transaction, isPackaged func(id txbasic.TxID) bool, isLocal bool) (dropOldTxID []txbasic.TxID, dropOldSize int64, dropNewTxs []*txbasic.Transaction) {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	var dropTxs []*txbasic.Transaction
	var dropTxIDs []txbasic.TxID
	var dropSize int64
	for _, tx := range txs {
		txID, _ := tx.TxID()
		address := tpcrtypes.Address(tx.Head.FromAddr)
		list, ok := pd.addrTxs[address]
		if !ok {
			list = newMapNonceTxs()
		}

		if txOld := list.Get(tx.Head.Nonce); txOld != nil {
			if isLocal {
				if GasPrice(txOld) < GasPrice(tx) {
					oldTxID, _ := txOld.TxID()
					if isPackaged(oldTxID) {
						dropTxs = append(dropTxs, tx)
						continue
					}
					list.Remove(tx.Head.Nonce)
					list.Put(tx)
					pd.addrTxs[address] = list
					dropTxIDs = append(dropTxIDs, txID)
					dropSize += int64(txOld.Size())
				} else {
					dropTxs = append(dropTxs, tx)
					continue
				}
			} else {
				dropTxs = append(dropTxs, tx)
				continue
			}
		}
		list.Put(tx)
		pd.addrTxs[address] = list
	}
	return dropTxIDs, dropSize, dropTxs
}

func (pd *pendingTxList) addTxToPending(tx *txbasic.Transaction, isPackaged func(id txbasic.TxID) bool, isLocal bool) (replaced bool, oldTxID txbasic.TxID, oldSize int) {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	address := tpcrtypes.Address(tx.Head.FromAddr)
	list, ok := pd.addrTxs[address]
	if !ok {
		list = newMapNonceTxs()
	}

	if txOld := list.Get(tx.Head.Nonce); txOld != nil {
		if isLocal {
			if GasPrice(txOld) < GasPrice(tx) {
				oldTxID, _ := txOld.TxID()
				if isPackaged(oldTxID) {
					return false, "", 0
				}
				list.Remove(tx.Head.Nonce)
				list.Put(tx)
				pd.addrTxs[address] = list
				return true, oldTxID, txOld.Size()
			} else {
				return false, "", 0
			}
		} else {
			return false, "", 0
		}

	}
	list.Put(tx)
	pd.addrTxs[address] = list
	return false, "", 0
}
func (pd *pendingTxList) getTxsByAddr(address tpcrtypes.Address) ([]*txbasic.Transaction, error) {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	list, ok := pd.addrTxs[address]
	if !ok {
		return nil, ErrAddrNotExist
	}
	return list.flatten(), nil
}
func (pd *pendingTxList) accountTxCnt(address tpcrtypes.Address) int64 {
	pd.mu.RLock()
	defer pd.mu.RUnlock()
	return atomic.LoadInt64(&pd.addrTxs[address].cnt)
}

func (pd *pendingTxList) isNonceContinuous(address tpcrtypes.Address) (bool, error) {
	pd.mu.RLock()
	defer pd.mu.RUnlock()
	list, ok := pd.addrTxs[address]
	if !ok {
		return false, ErrAddrNotExist
	}

	flatTxs := list.flatten()
	if flatTxs[0].Head.Nonce+uint64(len(flatTxs))-1 == flatTxs[len(flatTxs)-1].Head.Nonce {
		return true, nil
	}
	return false, ErrTxsNotContinuous
}
func (pd *pendingTxList) removeTxsForRevert(removeCnt int64, isLocal func(id txbasic.TxID) bool, removeWrappedTx func(id txbasic.TxID), removeCache func(id txbasic.TxID), remoteTxs func() []*txbasic.Transaction) (cnt int64, isEnough bool) {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	for addr, txList := range pd.addrTxs {
		if removeCnt > 0 {
			flatTxs := txList.flatten()
			if flatTxs[0].Head.Nonce+uint64(len(flatTxs))-1 != flatTxs[len(flatTxs)-1].Head.Nonce {
				for _, tx := range flatTxs {
					txID, _ := tx.TxID()
					if isLocal(txID) {
						goto forNextAddr
					}
				}
				for _, tx := range flatTxs {
					txID, _ := tx.TxID()
					removeWrappedTx(txID)
					removeCache(txID)
					removeCnt -= 1
				}
				delete(pd.addrTxs, addr)
			}
		forNextAddr:
		} else {
			return int64(0), true
		}
	}
	if removeCnt > 0 {
		remotes := remoteTxs()
		sort.Sort(TxByGasAndTime(remotes))
		for _, tx := range remotes {
			if removeCnt > 0 {
				txID, _ := tx.TxID()
				list := pd.addrTxs[tpcrtypes.Address(tx.Head.FromAddr)]
				list.Remove(tx.Head.Nonce)
				if list.Len() == 0 {
					delete(pd.addrTxs, tpcrtypes.Address(tx.Head.FromAddr))
				} else {
					pd.addrTxs[tpcrtypes.Address(tx.Head.FromAddr)] = list
				}
				removeWrappedTx(txID)
				removeCache(txID)
				removeCnt -= 1
			} else {
				return int64(0), true
			}
		}
	}
	return removeCnt, false
}

func (pd *pendingTxList) getAllCommitTxs(lastNonce func(address tpcrtypes.Address) (uint64, error), packagedTxs func(txs []*txbasic.Transaction)) (txs []*txbasic.Transaction, dropTxCnt, dropTxSize int) {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	dropTxCnt, dropTxSize = 0, 0
	for addr, list := range pd.addrTxs {
		nonce, _ := lastNonce(addr)
		dropped := list.dropLowNonceTx(nonce)
		if len(dropped) > 0 {
			for _, tx := range dropped {
				dropTxCnt += 1
				dropTxSize += tx.Size()
			}
		}
		flatTxs := list.flatten()
		if flatTxs[0].Head.Nonce+uint64(len(flatTxs))-1 == flatTxs[len(flatTxs)-1].Head.Nonce {
			txs = append(txs, flatTxs...)
			packagedTxs(txs)
		}
		pd.addrTxs[addr] = list
	}
	return txs, dropTxCnt, dropTxSize
}

func (pd *pendingTxList) removeTx(address tpcrtypes.Address, txID txbasic.TxID, nonce uint64, force bool) error {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	list, ok := pd.addrTxs[address]
	if !ok {
		return ErrAddrNotExist
	}
	if !force {
		flatTxs := list.flatten()
		if flatTxs[0].Head.Nonce+uint64(len(flatTxs))-1 != flatTxs[len(flatTxs)-1].Head.Nonce {
			return ErrTxsNotContinuous
		}
		if nonce != flatTxs[0].Head.Nonce {
			return ErrNonceNotMin
		}
		oldTxID, _ := flatTxs[0].TxID()
		if txID != oldTxID {
			return ErrTxIDDiff
		}
		list.Remove(nonce)
		if list.Len() == 0 {
			delete(pd.addrTxs, address)
		} else {
			pd.addrTxs[address] = list
		}
		return nil
	}
	oldTxID, _ := list.Get(nonce).TxID()
	if txID != oldTxID {
		return ErrTxIDDiff
	}
	list.Remove(nonce)
	if list.Len() == 0 {
		delete(pd.addrTxs, address)
	} else {
		pd.addrTxs[address] = list
	}
	return nil
}

func (pd *pendingTxList) removeTxs(address tpcrtypes.Address, list []*txbasic.Transaction, removeCache func(id txbasic.TxID), removeLookup func(id txbasic.TxID)) (dropCnt, dropSize int64) {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	nonceTxs := pd.addrTxs[address]
	for _, tx := range list {
		nonceTxs.Remove(tx.Head.Nonce)
		dropCnt += 1
		dropSize += int64(tx.Size())
		txID, _ := tx.TxID()
		removeCache(txID)
		removeLookup(txID)
	}
	if nonceTxs.Len() == 0 {
		delete(pd.addrTxs, address)
	} else {
		pd.addrTxs[address] = nonceTxs
	}
	return
}

type allLookupTxs struct {
	localTxs  *tpcmm.ShrinkableMap
	remoteTxs *tpcmm.ShrinkableMap
}

func newAllLookupTxs() *allLookupTxs {
	all := &allLookupTxs{
		localTxs:  tpcmm.NewShrinkMap(),
		remoteTxs: tpcmm.NewShrinkMap(),
	}
	return all
}

func (all *allLookupTxs) Get(id txbasic.TxID) (*wrappedTx, bool) {
	if value, ok := all.localTxs.Get(id); ok {
		return value.(*wrappedTx), true
	}
	if value, ok := all.remoteTxs.Get(id); ok {
		return value.(*wrappedTx), true
	}
	return nil, false
}
func (all *allLookupTxs) isRepublished(id txbasic.TxID) (bool, error) {
	if value, ok := all.localTxs.Get(id); ok {
		return value.(*wrappedTx).IsRepublished, nil
	}
	if value, ok := all.remoteTxs.Get(id); ok {
		return value.(*wrappedTx).IsRepublished, nil
	}
	return false, ErrTxNotExist
}
func (all *allLookupTxs) setPublishTag(id txbasic.TxID) {
	if value, ok := all.localTxs.Get(id); ok {
		value.(*wrappedTx).IsRepublished = true
		all.localTxs.CallBackSetNoLock(id, value)
	}
	if value, ok := all.remoteTxs.Get(id); ok {
		value.(*wrappedTx).IsRepublished = true
		all.localTxs.CallBackSetNoLock(id, value)
	}
}

func (all *allLookupTxs) Del(id txbasic.TxID) {
	all.localTxs.Del(id)
	all.remoteTxs.Del(id)
}

func (all *allLookupTxs) callBackDel(id txbasic.TxID, isLocal bool) {
	if isLocal {
		all.localTxs.CallBackDelNoLock(id)
	} else {
		all.remoteTxs.CallBackDelNoLock(id)
	}
}
func (all *allLookupTxs) Set(id txbasic.TxID, value *wrappedTx) {
	if value.IsLocal {
		all.localTxs.Set(id, value)
	} else {
		all.remoteTxs.Set(id, value)
	}
}

type wrappedTx struct {
	TxID          txbasic.TxID
	IsLocal       bool
	Category      txbasic.TransactionCategory
	LastTime      time.Time
	LastHeight    uint64
	TxState       txpooli.TransactionState
	IsRepublished bool
	FromAddr      tpcrtypes.Address
	Nonce         uint64
	Tx            *txbasic.Transaction
}

type wrappedTxData struct {
	TxID          txbasic.TxID
	IsLocal       bool
	LastTime      time.Time
	LastHeight    uint64
	TxState       txpooli.TransactionState
	IsRepublished bool
	Tx            []byte
}

type packagedTxs struct {
	mu  sync.RWMutex
	txs map[txbasic.TxID]*txbasic.Transaction
}

func newPackagedTxs() *packagedTxs {
	pt := &packagedTxs{
		txs: make(map[txbasic.TxID]*txbasic.Transaction, 0),
	}
	return pt
}
func (pt *packagedTxs) isPackaged(id txbasic.TxID) bool {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	_, ok := pt.txs[id]
	return ok
}
func (pt *packagedTxs) putTxs(list []*txbasic.Transaction) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	for _, tx := range list {
		txID, _ := tx.TxID()
		pt.txs[txID] = tx
	}
}
func (pt *packagedTxs) purge() {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	pt.txs = make(map[txbasic.TxID]*txbasic.Transaction, 0)
}

func GasPrice(tx *txbasic.Transaction) uint64 {
	gasPrice := explainTx(tx, explainGasPrice)
	if gasPrice == nil {
		return 0
	}
	return gasPrice.(uint64)
}

func GasLimit(tx *txbasic.Transaction) uint64 {
	gasLimit := explainTx(tx, explainGasLimit)
	if gasLimit == nil {
		return 0
	}
	return gasLimit.(uint64)
}

type explainType int

const (
	explainToAddress explainType = iota
	explainGasPrice
	explainGasLimit
)

func explainTx(tx *txbasic.Transaction, eType explainType) interface{} {

	switch txbasic.TransactionCategory(tx.Head.Category) {
	case txbasic.TransactionCategory_Topia_Universal:
		var txUniversal txuni.TransactionUniversal
		_ = json.Unmarshal(tx.Data.Specification, &txUniversal)
		switch eType {
		case explainToAddress:
			if txUniversal.Head.Type == uint32(txuni.TransactionUniversalType_Transfer) {
				var targetData txuni.TransactionUniversalTransfer
				_ = json.Unmarshal(txUniversal.Data.Specification, &targetData)
				return targetData.TargetAddr
			} else {
				return ""
			}
		case explainGasPrice:
			return txUniversal.Head.GasPrice
		case explainGasLimit:
			return txUniversal.Head.GasLimit
		}
	}
	return nil
}
