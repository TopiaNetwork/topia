package transactionpool

import (
	"container/heap"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/transaction/basic"
)

const (
	txSegmentSize = 32 * 1024
	txMaxSize     = 4 * txSegmentSize
)

type nonceHp []uint64

func (h nonceHp) Len() int           { return len(h) }
func (h nonceHp) Less(i, j int) bool { return h[i] < h[j] }
func (h nonceHp) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *nonceHp) Push(x interface{}) {
	*h = append(*h, x.(uint64))
}

func (h *nonceHp) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// txSortedMapByNonce is a nonce->transaction hash map.
type txSortedMapByNonce struct {
	items map[uint64]*basic.Transaction
	index *nonceHp
	cache []*basic.Transaction
}

func newtxSortedMapByNonce() *txSortedMapByNonce {
	return &txSortedMapByNonce{
		items: make(map[uint64]*basic.Transaction),
		index: new(nonceHp),
	}
}

func (m *txSortedMapByNonce) Get(nonce uint64) *basic.Transaction {
	return m.items[nonce]
}

// Put inserts a new transaction into the map and updates the nonce index of the map.
//If a transaction already has the same Nonce, it will be overwritten
func (m *txSortedMapByNonce) Put(tx *basic.Transaction) {

	nonce := tx.Head.Nonce
	if m.items[nonce] == nil {
		heap.Push(m.index, nonce)
	}
	m.items[nonce], m.cache = tx, nil
}

// DropedTxForLessNonce drop all txs whose nonce is less than the threshold.
// return the droped txs
func (m *txSortedMapByNonce) dropedTxForLessNonce(threshold uint64) []*basic.Transaction {
	var removed []*basic.Transaction

	// Pop off heap items until the threshold is reached
	for m.index.Len() > 0 && (*m.index)[0] < threshold {
		nonce := heap.Pop(m.index).(uint64)
		removed = append(removed, m.items[nonce])
		delete(m.items, nonce)
	}
	// If we had a cached order, shift the front
	if m.cache != nil {
		m.cache = m.cache[len(removed):]
	}
	return removed
}

func (m *txSortedMapByNonce) Filter(filter func(*basic.Transaction) bool) []*basic.Transaction {
	removed := m.filter(filter)
	// If transactions were removed, the heap and cache are ruined
	if len(removed) > 0 {
		m.reheap()
	}
	return removed
}

func (m *txSortedMapByNonce) reheap() {
	*m.index = make([]uint64, 0, len(m.items))
	for nonce := range m.items {
		*m.index = append(*m.index, nonce)
	}
	heap.Init(m.index)
	m.cache = nil
}

func (m *txSortedMapByNonce) filter(filter func(*basic.Transaction) bool) []*basic.Transaction {
	var removed []*basic.Transaction

	// Collect all the transactions to filter out
	for nonce, tx := range m.items {
		if filter(tx) {
			removed = append(removed, tx)
			delete(m.items, nonce)
		}
	}
	if len(removed) > 0 {
		m.cache = nil
	}
	return removed
}

func (m *txSortedMapByNonce) CapItems(threshold int) []*basic.Transaction {
	// Short circuit if the number of items is under the limit
	if len(m.items) <= threshold {
		return nil
	}
	// Otherwise gather and drop the highest nonce'd transactions
	var drops []*basic.Transaction

	sort.Sort(*m.index)
	for size := len(m.items); size > threshold; size-- {
		drops = append(drops, m.items[(*m.index)[size-1]])
		delete(m.items, (*m.index)[size-1])
	}
	*m.index = (*m.index)[:threshold]
	heap.Init(m.index)

	// If we had a cache, shift the back
	if m.cache != nil {
		m.cache = m.cache[:len(m.cache)-len(drops)]
	}
	return drops
}

func (m *txSortedMapByNonce) Remove(nonce uint64) bool {
	_, ok := m.items[nonce]
	if !ok {
		return false
	}
	for i := 0; i < m.index.Len(); i++ {
		if (*m.index)[i] == nonce {
			heap.Remove(m.index, i)
			break
		}
	}
	delete(m.items, nonce)
	m.cache = nil

	return true
}

func (m *txSortedMapByNonce) Ready(start uint64) []*basic.Transaction {
	if m.index.Len() == 0 || (*m.index)[0] > start {
		return nil
	}
	var ready []*basic.Transaction
	for next := (*m.index)[0]; m.index.Len() > 0 && (*m.index)[0] == next; next++ {
		ready = append(ready, m.items[next])
		delete(m.items, next)
		heap.Pop(m.index)
	}
	m.cache = nil

	return ready
}

func (m *txSortedMapByNonce) Len() int {
	return len(m.items)
}

func (m *txSortedMapByNonce) flatten() []*basic.Transaction {
	// If the sorting was not cached yet, create and cache it
	if m.cache == nil {
		m.cache = make([]*basic.Transaction, 0, len(m.items))
		for _, tx := range m.items {
			m.cache = append(m.cache, tx)
		}
		sort.Sort(TxByNonce(m.cache))
	}
	return m.cache
}

// Flatten creates an unsorted transaction slice based on its loosely sorted internal representation.
// The sorted result is cached before any changes are made to the content, in case it is requested again.
func (m *txSortedMapByNonce) Flatten() []*basic.Transaction {
	cache := m.flatten()
	txs := make([]*basic.Transaction, len(cache))
	copy(txs, cache)
	return txs
}

// LastElement return transaction with the highest nonce
func (m *txSortedMapByNonce) LastElement() *basic.Transaction {
	cache := m.flatten()
	return cache[len(cache)-1]
}

type txCoreList struct {
	lock   sync.RWMutex
	strict bool                // nonces are strictly continuous or not
	txs    *txSortedMapByNonce // Hash map of the transactions
}

func newCoreList(strict bool) *txCoreList {
	return &txCoreList{
		strict: strict,
		txs:    newtxSortedMapByNonce(),
	}
}

// WhetherSameNonce returns Whether the specified transaction has the same Nonce as the transaction in the list.
func (l *txCoreList) WhetherSameNonce(tx *basic.Transaction) bool {
	l.lock.RLock()
	defer l.lock.RUnlock()
	Nonce := tx.Head.Nonce
	return l.txs.Get(Nonce) != nil
}

func (l *txCoreList) txCoreAdd(tx *basic.Transaction) (bool, *basic.Transaction) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	Nonce := tx.Head.Nonce
	old := l.txs.Get(Nonce)
	if old != nil {
		gaspriceOld := GasPrice(old)
		gaspriceTx := GasPrice(tx)
		if gaspriceOld >= gaspriceTx && old.Head.Nonce == tx.Head.Nonce {
			return false, nil
		}
	}
	l.txs.Put(tx)
	return true, old
}

// RemovedTxForLessNonce removes all transactions less than the current threshold from the list.
// For any post-deletion maintenance, each deleted transaction is returned.
func (l *txCoreList) RemovedTxForLessNonce(threshold uint64) []*basic.Transaction {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.txs.dropedTxForLessNonce(threshold)
}

// CapLimitTxs places a limit on the number of items, returning all transactions
// exceeding that limit.
func (l *txCoreList) CapLimitTxs(threshold int) []*basic.Transaction {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.txs.CapItems(threshold)
}

func (l *txCoreList) Remove(tx *basic.Transaction) (bool, []*basic.Transaction) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	Nonce := tx.Head.Nonce
	nonce := Nonce
	if removed := l.txs.Remove(nonce); !removed {
		return false, nil
	}

	// Filter out non-executable transactions
	if l.strict {
		return true, l.txs.Filter(func(tx *basic.Transaction) bool {
			Noncei := tx.Head.Nonce
			return Noncei > nonce
		})
	}
	return true, nil
}

func (l *txCoreList) Ready(start uint64) []*basic.Transaction {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.txs.Ready(start)
}

func (l *txCoreList) Len() int {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.txs.Len()
}

func (l *txCoreList) Empty() bool {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.Len() == 0
}

func (l *txCoreList) Flatten() []*basic.Transaction {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.txs.Flatten()
}

// LastElementWithHighestNonce returns tx with the highest nonce
func (l *txCoreList) LastElementWithHighestNonce() *basic.Transaction {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.txs.LastElement()
}

