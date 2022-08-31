package transactionpool

import (
	"container/heap"
	"encoding/json"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru"

	tpcmm "github.com/TopiaNetwork/topia/common"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/service"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txuni "github.com/TopiaNetwork/topia/transaction/universal"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

type addTxsItem struct {
	txs     []*txbasic.Transaction
	isLocal bool
}

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
	items       map[uint64]*txbasic.Transaction
	index       *nonceHeap
	cache       []*txbasic.Transaction
	cnt         int64
	maxPrice    uint64
	isMaxChange bool
	lock        sync.RWMutex
}

func newMapNonceTxs() *mapNonceTxs {
	return &mapNonceTxs{
		items: make(map[uint64]*txbasic.Transaction, 0),
		index: new(nonceHeap),
		lock:  sync.RWMutex{},
	}
}
func (m *mapNonceTxs) Get(nonce uint64) *txbasic.Transaction {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.items[nonce]
}
func (m *mapNonceTxs) isExist(nonce uint64) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()

	_, ok := m.items[nonce]
	return ok
}

func (m *mapNonceTxs) Put(tx *txbasic.Transaction) {
	m.lock.Lock()
	defer m.lock.Unlock()
	isNil := len(m.items) == 0
	nonce := tx.Head.Nonce
	if m.items[nonce] == nil {
		heap.Push(m.index, nonce)
		atomic.AddInt64(&m.cnt, 1)
		m.items[nonce], m.cache = tx, nil
	} else {
		m.items[nonce], m.cache = tx, nil
	}

	if GasPrice(tx) > m.maxPrice {
		m.maxPrice = GasPrice(tx)
		if !isNil {
			m.isMaxChange = true
		}
	}
}

func (m *mapNonceTxs) Remove(nonce uint64) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	if len(m.items) == 0 {
		return false
	}
	if _, ok := m.items[nonce]; !ok {
		return false
	}

	oldMaxPrice := GasPrice(m.items[nonce])

	if oldMaxPrice < m.maxPrice {
		for i := 0; i < m.index.Len(); i++ {
			if (*m.index)[i] == nonce {
				atomic.AddInt64(&m.cnt, -1)
				heap.Remove(m.index, i)
				break
			}
		}
	} else if oldMaxPrice == m.maxPrice {
		m.maxPrice = 0
		for i := 0; i < m.index.Len(); i++ {
			if (*m.index)[i] == nonce {
				if GasPrice(m.items[(*m.index)[i]]) > m.maxPrice {
					m.maxPrice = GasPrice(m.items[(*m.index)[i]])
				}
				atomic.AddInt64(&m.cnt, -1)
				heap.Remove(m.index, i)
			}

		}
		if m.maxPrice < oldMaxPrice {
			m.isMaxChange = true
		}
	}

	delete(m.items, nonce)
	m.cache = nil
	return true
}

func (m *mapNonceTxs) Filter(fil func(tx *txbasic.Transaction) bool) ([]*txbasic.Transaction, int64) {
	m.lock.Lock()
	defer m.lock.Unlock()

	removed, removedSize := m.filter(fil)
	if len(removed) > 0 {
		m.reHeap()
	}
	return removed, removedSize
}

func (m *mapNonceTxs) reHeap() {

	*m.index = make([]uint64, 0, len(m.items))
	for nonce := range m.items {
		*m.index = append(*m.index, nonce)
	}
	heap.Init(m.index)
	m.cache = nil
}

func (m *mapNonceTxs) filter(fil func(tx *txbasic.Transaction) bool) ([]*txbasic.Transaction, int64) {

	var removed []*txbasic.Transaction
	var filterSize int64
	oldMaxPrice := m.maxPrice
	m.maxPrice = 0
	for nonce, tx := range m.items {
		if fil(tx) {
			removed = append(removed, tx)
			filterSize += int64(tx.Size())
			atomic.AddInt64(&m.cnt, -1)
			delete(m.items, nonce)
		} else {
			if GasPrice(tx) > m.maxPrice {
				m.maxPrice = GasPrice(tx)
			}
		}
	}
	if oldMaxPrice != m.maxPrice {
		m.isMaxChange = true
	}
	if len(removed) > 0 {
		m.cache = nil
	}
	return removed, filterSize
}

