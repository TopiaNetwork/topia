package transactionpool

import (
	"container/heap"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

type TransactionState string

const (
	StateTxAdded                   TransactionState = "Tx Added"
	StateTxRemoved                                  = "tx removed"
	StateTxDiscardForReplaceFailed                  = "Tx Discard For replace failed"
	StateTxDiscardForTxPoolFull                     = "Tx Discard For TxPool is Full"
	StateTxAddToQueue                               = "Tx Add To Queue"
)

type TxRepublicPolicy byte

const (
	TxRepublicTime TxRepublicPolicy = iota
	TxRepublicHeight
	TxRepublicTimeAndHeight
	TxRepublicTimeOrHeight
)

type BlockAddedEvent struct{ Block *tpchaintypes.Block }

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
	items map[uint64]*txbasic.Transaction
	index *nonceHp
	cache []*txbasic.Transaction
	size  int64
}

func newtxSortedMapByNonce() *txSortedMapByNonce {
	return &txSortedMapByNonce{
		items: make(map[uint64]*txbasic.Transaction),
		index: new(nonceHp),
		size:  0,
	}
}

func (m *txSortedMapByNonce) Get(nonce uint64) *txbasic.Transaction {
	return m.items[nonce]
}

// Put inserts a new transaction into the map and updates the nonce index of the map.
//If a transaction already has the same Nonce, it will be overwritten
func (m *txSortedMapByNonce) Put(tx *txbasic.Transaction) {

	nonce := tx.Head.Nonce
	if m.items[nonce] == nil {
		heap.Push(m.index, nonce)
		atomic.AddInt64(&m.size, int64(tx.Size()))
	}
	m.items[nonce], m.cache = tx, nil
}

// DropedTxForLessNonce drop all txs whose nonce is less than the threshold.
// return the droped txs
func (m *txSortedMapByNonce) dropedTxForLessNonce(threshold uint64) []*txbasic.Transaction {
	var removed []*txbasic.Transaction

	// Pop off heap items until the threshold is reached
	for m.index.Len() > 0 && (*m.index)[0] < threshold {
		nonce := heap.Pop(m.index).(uint64)
		removed = append(removed, m.items[nonce])
		atomic.AddInt64(&m.size, -int64(m.items[nonce].Size()))
		delete(m.items, nonce)
	}
	// If we had a cached order, shift the front
	if m.cache != nil {
		m.cache = m.cache[len(removed):]
	}
	return removed
}