type pendingTxs struct {
	Mu         sync.RWMutex
	addrTxList map[tpcrtypes.Address]*txCoreList
}

func newPendingTxs() *pendingTxs {
	addtxs := &pendingTxs{
		addrTxList: make(map[tpcrtypes.Address]*txCoreList, 0),
	}
	return addtxs
}

type pendingsMap struct {
	Mu      sync.RWMutex
	pending map[basic.TransactionCategory]*pendingTxs
}

func newPendingsMap() *pendingsMap {
	pendings := &pendingsMap{
		pending: make(map[basic.TransactionCategory]*pendingTxs, 0),
	}
	pendings.pending[basic.TransactionCategory_Topia_Universal] = newPendingTxs()
	pendings.pending[basic.TransactionCategory_Eth] = newPendingTxs()
	return pendings
}

func (pendingmap *pendingsMap) getAllCommitTxs() []*basic.Transaction {
	pendingmap.Mu.RLock()
	defer pendingmap.Mu.RUnlock()
	var txs = make([]*basic.Transaction, 0)
	for _, pendingtxs := range pendingmap.pending {
		pendingtxs.Mu.RLock()
		pendingtxs.Mu.RUnlock()
		for _, txList := range pendingtxs.addrTxList {
			for _, tx := range txList.txs.cache {
				txs = append(txs, tx)
			}
		}
	}
	return txs
}

func (pendingmap *pendingsMap) getTxsByCategory(category basic.TransactionCategory) []*basic.Transaction {
	pendingmap.Mu.RLock()
	defer pendingmap.Mu.RUnlock()
	pending := make([]*basic.Transaction, 0)
	pendingcat := pendingmap.pending[category]
	if pendingcat == nil {
		pendingcat = newPendingTxs()
	}
	pendingcat.Mu.RLock()
	defer pendingcat.Mu.RUnlock()
	for _, list := range pendingcat.addrTxList {
		txs := list.Flatten()
		if len(txs) > 0 {
			pending = append(pending, txs...)
		}
	}
	return pending
}
func (pendingmap *pendingsMap) getPendingTxsByCategory(category basic.TransactionCategory) *pendingTxs {
	pendingmap.Mu.RLock()
	defer pendingmap.Mu.RUnlock()
	if pendingcat := pendingmap.pending[category]; pendingcat != nil {
		pendingcat.Mu.RLock()
		defer pendingcat.Mu.RUnlock()
		return pendingcat
	}
	return nil
}
func (pendingmap *pendingsMap) demoteUnexecutablesByCategory(category basic.TransactionCategory,
	f1 func(tpcrtypes.Address) uint64,
	f2 func(transactionCategory basic.TransactionCategory, txId string),
	f3 func(hash string, tx *basic.Transaction)) {
	pendingmap.Mu.Lock()
	defer pendingmap.Mu.Unlock()
	pendingcat := pendingmap.pending[category]
	if pendingcat == nil {
		pendingcat = newPendingTxs()
	}
	pendingcat.Mu.Lock()
	defer pendingcat.Mu.Unlock()
	for addr, list := range pendingcat.addrTxList {
		nonce := f1(addr)
		olds := list.RemovedTxForLessNonce(nonce)
		for _, tx := range olds {
			txId, _ := tx.HashHex()
			f2(category, txId)
		}
		if list.Len() > 0 && list.txs.Get(nonce) == nil {
			gaped := list.CapLimitTxs(0)
			for _, tx := range gaped {
				hash, _ := tx.HashHex()
				f3(hash, tx)
			}

		}
		if list.Empty() {
			delete(pendingcat.addrTxList, addr)
		}

	}
}

func (pendingmap *pendingsMap) getAddrTxsByCategory(category basic.TransactionCategory) map[tpcrtypes.Address][]*basic.Transaction {
	pendingmap.Mu.RLock()
	defer pendingmap.Mu.RUnlock()
	pending := make(map[tpcrtypes.Address][]*basic.Transaction)
	for addr, list := range pendingmap.pending[category].addrTxList {
		txs := list.Flatten()
		if len(txs) > 0 {
			pending[addr] = txs
		}
	}
	return pending
}
func (pendingmap *pendingsMap) getAddrTxListOfCategory(category basic.TransactionCategory) map[tpcrtypes.Address]*txCoreList {
	pendingmap.Mu.RLock()
	defer pendingmap.Mu.RUnlock()
	pendingcat := pendingmap.pending[category]
	if pendingcat == nil {
		pendingcat = newPendingTxs()
	}
	pendingcat.Mu.RLock()
	defer pendingcat.Mu.RUnlock()
	if maptxlist := pendingcat.addrTxList; maptxlist != nil {
		return maptxlist
	}

	return nil
}

func (pendingmap *pendingsMap) noncesForAddrTxListOfCategory(category basic.TransactionCategory) {
	pendingmap.Mu.Lock()
	defer pendingmap.Mu.Unlock()
	pendingcat := pendingmap.pending[category]
	if pendingcat == nil {
		pendingcat = newPendingTxs()
	}
	pendingcat.Mu.Lock()
	defer pendingcat.Mu.Unlock()
	if maptxlist := pendingmap.pending[category].addrTxList; maptxlist != nil {
		nonces := make(map[tpcrtypes.Address]uint64, len(maptxlist))
		for addr, list := range maptxlist {
			highestPending := list.LastElementWithHighestNonce()
			Noncei := highestPending.Head.Nonce
			nonces[addr] = Noncei + 1
		}
	}
}

func (pendingmap *pendingsMap) getCommitTxsCategory(category basic.TransactionCategory) []*basic.Transaction {
	pendingmap.Mu.RLock()
	defer pendingmap.Mu.RUnlock()
	pendingcat := pendingmap.pending[category]
	if pendingcat == nil {
		pendingcat = newPendingTxs()
	}
	pendingcat.Mu.RLock()
	defer pendingcat.Mu.RUnlock()
	var txs = make([]*basic.Transaction, 0)
	maptxlist := pendingcat.addrTxList
	if maptxlist == nil {
		return nil
	}
	for _, txlist := range maptxlist {
		for _, tx := range txlist.txs.cache {
			txs = append(txs, tx)
		}
	}

	return txs

}

func (pendingmap *pendingsMap) getStatsOfCategory(category basic.TransactionCategory) int {
	pendingmap.Mu.RLock()
	defer pendingmap.Mu.RUnlock()
	pendingcat := pendingmap.pending[category]
	if pendingcat == nil {
		pendingcat = newPendingTxs()
	}
	pendingcat.Mu.RLock()
	defer pendingcat.Mu.RUnlock()
	pending := 0
	if maptxlist := pendingcat.addrTxList; maptxlist != nil {
		for _, list := range maptxlist {
			pending += list.Len()
		}
		return pending

	}
	return 0
}
func (pendingmap *pendingsMap) truncatePendingByCategoryFun1(category basic.TransactionCategory) uint64 {
	pendingmap.Mu.Lock()
	defer pendingmap.Mu.Unlock()
	pendingcat := pendingmap.pending[category]
	if pendingcat == nil {
		pendingcat = newPendingTxs()
	}
	pendingcat.Mu.Lock()
	defer pendingcat.Mu.Unlock()
	pending := uint64(0)
	for _, list := range pendingcat.addrTxList {
		pending += uint64(list.Len())
	}
	return pending
}
func (pendingmap *pendingsMap) truncatePendingByCategoryFun2(category basic.TransactionCategory,
	f1 func(tpcrtypes.Address) bool, PendingAccountSegments uint64) map[tpcrtypes.Address]int {
	pendingmap.Mu.Lock()
	defer pendingmap.Mu.Unlock()
	pendingcat := pendingmap.pending[category]
	if pendingcat == nil {
		pendingcat = newPendingTxs()
	}
	var greyAccounts map[tpcrtypes.Address]int
	for addr, list := range pendingmap.pending[category].addrTxList {
		// Only evict transactions from high rollers
		if f1(addr) && uint64(list.Len()) > PendingAccountSegments {
			greyAccounts[addr] = list.Len()
		}
	}
	return greyAccounts
}
func (pendingmap *pendingsMap) truncatePendingByCategoryFun3(f31 func(category basic.TransactionCategory, txId string),
	category basic.TransactionCategory, addr tpcrtypes.Address) int {
	pendingmap.Mu.Lock()
	defer pendingmap.Mu.Unlock()
	pendingcat := pendingmap.pending[category]
	if pendingcat == nil {
		pendingcat = newPendingTxs()
	}
	pendingcat.Mu.Lock()
	defer pendingcat.Mu.Unlock()
	if list := pendingcat.addrTxList[addr]; list != nil {
		caps := list.CapLimitTxs(list.Len() - 1)
		for _, tx := range caps {
			txId, _ := tx.HashHex()
			f31(category, txId)
		}
		return len(caps)
	}
	return 0
}