func (m *mapNonceTxs) Len() int {
	m.lock.RLock()
	defer m.lock.RUnlock()

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

type accTxs struct {
	addrTxs *tpcmm.ShrinkableMap
}

func newAccTxs() *accTxs {
	accList := &accTxs{
		addrTxs: tpcmm.NewShrinkMap(),
	}
	return accList
}

func (atx *accTxs) addTxToPrepared(address tpcrtypes.Address, isLocal bool, tx *txbasic.Transaction,
	callBackdropOldTx func(txbasic.TxID),
	callBackAddTxInfo func(isLocal bool, tx *txbasic.Transaction),
	callBackAddSize func(pendingCnt, pendingSize, poolCnt, poolSize int64)) (added bool) {

	listInf, ok := atx.addrTxs.Get(address)
	var list *mapNonceTxs
	if !ok {
		list = newMapNonceTxs()
	} else {
		list = listInf.(*mapNonceTxs)
	}
	if isLocal {
		if txOld := list.Get(tx.Head.Nonce); txOld != nil {
			if GasPrice(txOld) < GasPrice(tx) {
				oldTxID, _ := txOld.TxID()
				callBackdropOldTx(oldTxID)
				dltSize := int64(tx.Size() - txOld.Size())
				callBackAddTxInfo(isLocal, tx)
				callBackAddSize(0, 0, 0, dltSize)

				list.Put(tx)
				atx.addrTxs.Set(address, list)
				return true
			} else {
				return false
			}
		} else {
			list.Put(tx)
			atx.addrTxs.Set(address, list)
			callBackAddTxInfo(true, tx)
			callBackAddSize(0, 0, 1, int64(tx.Size()))
			return true
		}
	} else {
		if txOld := list.Get(tx.Head.Nonce); txOld != nil {
			return false
		} else {
			list.Put(tx)
			atx.addrTxs.Set(address, list)
			callBackAddTxInfo(false, tx)
			callBackAddSize(0, 0, 1, int64(tx.Size()))
			return true
		}
	}
}

func (atx *accTxs) fetchTxsToPending(address tpcrtypes.Address, nonce uint64) []*txbasic.Transaction {

	listInf, ok := atx.addrTxs.Get(address)
	var list *mapNonceTxs
	if !ok {
		list = newMapNonceTxs()
	} else {
		list = listInf.(*mapNonceTxs)
	}
	var txs []*txbasic.Transaction
	for {
		if tx := list.Get(nonce + 1); tx != nil {
			txs = append(txs, tx)
			list.Remove(nonce + 1)
			nonce += 1
		} else {
			break
		}
	}

	if list.Len() == 0 {
		atx.addrTxs.Del(address)
	} else {
		atx.addrTxs.Set(address, list)
	}
	if len(txs) > 0 {
		return txs
	} else {
		return nil
	}
}

func (atx *accTxs) removeAddr(addr tpcrtypes.Address, txs []*txbasic.Transaction, callBackdropTx func(id txbasic.TxID)) (rmCnt, rmSize int64) {
	listInf, ok := atx.addrTxs.Get(addr)
	if !ok {
		return int64(0), int64(0)
	}
	list := listInf.(*mapNonceTxs)

	atx.addrTxs.Del(addr)

	for _, tx := range txs {
		id, _ := tx.TxID()
		ok := list.Remove(tx.Head.Nonce)
		if ok {
			rmCnt += int64(1)
			rmSize += int64(tx.Size())
			callBackdropTx(id)
		}
	}

	if len(list.items) == 0 {
		atx.addrTxs.Del(addr)
	} else {
		list.reHeap()
		atx.addrTxs.Set(addr, list)
	}

	return rmCnt, rmSize
}

func (atx *accTxs) addTxsToPending(txs []*txbasic.Transaction, isLocal bool,
	getPendingNonce func(addr tpcrtypes.Address) (uint64, error), setPendingNonce func(tpcrtypes.Address, uint64),
	isPackaged func(id txbasic.TxID) bool,
	callBackdropOldTx func(txbasic.TxID),
	callBackAddTxInfo func(isLocal bool, tx *txbasic.Transaction),
	callBackAddSize func(pendingCnt, pendingSize, poolCnt, poolSize int64),
	addIntoSorted func(addr tpcrtypes.Address, masPrice uint64, isMaxPriceChanged bool, txs []*txbasic.Transaction) bool,
	fetchTxsPrepared func(address tpcrtypes.Address, nonce uint64) []*txbasic.Transaction,
	insertToPrepare func(address tpcrtypes.Address, isLocal bool, tx *txbasic.Transaction, dropOldTx func(id txbasic.TxID), addTxInfo func(isLocal bool, tx *txbasic.Transaction), addSize func(pendingCnt, pendingSize, poolCnt, poolSize int64))) {

	for _, tx := range txs {
		address := tpcrtypes.Address(tx.Head.FromAddr)
		pendingNonce, err := getPendingNonce(address)

		if err != nil {
			//the first tx for address
			listInf, ok := atx.addrTxs.Get(address)
			var list *mapNonceTxs
			if !ok {
				list = newMapNonceTxs()
			} else {
				list = listInf.(*mapNonceTxs)
			}
			list.Put(tx)

			setPendingNonce(address, tx.Head.Nonce)

			callBackAddTxInfo(isLocal, tx)

			callBackAddSize(int64(1), int64(tx.Size()), int64(1), int64(tx.Size()))

			txsPrepare := fetchTxsPrepared(address, tx.Head.Nonce)

			if len(txsPrepare) > 0 {
				for _, txPrepare := range txsPrepare {
					list.Put(txPrepare)
					setPendingNonce(address, txPrepare.Head.Nonce)
					callBackAddSize(int64(1), int64(txPrepare.Size()), int64(0), int64(0))
				}
			}
			list.reHeap()
			atx.addrTxs.Set(address, list)
			addIntoSorted(address, list.maxPrice, list.isMaxChange, list.flatten())
			continue
		}

		if tx.Head.Nonce == pendingNonce+1 {
			//insert the next nonce tx
			listInf, ok := atx.addrTxs.Get(address)
			var list *mapNonceTxs
			if !ok {
				list = newMapNonceTxs()
			} else {
				list = listInf.(*mapNonceTxs)
			}
			list.Put(tx)
			setPendingNonce(address, tx.Head.Nonce)
			callBackAddTxInfo(isLocal, tx)
			callBackAddSize(int64(1), int64(tx.Size()), int64(1), int64(tx.Size()))
			txsPrepare := fetchTxsPrepared(address, tx.Head.Nonce)
			if len(txsPrepare) > 0 {
				for _, txPrepared := range txsPrepare {
					list.Put(txPrepared)
					setPendingNonce(address, txPrepared.Head.Nonce)
					callBackAddSize(int64(1), int64(tx.Size()), int64(0), int64(0))
				}
			}
			list.reHeap()
			atx.addrTxs.Set(address, list)
			addIntoSorted(address, list.maxPrice, list.isMaxChange, list.flatten())

			continue
		}
		// replace local tx if gasPrice is higher
		if isLocal {
			if tx.Head.Nonce <= pendingNonce {
				listInf, ok := atx.addrTxs.Get(address)
				var list *mapNonceTxs
				if !ok {
					list = newMapNonceTxs()
				} else {
					list = listInf.(*mapNonceTxs)
				}
				if txOld := list.Get(tx.Head.Nonce); txOld != nil {
					if GasPrice(txOld) < GasPrice(tx) {
						oldTxID, _ := txOld.TxID()
						if isPackaged(oldTxID) {
							continue
						}
						list.Put(tx)
						list.reHeap()
						atx.addrTxs.Set(address, list)
						callBackdropOldTx(oldTxID)
						dltSize := int64(tx.Size() - txOld.Size())
						callBackAddTxInfo(isLocal, tx)
						callBackAddSize(int64(0), dltSize, int64(0), dltSize)
						addIntoSorted(address, list.maxPrice, list.isMaxChange, list.flatten())

						continue
					}
					continue
				}
			}
		} else {
			if tx.Head.Nonce <= pendingNonce {
				continue
			}
		}
		//nonce is too big insert to prepareTxs
		insertToPrepare(address, isLocal, tx, callBackdropOldTx, callBackAddTxInfo, callBackAddSize)
	}
}

func (atx *accTxs) reInjectTxsToPrepare(addr tpcrtypes.Address, txs []*txbasic.Transaction) {

	listInf, ok := atx.addrTxs.Get(addr)
	var list *mapNonceTxs
	if !ok {
		list = newMapNonceTxs()
	} else {
		list = listInf.(*mapNonceTxs)
	}
	for _, tx := range txs {
		list.Put(tx)
	}
	atx.addrTxs.Set(addr, list)
}

func (atx *accTxs) getTxsByAddr(address tpcrtypes.Address) ([]*txbasic.Transaction, error) {

	listInf, ok := atx.addrTxs.Get(address)
	var list *mapNonceTxs
	if !ok {
		list = newMapNonceTxs()
	} else {
		list = listInf.(*mapNonceTxs)
	}
	return list.flatten(), nil
}

func (atx *accTxs) accountTxCnt(address tpcrtypes.Address) int64 {
	listInf, ok := atx.addrTxs.Get(address)
	if !ok {
		return 0
	}
	return atomic.LoadInt64(&listInf.(*mapNonceTxs).cnt)
}

func (atx *accTxs) removeTxsForRevert(removeCnt int64, isLocal func(id txbasic.TxID) bool,
	rmTxInfo func(id txbasic.TxID), delFromSorted func(addr tpcrtypes.Address), addToSorted func(addr tpcrtypes.Address, maxPrice uint64, isMaxChange bool, txs []*txbasic.Transaction) bool,
	remoteTxs func() []*txbasic.Transaction) (cnt int64, isEnough bool) {
	var delAddrs []tpcrtypes.Address
	if removeCnt == 0 {
		return int64(0), true
	}
	callBackRemove := func(k interface{}, v interface{}) {
		addr, txList := k.(tpcrtypes.Address), v.(*mapNonceTxs)
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
					rmTxInfo(txID)

					removeCnt -= 1
				}
				delAddrs = append(delAddrs, addr)
			}
		forNextAddr:
		}
	}
	atx.addrTxs.IterateCallback(callBackRemove)
	if len(delAddrs) > 0 {
		for _, addr := range delAddrs {
			atx.addrTxs.Del(addr)
			delFromSorted(addr)
		}
	}

	if removeCnt > 0 {
		remotes := remoteTxs()
		sort.Sort(TxByGasAndTime(remotes))
		for _, tx := range remotes {
			if removeCnt > 0 {
				txID, _ := tx.TxID()
				listInf, ok := atx.addrTxs.Get(tpcrtypes.Address(tx.Head.FromAddr))
				if !ok {
					continue
				}
				list := listInf.(*mapNonceTxs)
				list.Remove(tx.Head.Nonce)
				if list.Len() == 0 {
					atx.addrTxs.Del(tpcrtypes.Address(tx.Head.FromAddr))
					delFromSorted(tpcrtypes.Address(tx.Head.FromAddr))
				} else {
					list.reHeap()
					atx.addrTxs.Set(tpcrtypes.Address(tx.Head.FromAddr), list)
					addToSorted(tpcrtypes.Address(tx.Head.FromAddr), list.maxPrice, list.isMaxChange, list.flatten())
				}
				rmTxInfo(txID)
				removeCnt -= 1
			} else {
				return int64(0), true
			}
		}
	}
	return removeCnt, false
}