func (m *txSortedMapByNonce) Filter(filter func(*txbasic.Transaction) bool) []*txbasic.Transaction {
	removed := m.filter(filter)
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

func (m *txSortedMapByNonce) filter(filter func(*txbasic.Transaction) bool) []*txbasic.Transaction {
	var removed []*txbasic.Transaction
	for nonce, tx := range m.items {
		if filter(tx) {
			removed = append(removed, tx)
			atomic.AddInt64(&m.size, -int64(tx.Size()))
			delete(m.items, nonce)
		}
	}
	if len(removed) > 0 {
		m.cache = nil
	}
	return removed
}

func (m *txSortedMapByNonce) CapItems(threshold int) []*txbasic.Transaction {
	if len(m.items) <= threshold {
		return nil
	}
	var drops []*txbasic.Transaction

	sort.Sort(*m.index)
	for size := len(m.items); size > threshold; size-- {
		drops = append(drops, m.items[(*m.index)[size-1]])
		atomic.AddInt64(&m.size, -int64(m.items[(*m.index)[size-1]].Size()))
		delete(m.items, (*m.index)[size-1])
	}
	*m.index = (*m.index)[:threshold]
	heap.Init(m.index)

	if m.cache != nil {
		m.cache = m.cache[:len(m.cache)-len(drops)]
	}
	return drops
}

func (m *txSortedMapByNonce) Remove(nonce uint64) bool {
	tx, ok := m.items[nonce]
	if !ok {
		return false
	}
	for i := 0; i < m.index.Len(); i++ {
		if (*m.index)[i] == nonce {
			atomic.AddInt64(&m.size, -int64(tx.Size()))
			heap.Remove(m.index, i)
			break
		}
	}
	delete(m.items, nonce)
	m.cache = nil

	return true
}

func (m *txSortedMapByNonce) Ready(start uint64) []*txbasic.Transaction {
	if m.index.Len() == 0 || (*m.index)[0] <= start {
		return nil
	}
	var ready []*txbasic.Transaction
	for next := (*m.index)[0]; m.index.Len() > 0 && (*m.index)[0] == next; next++ {
		ready = append(ready, m.items[next])
		atomic.AddInt64(&m.size, -int64(m.items[next].Size()))
		delete(m.items, next)
		heap.Pop(m.index)
	}
	m.cache = nil

	return ready
}

func (m *txSortedMapByNonce) Len() int {
	return len(m.items)
}
func (m *txSortedMapByNonce) Size() int64 {
	return m.size
}

func (m *txSortedMapByNonce) flatten() []*txbasic.Transaction {
	if m.cache == nil {
		m.cache = make([]*txbasic.Transaction, 0, len(m.items))
		for _, tx := range m.items {
			m.cache = append(m.cache, tx)
		}
		sort.Sort(TxByNonce(m.cache))
	}
	return m.cache
}

func (m *txSortedMapByNonce) Flatten() []*txbasic.Transaction {
	cache := m.flatten()
	txs := make([]*txbasic.Transaction, len(cache))
	copy(txs, cache)
	return txs
}

func (m *txSortedMapByNonce) LastElement() *txbasic.Transaction {
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
func (l *txCoreList) WhetherSameNonce(tx *txbasic.Transaction) bool {
	l.lock.RLock()
	defer l.lock.RUnlock()
	Nonce := tx.Head.Nonce
	return l.txs.Get(Nonce) != nil
}

func (l *txCoreList) txCoreAdd(tx *txbasic.Transaction) (bool, *txbasic.Transaction) {
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

func (l *txCoreList) RemovedTxForLessNonce(threshold uint64) []*txbasic.Transaction {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.txs.dropedTxForLessNonce(threshold)
}

// CapLimitTxs places a limit on the number of items, returning all transactions
// exceeding that limit.
func (l *txCoreList) CapLimitTxs(threshold int) []*txbasic.Transaction {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.txs.CapItems(threshold)
}

func (l *txCoreList) Remove(tx *txbasic.Transaction) (bool, []*txbasic.Transaction) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	Nonce := tx.Head.Nonce
	nonce := Nonce
	if removed := l.txs.Remove(nonce); !removed {
		return false, nil
	}

	// Filter out non-executable transactions
	if l.strict {
		return true, l.txs.Filter(func(tx *txbasic.Transaction) bool {
			Noncei := tx.Head.Nonce
			return Noncei > nonce
		})
	}
	return true, nil
}

func (l *txCoreList) Ready(start uint64) []*txbasic.Transaction {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.txs.Ready(start)
}

func (l *txCoreList) Len() int {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.txs.Len()
}
func (l *txCoreList) Size() int64 {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.txs.Size()
}

func (l *txCoreList) Empty() bool {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.Len() == 0
}

func (l *txCoreList) Flatten() []*txbasic.Transaction {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.txs.Flatten()
}

// LastElementWithHighestNonce returns tx with the highest nonce
func (l *txCoreList) LastElementWithHighestNonce() *txbasic.Transaction {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.txs.LastElement()
}

type pendingTxs struct {
	Mu                sync.RWMutex
	mapAddrTxCoreList map[tpcrtypes.Address]*txCoreList
}

func newPendingTxs() *pendingTxs {
	addtxs := &pendingTxs{
		mapAddrTxCoreList: make(map[tpcrtypes.Address]*txCoreList, 0),
	}
	return addtxs
}

type pendingsMap struct {
	Mu      sync.RWMutex
	pending map[txbasic.TransactionCategory]*pendingTxs
}

func newPendingsMap() *pendingsMap {
	pendings := &pendingsMap{
		pending: make(map[txbasic.TransactionCategory]*pendingTxs, 0),
	}
	return pendings
}
func (pendingmap *pendingsMap) removeAll(category txbasic.TransactionCategory) {
	pendingmap.Mu.Lock()
	defer pendingmap.Mu.Unlock()
	delete(pendingmap.pending, category)
}
func (pendingmap *pendingsMap) ifEmptyDropCategory(category txbasic.TransactionCategory) {
	pendingmap.Mu.Lock()
	defer pendingmap.Mu.Unlock()
	if len(pendingmap.pending) == 0 {
		delete(pendingmap.pending, category)
	}
}
func (pendingmap *pendingsMap) getAllCommitTxs() []*txbasic.Transaction {
	var txs = make([]*txbasic.Transaction, 0)
	for category, _ := range pendingmap.pending {
		pendingtxs := pendingmap.getPendingTxsByCategory(category)
		for _, txList := range pendingtxs.mapAddrTxCoreList {
			for _, tx := range txList.txs.cache {
				txs = append(txs, tx)
			}
		}
	}
	return txs
}

func (pendingmap *pendingsMap) getTxsByCategory(category txbasic.TransactionCategory) []*txbasic.Transaction {
	pending := make([]*txbasic.Transaction, 0)
	pendingcat := pendingmap.getPendingTxsByCategory(category)
	if pendingcat == nil {
		return nil
	}
	pendingcat.Mu.RLock()
	defer pendingcat.Mu.RUnlock()
	for _, list := range pendingcat.mapAddrTxCoreList {
		txs := list.Flatten()
		if len(txs) > 0 {
			pending = append(pending, txs...)
		}
	}
	return pending
}
func (pendingmap *pendingsMap) getPendingTxsByCategory(category txbasic.TransactionCategory) *pendingTxs {
	pendingmap.Mu.RLock()
	defer pendingmap.Mu.RUnlock()
	return pendingmap.pending[category]
}
func (pendingmap *pendingsMap) demoteUnexecutablesByCategory(category txbasic.TransactionCategory,
	f1 func(tpcrtypes.Address) uint64,
	f2 func(transactionCategory txbasic.TransactionCategory, txId txbasic.TxID),
	f3 func(hash txbasic.TxID, tx *txbasic.Transaction)) {

	pendingcat := pendingmap.getPendingTxsByCategory(category)
	if pendingcat == nil {
		return
	}
	pendingcat.Mu.Lock()
	defer pendingcat.Mu.Unlock()
	for addr, list := range pendingcat.mapAddrTxCoreList {
		nonce := f1(addr)
		olds := list.RemovedTxForLessNonce(nonce)
		for _, tx := range olds {
			txId, _ := tx.TxID()
			f2(category, txId)
		}
		if list.Len() > 0 && list.txs.Get(nonce) == nil {
			gaped := list.CapLimitTxs(0)
			for _, tx := range gaped {
				hash, _ := tx.TxID()
				f3(hash, tx)
			}

		}
		if list.Empty() {
			delete(pendingcat.mapAddrTxCoreList, addr)
		}

	}
}

func (pendingmap *pendingsMap) getAddrTxsByCategory(category txbasic.TransactionCategory) map[tpcrtypes.Address][]*txbasic.Transaction {

	pending := make(map[tpcrtypes.Address][]*txbasic.Transaction)
	for addr, list := range pendingmap.getPendingTxsByCategory(category).mapAddrTxCoreList {
		txs := list.Flatten()
		if len(txs) > 0 {
			pending[addr] = txs
		}
	}
	return pending
}
func (pendingmap *pendingsMap) getAddrTxListOfCategory(category txbasic.TransactionCategory) map[tpcrtypes.Address]*txCoreList {

	pendingcat := pendingmap.getPendingTxsByCategory(category)
	if pendingcat == nil {
		return nil
	}
	pendingcat.Mu.RLock()
	defer pendingcat.Mu.RUnlock()
	return pendingcat.mapAddrTxCoreList
}

func (pendingmap *pendingsMap) noncesForAddrTxListOfCategory(category txbasic.TransactionCategory) {

	pendingcat := pendingmap.getPendingTxsByCategory(category)
	if pendingcat == nil {
		return
	}
	pendingcat.Mu.Lock()
	defer pendingcat.Mu.Unlock()
	if maptxlist := pendingmap.getPendingTxsByCategory(category).mapAddrTxCoreList; maptxlist != nil {
		nonces := make(map[tpcrtypes.Address]uint64, len(maptxlist))
		for addr, list := range maptxlist {
			highestPending := list.LastElementWithHighestNonce()
			Noncei := highestPending.Head.Nonce
			nonces[addr] = Noncei + 1
		}
	}
}

func (pendingmap *pendingsMap) getCommitTxsCategory(category txbasic.TransactionCategory) []*txbasic.Transaction {

	pendingcat := pendingmap.getPendingTxsByCategory(category)
	if pendingcat == nil {
		return nil
	}
	pendingcat.Mu.RLock()
	defer pendingcat.Mu.RUnlock()
	var txs = make([]*txbasic.Transaction, 0)
	maptxlist := pendingcat.mapAddrTxCoreList
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

func (pendingmap *pendingsMap) PendingSizeByCategory(category txbasic.TransactionCategory) uint64 {

	pendingcat := pendingmap.getPendingTxsByCategory(category)
	if pendingcat == nil {
		return 0
	}
	pendingcat.Mu.Lock()
	defer pendingcat.Mu.Unlock()
	pending := uint64(0)
	for _, list := range pendingcat.mapAddrTxCoreList {
		pending += uint64(list.Size())
	}
	return pending
}
func (pendingmap *pendingsMap) truncatePendingByCategoryFun2(category txbasic.TransactionCategory, MaxSizeOfEachPendingAccount uint64) map[tpcrtypes.Address]int64 {

	pendingcat := pendingmap.getPendingTxsByCategory(category)
	if pendingcat == nil {
		return nil
	}
	var greyAccounts map[tpcrtypes.Address]int64
	for addr, list := range pendingmap.pending[category].mapAddrTxCoreList {
		// Only evict transactions from high rollers
		if uint64(list.Size()) > MaxSizeOfEachPendingAccount {
			greyAccounts[addr] = list.Size()
		}
	}
	return greyAccounts
}
func (pendingmap *pendingsMap) truncatePendingByCategoryFun3(f31 func(category txbasic.TransactionCategory, txId txbasic.TxID),
	category txbasic.TransactionCategory, addr tpcrtypes.Address) {

	pendingcat := pendingmap.getPendingTxsByCategory(category)
	if pendingcat == nil {
		return
	}
	pendingcat.Mu.Lock()
	defer pendingcat.Mu.Unlock()
	if list := pendingcat.mapAddrTxCoreList[addr]; list != nil {
		caps := list.CapLimitTxs(list.Len() - 1)
		for _, tx := range caps {
			txId, _ := tx.TxID()
			f31(category, txId)
		}
	}
	return
}

func (pendingmap *pendingsMap) getTxsByAddrOfCategory(category txbasic.TransactionCategory, addr tpcrtypes.Address) []*txbasic.Transaction {

	pendingcat := pendingmap.getPendingTxsByCategory(category)
	if pendingcat == nil {
		return nil
	}
	pendingcat.Mu.RLock()
	defer pendingcat.Mu.RUnlock()
	if txlist := pendingcat.mapAddrTxCoreList[addr]; txlist != nil {
		return txlist.Flatten()
	}
	return nil
}
func (pendingmap *pendingsMap) getTxListByAddrOfCategory(category txbasic.TransactionCategory, addr tpcrtypes.Address) *txCoreList {

	pendingcat := pendingmap.getPendingTxsByCategory(category)
	if pendingcat == nil {
		return nil
	}
	pendingcat.Mu.RLock()
	defer pendingcat.Mu.RUnlock()
	if txlist := pendingcat.mapAddrTxCoreList[addr]; txlist != nil {
		return txlist
	}
	return nil
}
func (pendingmap *pendingsMap) addTxToTxListByAddrOfCategory(
	category txbasic.TransactionCategory, addr tpcrtypes.Address, tx *txbasic.Transaction) (bool, *txbasic.Transaction) {

	pendingcat := pendingmap.getPendingTxsByCategory(category)
	if pendingcat == nil {
		return false, nil
	}
	pendingcat.Mu.Lock()
	defer pendingcat.Mu.Unlock()

	if txlist := pendingcat.mapAddrTxCoreList[addr]; txlist != nil {
		return txlist.txCoreAdd(tx)
	}
	return false, nil
}

func (pendingmap *pendingsMap) getTxListRemoveByAddrOfCategory(
	f1 func(string2 txbasic.TxID, transaction *txbasic.Transaction),
	tx *txbasic.Transaction,
	category txbasic.TransactionCategory, addr tpcrtypes.Address) {

	pendingcat := pendingmap.getPendingTxsByCategory(category)
	if pendingcat == nil {
		return
	}
	pendingcat.Mu.Lock()
	defer pendingcat.Mu.Unlock()
	if list := pendingcat.mapAddrTxCoreList[addr]; list != nil {
		removed, invalids := list.Remove(tx)
		if !removed {
			return
		}

		if list.Empty() {
			delete(pendingcat.mapAddrTxCoreList, addr)
		}

		// Postpone any invalidated transactions
		for _, txi := range invalids {
			txId, _ := txi.TxID()
			f1(txId, txi)
		}
	}
}

func (pendingmap *pendingsMap) replaceTxOfAddrOfCategory(category txbasic.TransactionCategory, from tpcrtypes.Address,
	txId txbasic.TxID, tx *txbasic.Transaction, isLocal bool,
	f1 func(category txbasic.TransactionCategory, txId txbasic.TxID),
	f3 func(category txbasic.TransactionCategory, tx *txbasic.Transaction, isLocal bool),
	f5 func(txId txbasic.TxID)) (bool, error) {
	var old *txbasic.Transaction
	pendingcat := pendingmap.getPendingTxsByCategory(category)
	if pendingcat == nil {
		pendingcat = newPendingTxs()
	}
	pendingcat.Mu.Lock()
	defer pendingcat.Mu.Unlock()
	if list := pendingcat.mapAddrTxCoreList[from]; list != nil && list.WhetherSameNonce(tx) {
		inserted, old := list.txCoreAdd(tx)
		if !inserted {
			return false, ErrReplaceUnderpriced
		}
		if old != nil {
			f1(category, txId)
		}
		f3(category, tx, isLocal)
		f5(txId)
	}
	return old != nil, nil
}
func (pendingmap *pendingsMap) setTxListOfCategory(category txbasic.TransactionCategory, addr tpcrtypes.Address, txs *txCoreList) {

	if pendingcat := pendingmap.getPendingTxsByCategory(category); pendingcat != nil {
		pendingcat.Mu.Lock()
		defer pendingcat.Mu.Unlock()
		pendingcat.mapAddrTxCoreList[addr] = txs
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
	queue map[txbasic.TransactionCategory]*queueTxs
}

func newQueuesMap() *queuesMap {
	queues := &queuesMap{
		queue: make(map[txbasic.TransactionCategory]*queueTxs, 0),
	}
	return queues
}
func (queuemap *queuesMap) removeAll(category txbasic.TransactionCategory) {
	queuemap.Mu.Lock()
	defer queuemap.Mu.Unlock()
	delete(queuemap.queue, category)
}
func (queuemap *queuesMap) ifEmptyDropCategory(category txbasic.TransactionCategory) {
	queuemap.Mu.Lock()
	defer queuemap.Mu.Unlock()
	if len(queuemap.queue) == 0 {
		delete(queuemap.queue, category)
	}
}

func (queuemap *queuesMap) getTxsByCategory(category txbasic.TransactionCategory) []*txbasic.Transaction {

	queue := make([]*txbasic.Transaction, 0)
	queuecat := queuemap.getQueueTxsByCategory(category)
	if queuecat == nil {
		return nil
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
func (queuemap *queuesMap) getAll() map[txbasic.TransactionCategory]*queueTxs {
	queuemap.Mu.RLock()
	defer queuemap.Mu.RUnlock()
	return queuemap.queue
}
func (queuemap *queuesMap) getAllAddress() []tpcrtypes.Address {
	addrs := make([]tpcrtypes.Address, 0)
	for _, queuetxs := range queuemap.queue {
		queuetxs.Mu.RLock()
		for addr, _ := range queuetxs.mapAddrTxCoreList {
			addrs = append(addrs, addr)
		}
		queuetxs.Mu.RUnlock()

	}
	return addrs
}

func (queuemap *queuesMap) getQueueTxsByCategory(category txbasic.TransactionCategory) *queueTxs {
	queuemap.Mu.RLock()
	defer queuemap.Mu.RUnlock()
	return queuemap.queue[category]
}

func (queuemap *queuesMap) getTxListByAddrOfCategory(category txbasic.TransactionCategory, addr tpcrtypes.Address) *txCoreList {

	if queuecat := queuemap.getQueueTxsByCategory(category); queuecat != nil {
		queuecat.Mu.RLock()
		defer queuecat.Mu.RUnlock()
		return queuecat.mapAddrTxCoreList[addr]
	}
	return nil
}
func (queuemap *queuesMap) getTxsByAddrOfCategory(category txbasic.TransactionCategory, addr tpcrtypes.Address) []*txbasic.Transaction {

	if queuecat := queuemap.getQueueTxsByCategory(category); queuecat != nil {
		queuecat.Mu.RLock()
		defer queuecat.Mu.RUnlock()
		return queuecat.mapAddrTxCoreList[addr].Flatten()
	}
	return nil
}
func (queuemap *queuesMap) getTxByNonceFromTxlistByAddrOfCategory(category txbasic.TransactionCategory, addr tpcrtypes.Address, Nonce uint64) *txbasic.Transaction {

	if queuecat := queuemap.getQueueTxsByCategory(category); queuecat != nil {
		queuecat.Mu.RLock()
		defer queuecat.Mu.RUnlock()
		if txlist := queuecat.mapAddrTxCoreList[addr]; txlist != nil {
			return txlist.txs.Get(Nonce)
		}
	}
	return nil
}

func (queuemap *queuesMap) getLenTxsByAddrOfCategory(category txbasic.TransactionCategory, addr tpcrtypes.Address) int {

	queuecat := queuemap.getQueueTxsByCategory(category)
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
	category txbasic.TransactionCategory, addr tpcrtypes.Address, tx *txbasic.Transaction) {

	queuecat := queuemap.getQueueTxsByCategory(category)
	if queuecat == nil {
		return
	}
	queuemap.queue[category].Mu.Lock()
	defer queuemap.queue[category].Mu.Unlock()

	if txlist := queuemap.queue[category].mapAddrTxCoreList[addr]; txlist != nil {
		txlist.Remove(tx)
	}
	return
}

func (queuemap *queuesMap) replaceExecutablesDropOverLimit(
	f2 func(category txbasic.TransactionCategory, txId txbasic.TxID),
	MaxSizeOfEachQueueAccount uint64,
	category txbasic.TransactionCategory, addr tpcrtypes.Address) {

	queuecat := queuemap.getQueueTxsByCategory(category)
	if queuecat == nil {
		return
	}
	queuecat.Mu.Lock()
	defer queuecat.Mu.Unlock()
	txlist := queuemap.queue[category].mapAddrTxCoreList[addr]
	if txlist == nil {
		return
	}
	var caps []*txbasic.Transaction
	caps = txlist.CapLimitTxs(int(MaxSizeOfEachQueueAccount))
	for _, tx := range caps {
		txId, _ := tx.TxID()
		f2(category, txId)
	}

	return
}

func (queuemap *queuesMap) replaceExecutablesTurnTx(f0 func(addr tpcrtypes.Address) uint64,
	f1 func(category txbasic.TransactionCategory, address tpcrtypes.Address, tx *txbasic.Transaction) (bool, *txbasic.Transaction),
	f2 func(category txbasic.TransactionCategory, txId txbasic.TxID, tx *txbasic.Transaction),
	f4 func(category txbasic.TransactionCategory, txId txbasic.TxID, tx *txbasic.Transaction),
	f5 func(txId txbasic.TxID),
	turnedTxs []*txbasic.Transaction,
	category txbasic.TransactionCategory, addr tpcrtypes.Address) int {

	queuecat := queuemap.getQueueTxsByCategory(category)
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
		txId, _ := tx.TxID()
		// Try to insert the transaction into the pending queue
		f := func(addr tpcrtypes.Address, txId txbasic.TxID, tx *txbasic.Transaction) bool {
			inserted, old := f1(category, addr, tx)
			if !inserted {
				// An older transaction was existed, discard this
				f2(category, txId, old)
				return false
			}

			if old != nil {
				oldkey, _ := old.TxID()
				f4(category, oldkey, old)
			}
			// Successful replace tx, bump the ActivationInterval
			f5(txId)
			return true
		}

		if f(addr, txId, tx) {
			turnedTxs = append(turnedTxs, tx)
		}
	}
	return len(turnedTxs)
}

func (queuemap *queuesMap) replaceExecutablesDeleteEmpty(category txbasic.TransactionCategory, addr tpcrtypes.Address) {

	queuecat := queuemap.getQueueTxsByCategory(category)
	if queuecat == nil {
		return
	}
	queuemap.queue[category].Mu.Lock()
	defer queuemap.queue[category].Mu.Unlock()

	if txlist := queuemap.queue[category].mapAddrTxCoreList[addr]; txlist == nil {
		return
	} else if txlist.Empty() {
		delete(queuemap.queue[category].mapAddrTxCoreList, addr)
	}
	return

}

func (queuemap *queuesMap) replaceExecutablesDropTooOld(category txbasic.TransactionCategory, addr tpcrtypes.Address,
	f1 func(address tpcrtypes.Address) uint64,
	f2 func(transactionCategory txbasic.TransactionCategory, txId txbasic.TxID)) {

	queuecat := queuemap.getQueueTxsByCategory(category)
	if queuecat == nil {
		return
	}
	queuemap.queue[category].Mu.Lock()
	defer queuemap.queue[category].Mu.Unlock()

	list := queuemap.queue[category].mapAddrTxCoreList[addr]
	if list == nil {
		return
	}
	// Drop all transactions that are deemed too old (low nonce)
	DropsLessNonce := list.RemovedTxForLessNonce(f1(addr))

	for _, tx := range DropsLessNonce {
		txID, _ := tx.TxID()
		f2(category, txID)

	}
	return
}

func (queuemap *queuesMap) getTxListRemoveFutureByAddrOfCategory(tx *txbasic.Transaction, f1 func(string2 txbasic.TxID), key txbasic.TxID, category txbasic.TransactionCategory, addr tpcrtypes.Address) {

	queuecat := queuemap.getQueueTxsByCategory(category)
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

func (queuemap *queuesMap) getAddrTxsByCategory(category txbasic.TransactionCategory) map[tpcrtypes.Address][]*txbasic.Transaction {

	queue := make(map[tpcrtypes.Address][]*txbasic.Transaction)
	for addr, list := range queuemap.getQueueTxsByCategory(category).mapAddrTxCoreList {
		txs := list.Flatten()
		if len(txs) > 0 {
			queue[addr] = txs
		}
	}
	return queue
}
func (queuemap *queuesMap) getAddrTxListOfCategory(category txbasic.TransactionCategory) map[tpcrtypes.Address]*txCoreList {

	if queuecat := queuemap.getQueueTxsByCategory(category); queuecat != nil {
		queuecat.Mu.RLock()
		defer queuecat.Mu.RUnlock()
		if maptxlist := queuecat.mapAddrTxCoreList; maptxlist != nil {
			return maptxlist
		}
	}
	return nil
}

func (queuemap *queuesMap) replaceAddrTxList() []tpcrtypes.Address {

	replaceAddrs := make([]tpcrtypes.Address, 0)
	for category, _ := range queuemap.queue {
		queuecat := queuemap.getQueueTxsByCategory(category)
		if queuecat == nil {
			return nil
		}
		queuecat.Mu.Lock()
		defer queuecat.Mu.Unlock()
		maptxlist := queuemap.getQueueTxsByCategory(category).mapAddrTxCoreList
		if maptxlist == nil {
			return nil
		}
		for addr, _ := range maptxlist {
			replaceAddrs = append(replaceAddrs, addr)
		}
	}
	return replaceAddrs
}

func (queuemap *queuesMap) getSizeOfCategory(category txbasic.TransactionCategory) int64 {
	queuecat := queuemap.getQueueTxsByCategory(category)
	if queuecat == nil {
		return 0
	}
	queuecat.Mu.RLock()
	defer queuecat.Mu.RUnlock()
	queue := int64(0)
	maptxlist := queuecat.mapAddrTxCoreList
	if maptxlist == nil {
		return 0
	}
	for _, list := range maptxlist {
		queue += list.Size()
	}
	return queue
}

func (queuemap *queuesMap) removeTxsForTruncateQueue(category txbasic.TransactionCategory,
	f2 func(string2 txbasic.TxID) time.Time,
	f3 func(transactionCategory txbasic.TransactionCategory, txHash txbasic.TxID) *txbasic.Transaction,
	f4 func(category txbasic.TransactionCategory, txId txbasic.TxID),
	f5 func(f51 func(txId txbasic.TxID, tx *txbasic.Transaction), tx *txbasic.Transaction, category txbasic.TransactionCategory, addr tpcrtypes.Address),
	f511 func(category txbasic.TransactionCategory, hash txbasic.TxID),
	f512 func(category txbasic.TransactionCategory, txId txbasic.TxID) *txbasic.Transaction,
	f513 func(txId txbasic.TxID),
	f514 func(txId txbasic.TxID, category txbasic.TransactionCategory),
	f6 func(txId txbasic.TxID),
	queueSize int64, MaxSizeOfQueue uint64) {

	queuecat := queuemap.getQueueTxsByCategory(category)
	if queuecat == nil {
		return
	}
	queuecat.Mu.Lock()
	defer queuecat.Mu.Unlock()
	txs := make(txsByActivationInterval, 0, len(queuemap.queue[category].mapAddrTxCoreList))
	for addr := range queuemap.queue[category].mapAddrTxCoreList {
		list := queuemap.queue[category].mapAddrTxCoreList[addr].Flatten()
		for _, tx := range list {
			txId, _ := tx.TxID()
			time := f2(txId)
			txs = append(txs, txByActivationInterval{txId, time, tx.Size()})
		}

	}
	sort.Sort(txs)
	for size := uint64(queueSize); size > MaxSizeOfQueue && len(txs) > 0; {
		txH := txs[len(txs)-1]
		txs = txs[:len(txs)-1]
		txId := txH.txId
		if tx := f3(category, txId); tx != nil {
			addr := tpcrtypes.Address(tx.Head.FromAddr)
			// Remove it from the list of known transactions
			f4(category, txId)

			f51 := func(txId txbasic.TxID, tx *txbasic.Transaction) {
				from := tpcrtypes.Address(tx.Head.FromAddr)
				inserted, old := queuecat.mapAddrTxCoreList[from].txCoreAdd(tx)
				if !inserted {
					// An older transaction was existed
					return
				}
				if old != nil {
					oldTxId, _ := old.TxID()
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
		size -= uint64(txH.size)
		continue
	}
}

func (queuemap *queuesMap) removeTxForLifeTime(category txbasic.TransactionCategory, expiredPolicy txpooli.TxExpiredPolicy,
	f1 func(string2 txbasic.TxID) time.Duration, duration2 time.Duration, f2 func(string2 txbasic.TxID),
	f3 func(string2 txbasic.TxID) uint64, dltHeight uint64) {
	queuecat := queuemap.getQueueTxsByCategory(category)
	if queuecat == nil {
		return
	}
	queuecat.Mu.Lock()
	defer queuecat.Mu.Unlock()
	for _, txlist := range queuecat.mapAddrTxCoreList {
		list := txlist.txs.Flatten()
		for _, tx := range list {
			txId, _ := tx.TxID()
			switch expiredPolicy {
			case txpooli.TxExpiredTime:
				if f1(txId) > duration2 {
					f2(txId)
				}
			case txpooli.TxExpiredHeight:
				if f3(txId) > dltHeight {
					f2(txId)
				}
			case txpooli.TxExpiredTimeAndHeight:
				if f1(txId) > duration2 && f3(txId) > dltHeight {
					f2(txId)
				}
			case txpooli.TxExpiredTimeOrHeight:
				if f1(txId) > duration2 || f3(txId) > dltHeight {
					f2(txId)
				}
			}
		}
	}
}

func (queuemap *queuesMap) republicTx(category txbasic.TransactionCategory, policy TxRepublicPolicy,
	f1 func(string2 txbasic.TxID) time.Duration, time2 time.Duration, f2 func(tx *txbasic.Transaction),
	f3 func(string2 txbasic.TxID) uint64, diffHegiht uint64) {

	queuecat := queuemap.getQueueTxsByCategory(category)
	if queuecat == nil {
		return
	}
	queuecat.Mu.Lock()
	defer queuecat.Mu.Unlock()
	for _, txlist := range queuecat.mapAddrTxCoreList {
		list := txlist.txs.Flatten()
		for _, tx := range list {
			txId, _ := tx.TxID()
			switch policy {
			case TxRepublicTime:
				if f1(txId) > time2 {
					f2(tx)
				}
			case TxRepublicHeight:
				if f3(txId) > diffHegiht {
					f2(tx)
				}
			case TxRepublicTimeAndHeight:
				if f1(txId) > time2 && f3(txId) > diffHegiht {
					f2(tx)
				}
			case TxRepublicTimeOrHeight:
				if f1(txId) > time2 || f3(txId) > diffHegiht {
					f2(tx)
				}
			}

		}
	}
}

func (queuemap *queuesMap) addTxByKeyOfCategory(
	f1 func(category txbasic.TransactionCategory, key txbasic.TxID),
	f2 func(category txbasic.TransactionCategory, key txbasic.TxID) *txbasic.Transaction,
	f3 func(string2 txbasic.TxID),
	f4 func(category txbasic.TransactionCategory, transaction *txbasic.Transaction, local bool),
	f5 func(string2 txbasic.TxID, category txbasic.TransactionCategory, local bool),
	key txbasic.TxID, tx *txbasic.Transaction, local bool, addAll bool) (bool, error) {

	category := txbasic.TransactionCategory(tx.Head.Category)
	if queuecat := queuemap.getQueueTxsByCategory(category); queuecat != nil {
		queuecat.Mu.Lock()
		defer queuecat.Mu.Unlock()
		from := tpcrtypes.Address(tx.Head.FromAddr)

		if queuecat.mapAddrTxCoreList[from] == nil {
			queuecat.mapAddrTxCoreList[from] = newCoreList(false)
		}
		inserted, old := queuecat.mapAddrTxCoreList[from].txCoreAdd(tx)
		if !inserted {
			// An older transaction has better price
			return false, ErrReplaceUnderpriced
		}
		if old != nil {
			oldTxId, _ := old.TxID()
			f1(category, oldTxId)
		}
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
	Mu   sync.RWMutex
	all  map[txbasic.TransactionCategory]*txForLookup
	cnt  int64
	size int64
}

func newAllTxsLookupMap() *allTxsLookupMap {
	allMap := &allTxsLookupMap{
		all: make(map[txbasic.TransactionCategory]*txForLookup, 0),
	}
	return allMap
}
func (alltxsmap *allTxsLookupMap) removeAll(category txbasic.TransactionCategory) {
	alltxsmap.Mu.Lock()
	defer alltxsmap.Mu.Unlock()
	delete(alltxsmap.all, category)
}

func (alltxsmap *allTxsLookupMap) getAll() map[txbasic.TransactionCategory]*txForLookup {
	return alltxsmap.all
}

func (alltxsmap *allTxsLookupMap) getAllCount() int64 {
	var cnt int64
	for category, _ := range alltxsmap.getAll() {
		cnt += alltxsmap.getTxCntFromAllTxsLookupByCategory(category)
	}
	return cnt
}
func (alltxsmap *allTxsLookupMap) getAllSize() int64 {
	var size int64
	for category, _ := range alltxsmap.getAll() {
		size += alltxsmap.getSizeFromAllTxsLookupByCategory(category)
	}
	return size
}

func (alltxsmap *allTxsLookupMap) getAllTxsByCategory(category txbasic.TransactionCategory) []*txbasic.Transaction {
	txs := make([]*txbasic.Transaction, 0)
	if lookuptxs := alltxsmap.getAllTxsLookupByCategory(category); lookuptxs != nil {
		lookuptxs.lock.RLock()
		defer lookuptxs.lock.RUnlock()
		for _, tx := range lookuptxs.locals {
			txs = append(txs, tx)
		}
		for _, tx := range lookuptxs.remotes {
			txs = append(txs, tx)
		}
	}
	return txs
}

func (alltxsmap *allTxsLookupMap) getAllTxsLookupByCategory(category txbasic.TransactionCategory) *txForLookup {
	return alltxsmap.all[category]
}

func (alltxsmap *allTxsLookupMap) getLocalCountByCategory(category txbasic.TransactionCategory) int {
	if lookuptxs := alltxsmap.getAllTxsLookupByCategory(category); lookuptxs != nil {
		lookuptxs.lock.RLock()
		defer lookuptxs.lock.RUnlock()
		return len(lookuptxs.locals)
	}
	return 0
}
func (alltxsmap *allTxsLookupMap) getLocalTxsByCategory(category txbasic.TransactionCategory) []*txbasic.Transaction {
	txs := make([]*txbasic.Transaction, 0)
	if lookuptxs := alltxsmap.getAllTxsLookupByCategory(category); lookuptxs != nil {
		lookuptxs.lock.RLock()
		defer lookuptxs.lock.RUnlock()
		for _, tx := range lookuptxs.locals {
			txs = append(txs, tx)
		}
		return txs
	}
	return nil
}

func (alltxsmap *allTxsLookupMap) getRemoteCountByCategory(category txbasic.TransactionCategory) int {

	if lookuptxs := alltxsmap.getAllTxsLookupByCategory(category); lookuptxs != nil {
		lookuptxs.lock.RLock()
		defer lookuptxs.lock.RUnlock()

		return len(lookuptxs.remotes)
	}
	return 0
}

func (alltxsmap *allTxsLookupMap) getSizeFromAllTxsLookupByCategory(category txbasic.TransactionCategory) int64 {

	if txlookup := alltxsmap.getAllTxsLookupByCategory(category); txlookup != nil {
		txlookup.lock.RLock()
		defer txlookup.lock.RUnlock()
		return txlookup.sizes
	}
	return 0
}
func (alltxsmap *allTxsLookupMap) getTxCntFromAllTxsLookupByCategory(category txbasic.TransactionCategory) int64 {

	if txlookup := alltxsmap.getAllTxsLookupByCategory(category); txlookup != nil {
		txlookup.lock.RLock()
		defer txlookup.lock.RUnlock()
		return txlookup.cnt
	}
	return 0
}

func (alltxsmap *allTxsLookupMap) getCountFromAllTxsLookupByCategory(category txbasic.TransactionCategory) int {

	if txlookup := alltxsmap.getAllTxsLookupByCategory(category); txlookup != nil {
		txlookup.lock.RLock()
		defer txlookup.lock.RUnlock()
		return len(txlookup.locals) + len(txlookup.remotes)
	}
	return 0
}

func (alltxsmap *allTxsLookupMap) getLocalKeyTxFromAllTxsLookupByCategory(category txbasic.TransactionCategory) map[txbasic.TxID]*txbasic.Transaction {

	if txlookup := alltxsmap.getAllTxsLookupByCategory(category); txlookup != nil {
		txlookup.lock.RLock()
		defer txlookup.lock.RUnlock()
		return txlookup.locals
	}
	return nil
}
func (alltxsmap *allTxsLookupMap) getTxFromKeyFromAllTxsLookupByCategory(category txbasic.TransactionCategory, txHash txbasic.TxID) *txbasic.Transaction {

	if txlookup := alltxsmap.getAllTxsLookupByCategory(category); txlookup != nil {

		txlookup.lock.RLock()
		defer txlookup.lock.RUnlock()

		if tx := txlookup.locals[txHash]; tx != nil {
			return tx
		}
		return txlookup.remotes[txHash]
	}
	return nil
}

func (alltxsmap *allTxsLookupMap) addTxToAllTxsLookupByCategory(category txbasic.TransactionCategory,
	tx *txbasic.Transaction, isLocal bool) (bool, error) {

	catAllMap := alltxsmap.getAllTxsLookupByCategory(category)
	if catAllMap == nil {
		return false, ErrAllTxListOfCategoryIsNil
	}

	catAllMap.lock.Lock()
	defer catAllMap.lock.Unlock()
	atomic.AddInt64(&catAllMap.sizes, int64(tx.Size()))
	atomic.AddInt64(&catAllMap.cnt, 1)
	if txId, err := tx.TxID(); err == nil {
		if isLocal {
			catAllMap.locals[txId] = tx
		} else {
			catAllMap.remotes[txId] = tx
		}
	}
	return true, nil
}

func (alltxsmap *allTxsLookupMap) removeTxHashFromAllTxsLookupByCategory(category txbasic.TransactionCategory, txhash txbasic.TxID) (isRemoved bool, txRm *txbasic.Transaction) {

	if txlookup := alltxsmap.getAllTxsLookupByCategory(category); txlookup != nil {
		txlookup.lock.Lock()
		defer txlookup.lock.Unlock()
		tx, ok := txlookup.locals[txhash]
		if !ok {
			tx, ok = txlookup.remotes[txhash]
		}
		if !ok {
			return false, nil
		}
		atomic.AddInt64(&txlookup.sizes, -int64(tx.Size()))
		atomic.AddInt64(&txlookup.cnt, -1)
		delete(txlookup.locals, txhash)
		delete(txlookup.remotes, txhash)
		return true, tx
	} else {
		return false, nil
	}
}

func (alltxsmap *allTxsLookupMap) getRemoteMapTxsLookupByCategory(category txbasic.TransactionCategory) map[txbasic.TxID]*txbasic.Transaction {

	if txlookup := alltxsmap.getAllTxsLookupByCategory(category); txlookup != nil {
		txlookup.lock.RLock()
		defer txlookup.lock.RUnlock()
		return txlookup.remotes
	}
	return nil
}

func (alltxsmap *allTxsLookupMap) ifEmptyDropCategory(category txbasic.TransactionCategory) {
	alltxsmap.Mu.Lock()
	defer alltxsmap.Mu.Unlock()
	if len(alltxsmap.all) == 0 {
		delete(alltxsmap.all, category)
	}
}

type activationInterval struct {
	Mu    sync.RWMutex
	activ map[txbasic.TxID]time.Time
}

func newActivationInterval() *activationInterval {
	activ := &activationInterval{
		activ: make(map[txbasic.TxID]time.Time),
	}
	return activ
}
func (activ *activationInterval) getAll() map[txbasic.TxID]time.Time {
	activ.Mu.Lock()
	defer activ.Mu.Unlock()
	return activ.activ
}
func (activ *activationInterval) getTxActivByKey(key txbasic.TxID) time.Time {
	activ.Mu.Lock()
	defer activ.Mu.Unlock()
	return activ.activ[key]
}
func (activ *activationInterval) setTxActiv(key txbasic.TxID, time time.Time) {
	activ.Mu.Lock()
	defer activ.Mu.Unlock()
	activ.activ[key] = time
	return
}
func (activ *activationInterval) removeTxActiv(key txbasic.TxID) {
	activ.Mu.Lock()
	defer activ.Mu.Unlock()
	delete(activ.activ, key)
	return
}

type HeightInterval struct {
	Mu sync.RWMutex
	HI map[txbasic.TxID]uint64
}

func newHeightInterval() *HeightInterval {
	hi := &HeightInterval{
		HI: make(map[txbasic.TxID]uint64),
	}
	return hi
}
func (hi *HeightInterval) getAll() map[txbasic.TxID]uint64 {
	hi.Mu.Lock()
	defer hi.Mu.Unlock()
	return hi.HI
}
func (hi *HeightInterval) getTxHeightByKey(key txbasic.TxID) uint64 {
	hi.Mu.Lock()
	defer hi.Mu.Unlock()
	return hi.HI[key]
}
func (hi *HeightInterval) setTxHeight(key txbasic.TxID, height uint64) {
	hi.Mu.Lock()
	defer hi.Mu.Unlock()
	hi.HI[key] = height
	return
}
func (hi *HeightInterval) removeTxHeight(key txbasic.TxID) {
	hi.Mu.Lock()
	defer hi.Mu.Unlock()
	delete(hi.HI, key)
	return
}

type txHashCategory struct {
	Mu              sync.RWMutex
	hashCategoryMap map[txbasic.TxID]txbasic.TransactionCategory
}

func newTxHashCategory() *txHashCategory {
	hashCat := &txHashCategory{
		hashCategoryMap: make(map[txbasic.TxID]txbasic.TransactionCategory),
	}
	return hashCat
}
func (hashCat *txHashCategory) getAll() map[txbasic.TxID]txbasic.TransactionCategory {
	hashCat.Mu.Lock()
	defer hashCat.Mu.Unlock()
	return hashCat.hashCategoryMap
}
func (hashCat *txHashCategory) getByHash(key txbasic.TxID) txbasic.TransactionCategory {
	hashCat.Mu.Lock()
	defer hashCat.Mu.Unlock()
	return hashCat.hashCategoryMap[key]
}
func (hashCat *txHashCategory) setHashCat(key txbasic.TxID, category txbasic.TransactionCategory) {
	hashCat.Mu.Lock()
	defer hashCat.Mu.Unlock()
	hashCat.hashCategoryMap[key] = category
}
func (hashCat *txHashCategory) removeHashCat(key txbasic.TxID) {
	hashCat.Mu.Lock()
	defer hashCat.Mu.Unlock()
	delete(hashCat.hashCategoryMap, key)
}

type txForLookup struct {
	sizes   int64
	cnt     int64
	lock    sync.RWMutex
	locals  map[txbasic.TxID]*txbasic.Transaction
	remotes map[txbasic.TxID]*txbasic.Transaction
}

func newTxForLookup() *txForLookup {
	return &txForLookup{
		locals:  make(map[txbasic.TxID]*txbasic.Transaction),
		remotes: make(map[txbasic.TxID]*txbasic.Transaction),
	}
}

func (t *txForLookup) RemoteCount() int {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return len(t.remotes)
}

type accountCache struct {
	mapAccount map[tpcrtypes.Address]struct{}
	cache      *[]tpcrtypes.Address
}

func newAccountCache(addrs ...tpcrtypes.Address) *accountCache {
	ac := &accountCache{
		mapAccount: make(map[tpcrtypes.Address]struct{}),
	}
	for _, addr := range addrs {
		ac.add(addr)
	}
	return ac
}

func (ac *accountCache) contains(addr tpcrtypes.Address) bool {
	_, exist := ac.mapAccount[addr]
	return exist
}

func (ac *accountCache) add(addr tpcrtypes.Address) {
	ac.mapAccount[addr] = struct{}{}
	ac.cache = nil
}

func (ac *accountCache) addTx(tx *txbasic.Transaction) {
	addr := tpcrtypes.Address(tx.Head.FromAddr)
	ac.add(addr)
}

// flatten returns the list of addresses within this set, also caching it for later
// reuse. The returned slice should not be changed!
func (ac *accountCache) flatten() []tpcrtypes.Address {
	if ac.cache == nil {
		accounts := make([]tpcrtypes.Address, 0, len(ac.mapAccount))
		for account := range ac.mapAccount {
			accounts = append(accounts, account)
		}
		ac.cache = &accounts
	}
	return *ac.cache
}

func (ac *accountCache) merge(other *accountCache) {
	for addr := range other.mapAccount {
		ac.mapAccount[addr] = struct{}{}
	}
	ac.cache = nil
}

// SizeAccountItem is an item  we manage in a priority queue for cnt.
type SizeAccountItem struct {
	accountAddr tpcrtypes.Address
	size        int64 // The priority of SizeAccountItem in the queue is counts of account in txPool.
	index       int   // The index of the SizeAccountItem item in the heap.
}

// A SizeAccountHeap implements heap.Interface and holds GreyAccCnt.
type SizeAccountHeap []*SizeAccountItem

func (pq SizeAccountHeap) Len() int { return len(pq) }

func (pq SizeAccountHeap) Less(i, j int) bool {
	return pq[i].size < pq[j].size
}

func (pq SizeAccountHeap) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *SizeAccountHeap) Push(x interface{}) {
	n := len(*pq)
	item := x.(*SizeAccountItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *SizeAccountHeap) Pop() interface{} {
	old := *pq
	n := len(old)
	GreyAccCnt := old[n-1]
	GreyAccCnt.index = -1 // for safety
	*pq = old[0 : n-1]
	return GreyAccCnt
}

// txByActivationInterval tagged with transaction's last activity timestamp.
type txByActivationInterval struct {
	txId               txbasic.TxID
	ActivationInterval time.Time
	size               int
}

type txsByActivationInterval []txByActivationInterval

func (t txsByActivationInterval) Len() int { return len(t) }
func (t txsByActivationInterval) Less(i, j int) bool {
	return t[i].ActivationInterval.Before(t[j].ActivationInterval)
}
func (t txsByActivationInterval) Swap(i, j int) { t[i], t[j] = t[j], t[i] }

type TxByNonce []*txbasic.Transaction

func (s TxByNonce) Len() int { return len(s) }
func (s TxByNonce) Less(i, j int) bool {
	return s[i].Head.Nonce < s[j].Head.Nonce
}
func (s TxByNonce) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type TxByPriceAndNonce []*txbasic.Transaction

func (pr TxByPriceAndNonce) Len() int { return len(pr) }
func (pr TxByPriceAndNonce) Less(i, j int) bool {
	switch pr.cmp(pr[i], pr[j]) {
	case -1:
		return true
	case 1:
		return false
	default:
		var Noncei, Noncej uint64
		Noncei = pr[i].Head.Nonce
		Noncej = pr[j].Head.Nonce
		return Noncei < Noncej
	}
}
func (pr TxByPriceAndNonce) cmp(a, b *txbasic.Transaction) int {
	if GasPrice(a) < GasPrice(b) {
		return -1
	} else if GasPrice(a) > GasPrice(b) {
		return 1
	} else {
		return 0
	}
}
func (pr TxByPriceAndNonce) Swap(i, j int) {
	pr[i], pr[j] = pr[j], pr[i]
}

type TxByPriceAndTime []*txbasic.Transaction

func (s TxByPriceAndTime) Len() int { return len(s) }
func (s TxByPriceAndTime) Less(i, j int) bool {
	if GasPrice(s[i]) < GasPrice(s[j]) {
		return true
	}
	if GasPrice(s[i]) == GasPrice(s[j]) {
		return time.Unix(int64(s[i].Head.TimeStamp), 0).Before(time.Unix(int64(s[j].Head.TimeStamp), 0))
	}
	return false
}
func (s TxByPriceAndTime) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s *TxByPriceAndTime) Push(x interface{}) {
	*s = append(*s, x.(*txbasic.Transaction))
}
func (s *TxByPriceAndTime) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
}