func (pendingmap *pendingsMap) getTxListByAddrOfCategory(category basic.TransactionCategory, addr tpcrtypes.Address) *txCoreList {
	pendingmap.Mu.RLock()
	defer pendingmap.Mu.RUnlock()
	pendingcat := pendingmap.pending[category]
	if pendingcat == nil {
		pendingcat = newPendingTxs()
	}
	pendingcat.Mu.RLock()
	defer pendingcat.Mu.RUnlock()
	if txlist := pendingcat.addrTxList[addr]; txlist != nil {
		return txlist
	}
	return nil
}
func (pendingmap *pendingsMap) addTxToTxListByAddrOfCategory(
	category basic.TransactionCategory, addr tpcrtypes.Address, tx *basic.Transaction) (bool, *basic.Transaction) {
	pendingmap.Mu.Lock()
	defer pendingmap.Mu.Unlock()
	pendingcat := pendingmap.pending[category]
	if pendingcat == nil {
		pendingcat = newPendingTxs()
	}
	pendingcat.Mu.Lock()
	defer pendingcat.Mu.Unlock()

	if txlist := pendingcat.addrTxList[addr]; txlist != nil {
		return txlist.txCoreAdd(tx)
	}
	return false, nil
}

func (pendingmap *pendingsMap) getTxListRemoveByAddrOfCategory(
	f1 func(string2 string, transaction *basic.Transaction),
	tx *basic.Transaction,
	category basic.TransactionCategory, addr tpcrtypes.Address) {
	pendingmap.Mu.Lock()
	defer pendingmap.Mu.Unlock()
	pendingcat := pendingmap.pending[category]
	if pendingcat == nil {
		pendingcat = newPendingTxs()
	}
	pendingcat.Mu.Lock()
	defer pendingcat.Mu.Unlock()
	if list := pendingcat.addrTxList[addr]; list != nil {
		removed, invalids := list.Remove(tx)
		if !removed {
			return
		}

		if list.Empty() {
			delete(pendingcat.addrTxList, addr)
		}

		// Postpone any invalidated transactions
		for _, txi := range invalids {
			txId, _ := txi.HashHex()
			f1(txId, txi)
		}
	}
}

func (pendingmap *pendingsMap) replaceTxOfAddrOfCategory(category basic.TransactionCategory, from tpcrtypes.Address,
	txId string, tx *basic.Transaction, isLocal bool,
	f1 func(category basic.TransactionCategory, txId string),
	f2 func(category basic.TransactionCategory),
	f3 func(category basic.TransactionCategory, tx *basic.Transaction, isLocal bool),
	f4 func(category basic.TransactionCategory, tx *basic.Transaction, isLocal bool),
	f5 func(txId string)) (bool, error) {
	pendingmap.Mu.Lock()
	defer pendingmap.Mu.Unlock()
	pendingcat := pendingmap.pending[category]
	if pendingcat == nil {
		pendingcat = newPendingTxs()
	}
	pendingcat.Mu.Lock()
	defer pendingcat.Mu.Unlock()
	if list := pendingcat.addrTxList[from]; list != nil && list.WhetherSameNonce(tx) {
		inserted, old := list.txCoreAdd(tx)
		if !inserted {
			return false, ErrReplaceUnderpriced
		}
		if old != nil {
			f1(category, txId)
			f2(category)
		}
		f3(category, tx, isLocal)
		f4(category, tx, isLocal)
		f5(txId)
	}
	return true, nil
}
func (pendingmap *pendingsMap) setTxListOfCategory(category basic.TransactionCategory, addr tpcrtypes.Address, txs *txCoreList) {
	pendingmap.Mu.Lock()
	defer pendingmap.Mu.Unlock()
	if pendingcat := pendingmap.pending[category]; pendingcat != nil {
		pendingcat.Mu.Lock()
		defer pendingcat.Mu.Unlock()
		pendingcat.addrTxList[addr] = txs
	}

}

type queueTxs struct {
	Mu                sync.RWMutex
	mapAddrTxCoreList map[tpcrtypes.Address]*txCoreList
}

func newQueueTxs() *queueTxs {
	addtxlist := &queueTxs{
		mapAddrTxCoreList: make(map[tpcrtypes.Address]*txCoreList, 0),
	}
	return addtxlist
}

type queuesMap struct {
	Mu    sync.RWMutex
	queue map[basic.TransactionCategory]*queueTxs
}

func newQueuesMap() *queuesMap {
	queues := &queuesMap{
		queue: make(map[basic.TransactionCategory]*queueTxs, 0),
	}
	queues.queue[basic.TransactionCategory_Topia_Universal] = newQueueTxs()
	queues.queue[basic.TransactionCategory_Eth] = newQueueTxs()
	return queues
}

func (queuemap *queuesMap) getTxsByCategory(category basic.TransactionCategory) []*basic.Transaction {
	queuemap.Mu.RLock()
	defer queuemap.Mu.RUnlock()
	queue := make([]*basic.Transaction, 0)
	queuecat := queuemap.queue[category]
	if queuecat == nil {
		queuecat = newQueueTxs()
	}
	queuecat.Mu.RLock()
	defer queuecat.Mu.RUnlock()
	for _, list := range queuecat.mapAddrTxCoreList {
		txs := list.Flatten()
		if len(txs) > 0 {
			queue = append(queue, txs...)
		}
	}
	return queue
}
func (queuemap *queuesMap) getAll() map[basic.TransactionCategory]*queueTxs {
	queuemap.Mu.RLock()
	defer queuemap.Mu.RUnlock()
	if mapqueue := queuemap.queue; mapqueue != nil {
		return mapqueue
	}
	return nil
}
func (queuemap *queuesMap) getQueueTxsByCategory(category basic.TransactionCategory) *queueTxs {
	queuemap.Mu.RLock()
	defer queuemap.Mu.RUnlock()
	if queuetxs := queuemap.queue[category]; queuetxs != nil {
		return queuetxs
	}
	return nil
}

func (queuemap *queuesMap) getTxListByAddrOfCategory(category basic.TransactionCategory, addr tpcrtypes.Address) *txCoreList {
	queuemap.Mu.RLock()
	defer queuemap.Mu.RUnlock()
	if queuecat := queuemap.queue[category]; queuecat != nil {
		queuecat.Mu.RLock()
		defer queuecat.Mu.RUnlock()

		if txlist := queuecat.mapAddrTxCoreList[addr]; txlist != nil {
			return txlist
		}
	}
	return nil
}
func (queuemap *queuesMap) getTxByNonceFromTxlistByAddrOfCategory(category basic.TransactionCategory, addr tpcrtypes.Address, Nonce uint64) *basic.Transaction {
	queuemap.Mu.RLock()
	defer queuemap.Mu.RUnlock()
	if queuecat := queuemap.queue[category]; queuecat != nil {
		queuecat.Mu.RLock()
		defer queuecat.Mu.RUnlock()
		if txlist := queuecat.mapAddrTxCoreList[addr]; txlist != nil {
			return txlist.txs.Get(Nonce)
		}
	}
	return nil
}