func (atx *accTxs) prepareTxsRemoveTx(address tpcrtypes.Address, nonce uint64) error {

	listInf, ok := atx.addrTxs.Get(address)
	if !ok {
		return ErrAddrNotExist
	}
	list := listInf.(*mapNonceTxs)
	if !list.isExist(nonce) {
		return ErrTxNotExist
	}
	list.Remove(nonce)
	if list.Len() == 0 {
		atx.addrTxs.Del(address)
	} else {
		atx.addrTxs.Set(address, list)
	}
	return nil

}

func (atx *accTxs) pendingRemoveTx(address tpcrtypes.Address, nonce uint64, reInjectToPrepare func(tpcrtypes.Address, []*txbasic.Transaction, int64),
	setPendingNonce func(address tpcrtypes.Address, nonce uint64),
	delFromSorted func(addr tpcrtypes.Address),
	addIntoSorted func(addr tpcrtypes.Address, maxPrice uint64, isMaxPriceChanged bool, txs []*txbasic.Transaction) bool) error {

	listInf, ok := atx.addrTxs.Get(address)

	if !ok {
		return ErrAddrNotExist
	}
	list := listInf.(*mapNonceTxs)
	curTx := list.Get(nonce)

	if curTx == nil {
		return ErrTxNotExist
	}
	curTxSize := curTx.Size()

	ok = list.Remove(nonce)

	setPendingNonce(address, nonce-1)

	pickHighNonce := func(tx *txbasic.Transaction) bool {
		return tx.Head.Nonce > nonce
	}
	pickOutTxs, removedSize := list.Filter(pickHighNonce)
	if list.Len() == 0 {
		atx.addrTxs.Del(address)
		delFromSorted(address)
	} else {
		list.reHeap()
		atx.addrTxs.Set(address, list)
		addIntoSorted(address, list.maxPrice, list.isMaxChange, list.flatten())
	}

	if len(pickOutTxs) > 0 {

		reInjectToPrepare(address, pickOutTxs, removedSize+int64(curTxSize))
	} else {

		reInjectToPrepare("", nil, int64(curTxSize))

	}
	return nil
}