func (queuemap *queuesMap) getLenTxsByAddrOfCategory(category basic.TransactionCategory, addr tpcrtypes.Address) int {
	queuemap.Mu.RLock()
	defer queuemap.Mu.RUnlock()
	queuecat := queuemap.queue[category]
	if queuecat == nil {
		return 0
	}
	queuecat.Mu.RLock()
	defer queuecat.Mu.RUnlock()

	if txlist := queuecat.mapAddrTxCoreList[addr]; txlist != nil {
		txlist.lock.RLock()
		defer txlist.lock.RUnlock()
		return txlist.txs.Len()
	}
	return 0
}
func (queuemap *queuesMap) removeTxFromTxListByAddrOfCategory(
	category basic.TransactionCategory, addr tpcrtypes.Address, tx *basic.Transaction) {
	queuemap.Mu.Lock()
	defer queuemap.Mu.Unlock()
	queuecat := queuemap.queue[category]
	if queuecat == nil {
		return
	}
	queuemap.queue[category].Mu.Lock()
	defer queuemap.queue[category].Mu.Unlock()

	if txlist := queuemap.queue[category].mapAddrTxCoreList[addr]; txlist != nil {
		txlist.Remove(tx)
	}
}

//		pool.queues.getTxListByAddrOfCategory(category, addr).Remove(tx)
func (queuemap *queuesMap) replaceExecutablesDropOverLimit(
	f1 func(address tpcrtypes.Address) bool,
	f2 func(category basic.TransactionCategory, txId string),
	QueueMaxTxsAccount uint64,
	category basic.TransactionCategory, addr tpcrtypes.Address) int {
	queuemap.Mu.Lock()
	defer queuemap.Mu.Unlock()
	queuecat := queuemap.queue[category]
	if queuecat == nil {
		return 0
	}
	queuecat.Mu.Lock()
	defer queuecat.Mu.Unlock()
	txlist := queuemap.queue[category].mapAddrTxCoreList[addr]
	if txlist == nil {
		return 0
	}
	var caps []*basic.Transaction
	if f1(addr) {
		caps = txlist.CapLimitTxs(int(QueueMaxTxsAccount))
		for _, tx := range caps {
			txId, _ := tx.HashHex()
			f2(category, txId)
		}
	}
	return len(caps)
}

func (queuemap *queuesMap) replaceExecutablesTurnTx(f0 func(addr tpcrtypes.Address) uint64,
	f1 func(category basic.TransactionCategory, address tpcrtypes.Address, tx *basic.Transaction) (bool, *basic.Transaction),
	f2 func(category basic.TransactionCategory, txId string),
	f3 func(category basic.TransactionCategory, addr tpcrtypes.Address, tx *basic.Transaction),
	f4 func(category basic.TransactionCategory, txId string),
	f5 func(txId string),
	replaced []*basic.Transaction,
	category basic.TransactionCategory, addr tpcrtypes.Address) int {
	queuemap.Mu.Lock()
	defer queuemap.Mu.Unlock()
	queuecat := queuemap.queue[category]
	if queuecat == nil {
		return 0
	}
	queuecat.Mu.Lock()
	defer queuecat.Mu.Unlock()
	txlist := queuecat.mapAddrTxCoreList[addr]
	if txlist == nil {
		return 0
	}
	readies := txlist.Ready(f0(addr))
	for _, tx := range readies {
		txId, _ := tx.HashHex()
		// Try to insert the transaction into the pending queue
		f := func(addr tpcrtypes.Address, txId string, tx *basic.Transaction) bool {
			if txlist.txs.Get(tx.Head.Nonce) != tx {
				return false
			}
			inserted, old := f1(category, addr, tx)
			if !inserted {
				// An older transaction was existed, discard this
				f2(category, txId)
				return false
			} else {
				f3(category, addr, tx)
			}

			if old != nil {
				oldkey, _ := old.HashHex()
				f4(category, oldkey)
			}
			// Successful replace tx, bump the ActivationInterval
			f5(txId)
			return true
		}

		if f(addr, txId, tx) {
			replaced = append(replaced, tx)
		}
	}
	return len(replaced)
}

func (queuemap *queuesMap) replaceExecutablesDeleteEmpty(category basic.TransactionCategory, addr tpcrtypes.Address) {
	queuemap.Mu.Lock()
	defer queuemap.Mu.Unlock()
	queuecat := queuemap.queue[category]
	if queuecat == nil {
		return
	}
	queuemap.queue[category].Mu.Lock()
	defer queuemap.queue[category].Mu.Unlock()

	txlist := queuemap.queue[category].mapAddrTxCoreList[addr]
	if txlist == nil {
		return
	}
	if txlist.Empty() {
		delete(queuemap.queue[category].mapAddrTxCoreList, addr)
	}
}

func (queuemap *queuesMap) replaceExecutablesDropTooOld(category basic.TransactionCategory, addr tpcrtypes.Address,
	f1 func(address tpcrtypes.Address) uint64,
	f2 func(transactionCategory basic.TransactionCategory, txId string)) int {
	queuemap.Mu.Lock()
	defer queuemap.Mu.Unlock()
	queuecat := queuemap.queue[category]
	if queuecat == nil {
		return 0
	}
	queuemap.queue[category].Mu.Lock()
	defer queuemap.queue[category].Mu.Unlock()

	list := queuemap.queue[category].mapAddrTxCoreList[addr]
	if list == nil {
		return 0
	}
	// Drop all transactions that are deemed too old (low nonce)
	DropsLessNonce := list.RemovedTxForLessNonce(f1(addr))

	for _, tx := range DropsLessNonce {
		if txId, err := tx.HashHex(); err != nil {
		} else {
			f2(category, txId)
		}
	}
	return len(DropsLessNonce)
}

func (queuemap *queuesMap) getTxListRemoveFutureByAddrOfCategory(tx *basic.Transaction, f1 func(string2 string), key string, category basic.TransactionCategory, addr tpcrtypes.Address) {
	queuemap.Mu.Lock()
	defer queuemap.Mu.Unlock()
	queuecat := queuemap.queue[category]
	if queuecat == nil {
		return
	}
	queuecat.Mu.Lock()
	defer queuecat.Mu.Unlock()
	future := queuemap.queue[category].mapAddrTxCoreList[addr]
	if future == nil {
		return
	}
	future.Remove(tx)
	if future.Empty() {
		txs := queuemap.queue[category]
		if txs == nil {
			return
		}
		delete(txs.mapAddrTxCoreList, addr)
		f1(key)
	}
}

func (queuemap *queuesMap) getAddrTxListOfCategory(category basic.TransactionCategory) map[tpcrtypes.Address]*txCoreList {
	queuemap.Mu.RLock()
	defer queuemap.Mu.RUnlock()
	if queuecat := queuemap.queue[category]; queuecat != nil {
		queuecat.Mu.RLock()
		defer queuecat.Mu.RUnlock()
		if maptxlist := queuecat.mapAddrTxCoreList; maptxlist != nil {
			return maptxlist
		}
	}
	return nil
}

func (queuemap *queuesMap) replaceAddrTxList() []tpcrtypes.Address {
	queuemap.Mu.Lock()
	defer queuemap.Mu.Unlock()
	replaceAddrs := make([]tpcrtypes.Address, 0)
	for category, _ := range queuemap.queue {
		queuecat := queuemap.queue[category]
		if queuecat == nil {
			return nil
		}
		queuecat.Mu.Lock()
		defer queuecat.Mu.Unlock()
		maptxlist := queuemap.queue[category].mapAddrTxCoreList
		if maptxlist == nil {
			return nil
		}
		for addr, _ := range maptxlist {
			replaceAddrs = append(replaceAddrs, addr)
		}
	}
	return replaceAddrs
}

func (queuemap *queuesMap) getStatsOfCategory(category basic.TransactionCategory) int {
	queuemap.Mu.RLock()
	defer queuemap.Mu.RUnlock()
	queuecat := queuemap.queue[category]
	if queuecat == nil {
		return 0
	}
	queuecat.Mu.RLock()
	defer queuecat.Mu.RUnlock()
	queue := 0
	maptxlist := queuecat.mapAddrTxCoreList
	if maptxlist == nil {
		return 0
	}
	for _, list := range maptxlist {
		queue += list.Len()
	}
	return queue
}

func (queuemap *queuesMap) removeTxsForTruncateQueue(category basic.TransactionCategory,
	f1 func(tpcrtypes.Address) bool, f2 func(string2 string) time.Time,
	f3 func(transactionCategory basic.TransactionCategory, txHash string) *basic.Transaction,
	f4 func(category basic.TransactionCategory, txId string),
	f5 func(f51 func(txId string, tx *basic.Transaction), tx *basic.Transaction, category basic.TransactionCategory, addr tpcrtypes.Address),
	f511 func(category basic.TransactionCategory, hash string),
	f512 func(category basic.TransactionCategory, txId string) *basic.Transaction,
	f513 func(txId string),
	f514 func(txId string, category basic.TransactionCategory),
	f6 func(txId string),
	queued int, QueueMaxTxsGlobal uint64) {
	queuemap.Mu.Lock()
	defer queuemap.Mu.Unlock()
	queuecat := queuemap.queue[category]
	if queuecat == nil {
		return
	}
	queuecat.Mu.Lock()
	defer queuecat.Mu.Unlock()
	txs := make(txsByActivationInterval, 0, len(queuemap.queue[category].mapAddrTxCoreList))
	for addr := range queuemap.queue[category].mapAddrTxCoreList {
		if f1(addr) {
			list := queuemap.queue[category].mapAddrTxCoreList[addr].Flatten()
			for _, tx := range list {
				txId, _ := tx.HashHex()
				txs = append(txs, txByActivationInterval{txId, f2(txId)})
			}
		}
	}
	sort.Sort(txs)
	for cnt := uint64(queued); cnt > QueueMaxTxsGlobal && len(txs) > 0; {
		txH := txs[len(txs)-1]
		txs = txs[:len(txs)-1]
		txId := txH.txId
		if tx := f3(category, txId); tx != nil {
			addr := tpcrtypes.Address(tx.Head.FromAddr)
			// Remove it from the list of known transactions
			f4(category, txId)

			f51 := func(txId string, tx *basic.Transaction) {
				from := tpcrtypes.Address(tx.Head.FromAddr)
				inserted, old := queuecat.mapAddrTxCoreList[from].txCoreAdd(tx)
				if !inserted {
					// An older transaction was existed
					return
				}
				if old != nil {
					oldTxId, _ := old.HashHex()
					f511(category, oldTxId)
				}
				if f512(category, txId) == nil {
					f513(txId)
				}
				f514(txId, category)
			}
			f5(f51, tx, category, addr)
			// Remove the transaction from the pending lists and reset the account nonce
			if future := queuecat.mapAddrTxCoreList[addr]; future != nil {
				future.Remove(tx)
				if future.Empty() {
					if droptxs := queuemap.queue[category]; droptxs != nil {
						delete(droptxs.mapAddrTxCoreList, addr)
					}
					f6(txId)
				}
			}

		}
		cnt -= 1
		continue
	}
}

func (queuemap *queuesMap) removeTxForLifeTime(category basic.TransactionCategory, f0 func(address tpcrtypes.Address) bool,
	f1 func(string2 string) time.Duration, duration2 time.Duration, f2 func(string2 string)) {

	queuemap.Mu.Lock()
	defer queuemap.Mu.Unlock()
	if queuecat := queuemap.queue[category]; queuecat != nil {
		queuecat.Mu.Lock()
		defer queuecat.Mu.Unlock()
		for addr, txlist := range queuecat.mapAddrTxCoreList {
			if f0(addr) {
				continue
			}
			list := txlist.txs.Flatten()
			for _, tx := range list {
				txId, _ := tx.HashHex()
				if f1(txId) > duration2 {
					f2(txId)
				}
			}
		}

	}
}
func (queuemap *queuesMap) republicTx(category basic.TransactionCategory,
	f1 func(string2 string) time.Duration, time2 time.Duration, f2 func(tx *basic.Transaction)) {
	queuemap.Mu.Lock()
	defer queuemap.Mu.Unlock()
	queuecat := queuemap.queue[category]
	if queuecat == nil {
		return
	}
	queuecat.Mu.Lock()
	defer queuecat.Mu.Unlock()
	for _, txlist := range queuecat.mapAddrTxCoreList {
		list := txlist.txs.Flatten()
		for _, tx := range list {
			txId, _ := tx.HashHex()
			if f1(txId) > time2 {
				f2(tx)
			}
		}
	}
}

func (l *txCoreList) FlattenRepublic(f1 func(string2 string) time.Duration, time2 time.Duration, f2 func(tx *basic.Transaction)) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	list := l.txs.Flatten()
	for _, tx := range list {
		txId, _ := tx.HashHex()
		if f1(txId) > time2 {
			f2(tx)
		}
	}
}

func (queuemap *queuesMap) addTxByKeyOfCategory(
	f1 func(category basic.TransactionCategory, key string),
	f2 func(category basic.TransactionCategory, key string) *basic.Transaction,
	f3 func(string2 string),
	f4 func(category basic.TransactionCategory, transaction *basic.Transaction, local bool),
	f5 func(string2 string, category basic.TransactionCategory, local bool),
	key string, tx *basic.Transaction, local bool, addAll bool) (bool, error) {

	queuemap.Mu.Lock()
	defer queuemap.Mu.Unlock()
	category := basic.TransactionCategory(tx.Head.Category)
	if queuecat := queuemap.queue[category]; queuecat != nil {
		queuecat.Mu.Lock()
		defer queuecat.Mu.Unlock()
		from := tpcrtypes.Address(tx.Head.FromAddr)
		if queuecat.mapAddrTxCoreList[from] == nil {
			queuecat.mapAddrTxCoreList[from] = newCoreList(false)
		}

		inserted, old := queuecat.mapAddrTxCoreList[from].txCoreAdd(tx)
		if !inserted {
			// An older transaction was existed
			return false, ErrReplaceUnderpriced
		}
		if old != nil {
			oldTxId, _ := old.HashHex()
			f1(category, oldTxId)
		}
		//if pool.allTxsForLook.getAllTxsLookupByCategory(category).Get(key) == nil && !addAll {
		if f2(category, key) == nil && !addAll {
			f3(key)
		}
		if addAll {
			f4(category, tx, local)
		}
		// If we never record the ActivationInterval, do it right now.
		f5(key, category, local)
		return old != nil, nil
	}
	return false, ErrQueueEmpty
}

type allTxsLookupMap struct {
	Mu  sync.RWMutex
	all map[basic.TransactionCategory]*txLookup
}

func newAllTxsLookupMap() *allTxsLookupMap {
	allMap := &allTxsLookupMap{
		all: make(map[basic.TransactionCategory]*txLookup, 0),
	}
	allMap.setAllTxsLookup(basic.TransactionCategory_Topia_Universal, newTxLookup())
	allMap.setAllTxsLookup(basic.TransactionCategory_Eth, newTxLookup())
	return allMap
}
func (alltxsmap *allTxsLookupMap) getAll() map[basic.TransactionCategory]*txLookup {
	alltxsmap.Mu.RLock()
	defer alltxsmap.Mu.RUnlock()
	if all := alltxsmap.all; all != nil {
		return all
	}
	return nil

}

func (alltxsmap *allTxsLookupMap) getAllCount() int {
	alltxsmap.Mu.RLock()
	defer alltxsmap.Mu.RUnlock()
	var cnt int
	for category, _ := range alltxsmap.all {
		cnt += alltxsmap.all[category].Count()
	}
	return cnt
}
func (alltxsmap *allTxsLookupMap) getLocalCountByCategory(category basic.TransactionCategory) int {
	alltxsmap.Mu.RLock()
	defer alltxsmap.Mu.RUnlock()
	if lookuptxs := alltxsmap.all[category]; lookuptxs != nil {
		return lookuptxs.LocalCount()
	}
	return 0
}
func (alltxsmap *allTxsLookupMap) getRemoteCountByCategory(category basic.TransactionCategory) int {
	alltxsmap.Mu.RLock()
	defer alltxsmap.Mu.RUnlock()
	if lookuptxs := alltxsmap.all[category]; lookuptxs != nil {
		return lookuptxs.RemoteCount()
	}
	return 0
}

func (alltxsmap *allTxsLookupMap) getAllTxsLookupByCategory(category basic.TransactionCategory) *txLookup {
	alltxsmap.Mu.RLock()
	defer alltxsmap.Mu.RUnlock()
	if txlookup := alltxsmap.all[category]; txlookup != nil {
		return txlookup
	}
	return nil
}
func (alltxsmap *allTxsLookupMap) getSegmentFromAllTxsLookupByCategory(category basic.TransactionCategory) int {
	alltxsmap.Mu.RLock()
	defer alltxsmap.Mu.RUnlock()
	if txlookup := alltxsmap.all[category]; txlookup != nil {
		return txlookup.Segments()
	}
	return 0
}