type sortedItem struct {
	account           tpcrtypes.Address
	maxPrice          uint64
	isMaxPriceChanged bool
	txs               []*txbasic.Transaction
}

type sortedTxItem struct {
	account  tpcrtypes.Address
	maxPrice uint64
	txs      []*txbasic.Transaction
}

type sortedByPolicy byte

const (
	sortedByMaxGasPrice sortedByPolicy = iota
	sortedByAvgGasPrice
	sortedByTime
)

type sortedTxList struct {
	sync   sync.RWMutex
	txList []*wrappedTx
	less   func(*wrappedTx, *wrappedTx) bool
}

func newSortedTxList(less func(*wrappedTx, *wrappedTx) bool) *sortedTxList {
	return &sortedTxList{
		less: less,
	}
}

func (st *sortedTxList) addTx(wTx *wrappedTx) {
	st.sync.Lock()
	defer st.sync.Unlock()

	index := sort.Search(len(st.txList), func(i int) bool {
		return st.less(st.txList[i], wTx)
	})

	if index == len(st.txList) {
		st.txList = append(st.txList, wTx)
		return
	}

	st.txList = append(st.txList[:index+1], st.txList[index:]...)
	st.txList[index] = wTx
}

func (st *sortedTxList) removeTx(txID txbasic.TxID) {
	st.sync.Lock()
	defer st.sync.Unlock()

	for i, wTx := range st.txList {
		if wTx.TxID == txID {
			st.txList = append(st.txList[:i], st.txList[i+1:]...)
			break
		}
	}
}