func (alltxsmap *allTxsLookupMap) getCountFromAllTxsLookupByCategory(category basic.TransactionCategory) int {
	alltxsmap.Mu.RLock()
	defer alltxsmap.Mu.RUnlock()
	if txlookup := alltxsmap.all[category]; txlookup != nil {
		return len(txlookup.locals) + len(txlookup.remotes)
	}
	return 0
}

func (alltxsmap *allTxsLookupMap) getLocalKeyTxFromAllTxsLookupByCategory(category basic.TransactionCategory) map[string]*basic.Transaction {
	alltxsmap.Mu.RLock()
	defer alltxsmap.Mu.RUnlock()
	if txlookup := alltxsmap.all[category]; txlookup != nil {
		txlookup.lock.RLock()
		defer txlookup.lock.RUnlock()
		return txlookup.locals
	}
	return nil
}
func (alltxsmap *allTxsLookupMap) getTxFromKeyFromAllTxsLookupByCategory(category basic.TransactionCategory, txHash string) *basic.Transaction {
	alltxsmap.Mu.RLock()
	defer alltxsmap.Mu.RUnlock()
	if txlookup := alltxsmap.all[category]; txlookup != nil {
		return txlookup.Get(txHash)
	}
	return nil
}

func (alltxsmap *allTxsLookupMap) remoteToLocalsAllTxsLookupByCategory(category basic.TransactionCategory, locals *accountSet) int {
	alltxsmap.Mu.Lock()
	defer alltxsmap.Mu.Unlock()
	if txlookup := alltxsmap.all[category]; txlookup != nil {
		return txlookup.RemoteToLocals(locals)
	}
	return 0
}

func (alltxsmap *allTxsLookupMap) addTxToAllTxsLookupByCategory(category basic.TransactionCategory,
	tx *basic.Transaction, isLocal bool) {
	alltxsmap.Mu.Lock()
	defer alltxsmap.Mu.Unlock()
	if catAllMap := alltxsmap.all[category]; catAllMap != nil {
		catAllMap.Add(tx, isLocal)
	}

}

func (alltxsmap *allTxsLookupMap) removeTxHashFromAllTxsLookupByCategory(category basic.TransactionCategory, txhash string) {
	alltxsmap.Mu.Lock()
	defer alltxsmap.Mu.Unlock()
	if txlookup := alltxsmap.all[category]; txlookup != nil {
		txlookup.Remove(txhash)
	}
}

func (alltxsmap *allTxsLookupMap) getRemoteMapTxsLookupByCategory(category basic.TransactionCategory) map[string]*basic.Transaction {
	alltxsmap.Mu.RLock()
	defer alltxsmap.Mu.RUnlock()
	if txlookup := alltxsmap.all[category]; txlookup != nil {
		txlookup.lock.RLock()
		defer txlookup.lock.RUnlock()
		return txlookup.remotes
	}
	return nil
}

func (alltxsmap *allTxsLookupMap) setAllTxsLookup(category basic.TransactionCategory, txlook *txLookup) {
	alltxsmap.Mu.Lock()
	defer alltxsmap.Mu.Unlock()
	if all := alltxsmap.all; all != nil {
		all[category] = txlook
	}
}
func (alltxsmap *allTxsLookupMap) removeAllTxsLookupByCategory(category basic.TransactionCategory) {
	alltxsmap.Mu.Lock()
	defer alltxsmap.Mu.Unlock()
	delete(alltxsmap.all, category)
}

type activationInterval struct {
	Mu    sync.RWMutex
	activ map[string]time.Time
}

func newActivationInterval() *activationInterval {
	activ := &activationInterval{
		activ: make(map[string]time.Time),
	}
	return activ
}
func (activ *activationInterval) getAll() map[string]time.Time {
	activ.Mu.Lock()
	defer activ.Mu.Unlock()
	if act := activ.activ; act != nil {
		return act
	}
	return nil
}
func (activ *activationInterval) getTxActivByKey(key string) time.Time {
	activ.Mu.Lock()
	defer activ.Mu.Unlock()
	return activ.activ[key]
}
func (activ *activationInterval) setTxActiv(key string, time2 time.Time) {
	activ.Mu.Lock()
	defer activ.Mu.Unlock()
	activ.activ[key] = time2
	return
}
func (activ *activationInterval) removeTxActiv(key string) {
	activ.Mu.Lock()
	defer activ.Mu.Unlock()
	delete(activ.activ, key)
	return
}

type txHashCategory struct {
	Mu              sync.RWMutex
	hashCategoryMap map[string]basic.TransactionCategory
}

func newTxHashCategory() *txHashCategory {
	hashCat := &txHashCategory{
		hashCategoryMap: make(map[string]basic.TransactionCategory),
	}
	return hashCat
}
func (hashCat *txHashCategory) getAll() map[string]basic.TransactionCategory {
	hashCat.Mu.Lock()
	defer hashCat.Mu.Unlock()
	if hc := hashCat.hashCategoryMap; hc != nil {
		return hc
	}
	return nil
}
func (hashCat *txHashCategory) getByHash(key string) basic.TransactionCategory {
	hashCat.Mu.Lock()
	defer hashCat.Mu.Unlock()
	return hashCat.hashCategoryMap[key]
}
func (hashCat *txHashCategory) setHashCat(key string, category basic.TransactionCategory) {
	hashCat.Mu.Lock()
	defer hashCat.Mu.Unlock()
	hashCat.hashCategoryMap[key] = category
}
func (hashCat *txHashCategory) removeHashCat(key string) {
	hashCat.Mu.Lock()
	defer hashCat.Mu.Unlock()
	delete(hashCat.hashCategoryMap, key)
}

type accountSet struct {
	accounts map[tpcrtypes.Address]struct{}
	cache    *[]tpcrtypes.Address
}

func newAccountSet(addrs ...tpcrtypes.Address) *accountSet {
	as := &accountSet{
		accounts: make(map[tpcrtypes.Address]struct{}),
	}
	for _, addr := range addrs {
		as.add(addr)
	}
	return as
}

func (accSet *accountSet) contains(addr tpcrtypes.Address) bool {

	_, exist := accSet.accounts[addr]
	return exist
}
func (accSet *accountSet) containsTx(tx *basic.Transaction) bool {
	addr := tpcrtypes.Address(tx.Head.FromAddr)
	return accSet.contains(addr)
}

func (accSet *accountSet) empty() bool {
	return len(accSet.accounts) == 0
}
func (accSet *accountSet) len() int {
	return len(accSet.accounts)
}

func (accSet *accountSet) add(addr tpcrtypes.Address) {
	accSet.accounts[addr] = struct{}{}
	accSet.cache = nil
}
func (accSet *accountSet) addTx(tx *basic.Transaction) {
	addr := tpcrtypes.Address(tx.Head.FromAddr)
	accSet.add(addr)
}
func (accSet *accountSet) merge(other *accountSet) {
	for addr := range other.accounts {
		accSet.accounts[addr] = struct{}{}
	}
	accSet.cache = nil
}

func (accSet *accountSet) RemoveAccount(addr tpcrtypes.Address) {
	delete(accSet.accounts, addr)
}

func (accSet *accountSet) flatten() []tpcrtypes.Address {
	if accSet.cache == nil {
		accounts := make([]tpcrtypes.Address, 0, len(accSet.accounts))
		for acc := range accSet.accounts {
			accounts = append(accounts, acc)
		}
		accSet.cache = &accounts
	}
	return *accSet.cache
}

type txLookup struct {
	segments int
	lock     sync.RWMutex
	locals   map[string]*basic.Transaction
	remotes  map[string]*basic.Transaction
}

func (t *txLookup) Range(f func(key string, tx *basic.Transaction, local bool) bool, local bool, remote bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	if local {
		for k, v := range t.locals {
			if !f(k, v, true) {
				return
			}
		}
	}
	if remote {
		for k, v := range t.remotes {
			if !f(k, v, false) {
				return
			}
		}
	}
}

func newTxLookup() *txLookup {
	return &txLookup{
		locals:  make(map[string]*basic.Transaction),
		remotes: make(map[string]*basic.Transaction),
	}
}
func (t *txLookup) Get(key string) *basic.Transaction {
	t.lock.RLock()
	defer t.lock.RUnlock()

	if tx := t.locals[key]; tx != nil {
		return tx
	}
	return t.remotes[key]
}
func (t *txLookup) GetAllLocalKeyTxs() map[string]*basic.Transaction {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.locals
}
func (t *txLookup) GetAllRemoteKeyTxs() map[string]*basic.Transaction {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.remotes
}

// GetLocalTx returns a transaction if it exists in the lookup, or nil if not found.
func (t *txLookup) GetLocalTx(key string) *basic.Transaction {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.locals[key]
}

// GetRemoteTx returns a transaction if it exists in the lookup, or nil if not found.
func (t *txLookup) GetRemoteTx(key string) *basic.Transaction {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.remotes[key]
}

func (t *txLookup) Count() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return len(t.locals) + len(t.remotes)
}

// LocalCount returns the current number of local transactions in the lookup.
func (t *txLookup) LocalCount() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return len(t.locals)
}

// RemoteCount returns the current number of remote transactions in the lookup.
func (t *txLookup) RemoteCount() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return len(t.remotes)
}

// Segments returns the current number of Quota used in the lookup.
func (t *txLookup) Segments() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.segments
}

func (t *txLookup) Add(tx *basic.Transaction, local bool) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.segments += numSegments(tx)
	if txId, err := tx.HashHex(); err == nil {
		if local {
			t.locals[txId] = tx
		} else {
			t.remotes[txId] = tx
		}
	}

}
func (t *txLookup) Remove(key string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	tx, ok := t.locals[key]
	if !ok {
		tx, ok = t.remotes[key]
	}
	if !ok {
		//tplog.Logger.Infof("No transaction found to be deleted", "txKey", key)
		return
	}
	t.segments -= numSegments(tx)
	delete(t.locals, key)
	delete(t.remotes, key)
}

// RemoteToLocals migrates the transactions belongs to the given locals to locals
// set. The assumption is held the locals set is thread-safe to be used.
func (t *txLookup) RemoteToLocals(locals *accountSet) int {
	t.lock.Lock()
	defer t.lock.Unlock()

	var migrated int
	for key, tx := range t.remotes {
		if locals.containsTx(tx) {
			t.locals[key] = tx
			delete(t.remotes, key)
			migrated += 1
		}
	}
	return migrated
}

// priceHp is a heap.Interface implementation over transactions for retrieving
// price-sorted transactions to discard when the pool fills up.
type priceHp struct {
	list []*basic.Transaction
}

func (h *priceHp) Len() int      { return len(h.list) }
func (h *priceHp) Swap(i, j int) { h.list[i], h.list[j] = h.list[j], h.list[i] }

func (h *priceHp) Less(i, j int) bool {
	switch h.cmp(h.list[i], h.list[j]) {
	case -1:
		return true
	case 1:
		return false
	default:
		var Noncei, Noncej uint64
		Noncei = h.list[i].Head.Nonce
		Noncej = h.list[j].Head.Nonce
		return Noncei > Noncej
	}
}
func (h *priceHp) cmp(a, b *basic.Transaction) int {

	if GasPrice(a) < GasPrice(b) {
		return -1
	} else if GasPrice(a) > GasPrice(b) {
		return 1
	} else {
		return 0
	}
}

func (h *priceHp) Push(x interface{}) {
	tx := x.(*basic.Transaction)
	h.list = append(h.list, tx)
}

func (h *priceHp) Pop() interface{} {
	old := h.list
	n := len(old)
	x := old[n-1]
	old[n-1] = nil
	h.list = old[0 : n-1]
	return x
}

type txSortedList struct {
	Mu         sync.RWMutex
	Pricedlist map[basic.TransactionCategory]*txPricedList
	//we can add other sorted list hear
}

func newTxSortedList(all *txLookup) *txSortedList {
	txSorted := &txSortedList{Pricedlist: make(map[basic.TransactionCategory]*txPricedList, 0)}
	txSorted.setPricedlist(basic.TransactionCategory_Topia_Universal, newTxPricedList(all))
	return txSorted
}

func (txsorts *txSortedList) getAllPricedlist() map[basic.TransactionCategory]*txPricedList {
	txsorts.Mu.Lock()
	defer txsorts.Mu.Unlock()
	return txsorts.Pricedlist
}
func (txsorts *txSortedList) getPricedlistByCategory(category basic.TransactionCategory) *txPricedList {
	txsorts.Mu.Lock()
	defer txsorts.Mu.Unlock()
	if pricelistcap := txsorts.Pricedlist[category]; pricelistcap != nil {
		return txsorts.Pricedlist[category]

	}
	return nil
}
func (txsorts *txSortedList) getAllLocalTxsByCategory(category basic.TransactionCategory) map[string]*basic.Transaction {
	txsorts.Mu.Lock()
	defer txsorts.Mu.Unlock()
	if pricelistcap := txsorts.Pricedlist[category]; pricelistcap != nil {
		if allTxs := pricelistcap.all; allTxs != nil {
			return pricelistcap.all.locals
		}
	}
	return nil
}
func (txsorts *txSortedList) getAllRemoteTxsByCategory(category basic.TransactionCategory) map[string]*basic.Transaction {
	txsorts.Mu.Lock()
	defer txsorts.Mu.Unlock()
	if pricelistcap := txsorts.Pricedlist[category]; pricelistcap != nil {
		if alltxs := pricelistcap.all; alltxs != nil {
			return pricelistcap.all.remotes
		}
	}
	return nil
}

func (txsorts *txSortedList) DiscardFromPricedlistByCategor(category basic.TransactionCategory, segments int, force bool) ([]*basic.Transaction, bool) {
	txsorts.Mu.Lock()
	defer txsorts.Mu.Unlock()
	pricelistcap := txsorts.Pricedlist[category]
	if pricelistcap == nil {
		return nil, false
	}
	discard := func(segments int, force bool) ([]*basic.Transaction, bool) {
		drop := make([]*basic.Transaction, 0, segments) // Remote underpriced transactions to drop
		for segments > 0 {
			// Discard stale transactions if found during cleanup
			if pricelistcap.remoteTxs.Len() == 0 {
				//Stop if heap is empty
				break
			}
			tx := heap.Pop(&pricelistcap.remoteTxs).(*basic.Transaction)
			if txId, err := tx.HashHex(); err == nil {
				if pricelistcap.all.GetRemoteTx(txId) == nil { // Removed or migrated
					atomic.AddInt64(&pricelistcap.stales, -1)
					continue
				}
			}
			drop = append(drop, tx)
			segments -= numSegments(tx)
		}
		// If we still can't make enough room for the new transaction
		if segments > 0 && !force {
			for _, tx := range drop {
				heap.Push(&pricelistcap.remoteTxs, tx)
			}
			return nil, false
		}
		return drop, true
	}
	return discard(segments, force)

}

func (txsorts *txSortedList) ReheapForPricedlistByCategory(category basic.TransactionCategory) {
	txsorts.Mu.Lock()
	defer txsorts.Mu.Unlock()
	if listcat := txsorts.Pricedlist[category]; listcat != nil {
		listcat.Reheap()
	}
}
func (txsorts *txSortedList) putTxToPricedlistByCategory(category basic.TransactionCategory, tx *basic.Transaction, isLocal bool) {
	txsorts.Mu.Lock()
	defer txsorts.Mu.Unlock()
	if catlist := txsorts.Pricedlist[category]; catlist != nil {
		catlist.Put(tx, isLocal)
	}
}
func (txsorts *txSortedList) removedPricedlistByCategory(category basic.TransactionCategory, num int) {
	txsorts.Mu.Lock()
	defer txsorts.Mu.Unlock()
	if catlist := txsorts.Pricedlist[category]; catlist != nil {
		catlist.Removed(num)
	}
}
func (txsorts *txSortedList) setPricedlist(category basic.TransactionCategory, txpriced *txPricedList) {
	txsorts.Mu.Lock()
	defer txsorts.Mu.Unlock()
	txsorts.Pricedlist[category] = txpriced
	return
}

type txPricedList struct {
	stales    int64
	all       *txLookup  // Pointer to the map of all transactions
	remoteTxs priceHp    // Heaps of prices of all the stored **remote** transactions
	reheapMu  sync.Mutex // Mutex asserts that only one routine is reheaping the list
}