func (st *sortedTxList) reset() {
	st.sync.Lock()
	defer st.sync.Unlock()

	st.txList = make([]*wrappedTx, 0)
}

func (st *sortedTxList) size() int {
	st.sync.RLock()
	defer st.sync.RUnlock()

	return len(st.txList)
}

func (st *sortedTxList) PickTxs(blockMaxBytes int64, blockMaxGas int64) []*txbasic.Transaction {
	st.sync.RLock()
	defer st.sync.RUnlock()

	if len(st.txList) == 0 {
		return nil
	}

	//var totalSize int
	var rtnWTx []*txbasic.Transaction
	for _, wTx := range st.txList {
		//totalSize += wTx.Tx.Size()
		//if totalSize > int(blockMaxBytes) {
		//	break
		//} else {
		rtnWTx = append(rtnWTx, wTx.Tx)
		//}
	}

	return rtnWTx
}

type accountNonce struct {
	accNonce *lru.Cache
	state    service.StateQueryService
}

func newAccountNonce(state service.StateQueryService) *accountNonce {
	accN := &accountNonce{
		state: state,
	}
	accN.accNonce, _ = lru.New(AccountNonceCacheSize)
	return accN
}

func (accN *accountNonce) get(addr tpcrtypes.Address) (uint64, error) {
	if _, ok := accN.accNonce.Get(addr); !ok {
		nonce, err := accN.state.GetNonce(addr)
		if err != nil {
			return 0, err
		} else {
			accN.accNonce.Add(addr, nonce)
		}
	}
	n, _ := accN.accNonce.Get(addr)
	return n.(uint64), nil
}

func (accN *accountNonce) set(addr tpcrtypes.Address, nonce uint64) {
	accN.accNonce.Add(addr, nonce)
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

func (all *allLookupTxs) Del(id txbasic.TxID) {
	if all.localTxs.Size() > 0 {
		all.localTxs.Del(id)
	}
	if all.remoteTxs.Size() > 0 {
		all.remoteTxs.Del(id)
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
		_ = txUniversal.Unmarshal(tx.Data.Specification)
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