// newTxPricedList creates a new price-sorted transaction heap.
func newTxPricedList(all *txLookup) *txPricedList {
	return &txPricedList{
		all: all,
	}
}

// Put inserts a new transaction into the heap.
func (l *txPricedList) Put(tx *basic.Transaction, local bool) {
	if local {
		return
	}
	// Insert every new transaction to the urgent heap first; Discard will balance the heaps
	heap.Push(&l.remoteTxs, tx)
}

//Removed Delete transactions and reset the queue when the number of deleted transactions
//is greater than a quarter of the number in the queue.
func (l *txPricedList) Removed(count int) {
	stales := atomic.AddInt64(&l.stales, int64(count))
	if int(stales) <= (len(l.remoteTxs.list))/4 {
		return
	}
	l.Reheap()
}

// Underpriced checks whether a transaction is cheaper than (or as cheap as) the
// lowest priced (remote) transaction currently being tracked.
func (l *txPricedList) Underpriced(tx *basic.Transaction) bool {
	// Note: with two queues, being underpriced is defined as being worse than the worst item
	// in all non-empty queues if there is any. If both queues are empty then nothing is underpriced.
	return (l.underpricedFor(&l.remoteTxs, tx) || len(l.remoteTxs.list) == 0) &&
		(len(l.remoteTxs.list) != 0)
}

// underpricedFor checks whether a transaction is cheaper than (or as cheap as) the
// lowest priced (remote) transaction in the given heap.
func (l *txPricedList) underpricedFor(h *priceHp, tx *basic.Transaction) bool {
	// Discard stale price points if found at the heap start
	for len(h.list) > 0 {
		head := h.list[0]
		txId, _ := head.HashHex()
		if l.all.GetRemoteTx(txId) == nil { // Removed or migrated
			atomic.AddInt64(&l.stales, -1)
			heap.Pop(h)
			continue
		}
		break

	}
	// Check if the transaction is underpriced or not
	if len(h.list) == 0 {
		return false // There is no remote transaction at all.
	}
	// If the remote transaction is even cheaper than the
	// cheapest one tracked locally, reject it.
	return h.cmp(h.list[0], tx) >= 0
}

// Discard finds a number of most underpriced transactions, removes them from the
// priced list and returns them for further removal from the entire pool.
// Note local transaction won't be considered for eviction.
func (l *txPricedList) Discard(segments int, force bool) ([]*basic.Transaction, bool) {
	drop := make([]*basic.Transaction, 0, segments) // Remote underpriced transactions to drop
	for segments > 0 {
		// Discard stale transactions if found during cleanup
		if l.remoteTxs.Len() == 0 {
			//Stop if heap is empty
			break
		}
		tx := heap.Pop(&l.remoteTxs).(*basic.Transaction)
		if txId, err := tx.HashHex(); err == nil {
			if l.all.GetRemoteTx(txId) == nil { // Removed or migrated
				atomic.AddInt64(&l.stales, -1)
				continue
			}
		}
		drop = append(drop, tx)
		segments -= numSegments(tx)
	}
	// If we still can't make enough room for the new transaction
	if segments > 0 && !force {
		for _, tx := range drop {
			heap.Push(&l.remoteTxs, tx)
		}
		return nil, false
	}
	return drop, true
}

// Reheap forcibly rebuilds the heap based on the current remote transaction set.
func (l *txPricedList) Reheap() {
	l.reheapMu.Lock()
	defer l.reheapMu.Unlock()
	atomic.StoreInt64(&l.stales, 0)
	l.remoteTxs.list = make([]*basic.Transaction, 0, l.all.RemoteCount())
	l.all.Range(func(key string, tx *basic.Transaction, local bool) bool {
		l.remoteTxs.list = append(l.remoteTxs.list, tx)
		return true
	}, false, true) // Only iterate remotes
	heap.Init(&l.remoteTxs)
}

// numSegments calculates the number of segments needed for a single transaction.
func numSegments(tx *basic.Transaction) int {
	return int((tx.Size() + txSegmentSize - 1) / txSegmentSize)
}

type TxByNonce []*basic.Transaction

func (s TxByNonce) Len() int { return len(s) }
func (s TxByNonce) Less(i, j int) bool {
	return s[i].Head.Nonce < s[j].Head.Nonce
}
func (s TxByNonce) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// CntAccountItem is an item  we manage in a priority queue for cnt.
type CntAccountItem struct {
	accountAddr tpcrtypes.Address
	cnt         int // The priority of CntAccountItem in the queue is counts of account in txPool.
	// The index is needed by update and is maintained by the heap.Interface methods.
	index int // The index of the CntAccountItem item in the heap.
}

// A CntAccountHeap implements heap.Interface and holds GreyAccCnt.
type CntAccountHeap []*CntAccountItem

func (pq CntAccountHeap) Len() int { return len(pq) }

func (pq CntAccountHeap) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return pq[i].cnt < pq[j].cnt
}

func (pq CntAccountHeap) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *CntAccountHeap) Push(x interface{}) {
	n := len(*pq)
	item := x.(*CntAccountItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *CntAccountHeap) Pop() interface{} {
	old := *pq
	n := len(old)
	GreyAccCnt := old[n-1]
	GreyAccCnt.index = -1 // for safety
	*pq = old[0 : n-1]
	return GreyAccCnt
}

// txByActivationInterval is an account address tagged with its last activity timestamp.
type txByActivationInterval struct {
	txId               string
	ActivationInterval time.Time
}

type txsByActivationInterval []txByActivationInterval

func (t txsByActivationInterval) Len() int { return len(t) }
func (t txsByActivationInterval) Less(i, j int) bool {
	return t[i].ActivationInterval.Before(t[j].ActivationInterval)
}
func (t txsByActivationInterval) Swap(i, j int) { t[i], t[j] = t[j], t[i] }

type TxByPriceAndTime []*basic.Transaction

func (s TxByPriceAndTime) Len() int { return len(s) }
func (s TxByPriceAndTime) Less(i, j int) bool {
	if GasPrice(s[i]) < GasPrice(s[j]) {
		return true
	}
	if GasPrice(s[i]) == GasPrice(s[j]) {
		//		return s[i].Time.Before(s[j].Time)
		return time.Unix(int64(s[i].Head.TimeStamp), 0).Before(time.Unix(int64(s[j].Head.TimeStamp), 0))
	}
	return false
}
func (s TxByPriceAndTime) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s *TxByPriceAndTime) Push(x interface{}) {
	*s = append(*s, x.(*basic.Transaction))
}
func (s *TxByPriceAndTime) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
}

type TxsByPriceAndNonce struct {
	txs   map[tpcrtypes.Address][]*basic.Transaction
	heads TxByPriceAndTime
}

func NewTxsByPriceAndNonce(txs map[tpcrtypes.Address][]*basic.Transaction) *TxsByPriceAndNonce {
	heads := make(TxByPriceAndTime, 0, len(txs))
	for from, accTxs := range txs {
		tx := accTxs[0]
		heads = append(heads, tx)
		txs[from] = accTxs[1:]
	}
	heap.Init(&heads)
	return &TxsByPriceAndNonce{
		txs:   txs,
		heads: heads,
	}
}

// Peek returns the next transaction by price.
func (t *TxsByPriceAndNonce) Peek() *basic.Transaction {
	if len(t.heads) == 0 {
		return nil
	}
	return t.heads[0]
}

// Shift replaces the current best head with the next one from the same account.
func (t *TxsByPriceAndNonce) Shift() {
	acc := tpcrtypes.Address(t.heads[0].Head.FromAddr)
	if txs, ok := t.txs[acc]; ok && len(txs) > 0 {
		wrapped := txs[0]
		t.heads[0], t.txs[acc] = wrapped, txs[1:]
		heap.Fix(&t.heads, 0)
		return
	}

	heap.Pop(&t.heads)
}

// Pop removes the best transaction, *not* replacing it with the next one from
// the same account. This should be used when a transaction cannot be executed
// and hence all subsequent ones should be discarded from the same account.
func (t *TxsByPriceAndNonce) Pop() {
	heap.Pop(&t.heads)
}
