package transactionpool

import (
	"container/heap"
	"github.com/TopiaNetwork/topia/transaction"
	"math"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"
)

type esCache struct {
	putNo uint64
	getNo uint64
	value *transaction.Transaction
}
type EsQueue struct {
	capacity uint64
	capMod   uint64
	putPos   uint64
	getPos   uint64
	cache    []esCache
}
//
//func NewQueue(capacity uint64) *EsQueue{
//	q := new(EsQueue)
//	q.capacity = MinQuantity(capacity)
//	q.capMod   = q.capacity - 1
//	q.putPos = 0
//	q.getPos = 0
//	q.cache    = make([]esCache,q.capacity)
//	for i := range q.cache {
//		cache := &q.cache[i]
//		cache.getNo = uint64(i)
//		cache.putNo = uint64(i)
//	}
//	cache := &q.cache[0]
//	cache.getNo = q.capacity
//	cache.putNo = q.capacity
//	return q
//}
//
//// MinQuantity Round up to the nearest multiple of two
//func MinQuantity(v uint64) uint64 {
//	v--
//	v |= v >> 1
//	v |= v >> 2
//	v |= v >> 4
//	v |= v >> 8
//	v |= v >> 16
//	v |= v >> 32
//	v++
//	return v
//}
//
//func (q *EsQueue) Capacity() uint64 {
//	return q.capacity
//}
//func (q *EsQueue) Quantity() uint64 {
//	var putPos,getPos uint64
//	var quantity uint64
//	getPos = atomic.LoadUint64(&q.getPos)
//	putPos = atomic.LoadUint64(&q.putPos)
//
//	if putPos >= getPos {
//		quantity = putPos - getPos
//	} else {
//		quantity = q.capMod + putPos - getPos
//	}
//	return quantity
//}
//// Put queue functions
//func (q *EsQueue) Put(val *transaction.Transaction) (ok bool,quantity uint64) {
//	var putPos,putPosNew,getPos,posCnt uint64
//	var cache *esCache
//	capMod := q.capMod
//	getPos = atomic.LoadUint64(&q.getPos)
//	putPos = atomic.LoadUint64(&q.putPos)
//	if putPos >= getPos {
//		posCnt = putPos - getPos
//	} else {
//		posCnt = capMod + putPos - getPos
//	}
//	if posCnt >= capMod-1 {
//		runtime.Gosched()
//		return false,posCnt
//	}
//	putPosNew = putPos + 1
//	if !atomic.CompareAndSwapUint64(&q.putPos,putPos,putPosNew) {
//		runtime.Gosched()
//		return false, posCnt
//	}
//	cache = &q.cache[putPosNew&capMod]
//
//	for {
//		getNo := atomic.LoadUint64(&cache.getNo)
//		putNo := atomic.LoadUint64(&cache.putNo)
//		if putPosNew == putNo && getNo == putNo {
//			cache.value = val
//			atomic.AddUint64(&cache.putNo, q.capacity)
//			return true, posCnt + 1
//		} else {
//			runtime.Gosched()
//		}
//	}
//}
//
//
//// Get queue functions
//func (q *EsQueue) Get() (val *transaction.Transaction,ok bool,quantity uint64) {
//	var putPos, getPos, getPosNew, posCnt uint64
//	var cache *esCache
//	capMod  := q.capMod
//	putPos = atomic.LoadUint64(&q.putPos)
//	getPos = atomic.LoadUint64(&q.getPos)
//
//	if putPos >= getPos {
//		posCnt = putPos - getPos
//	} else {
//		posCnt = capMod + putPos - getPos
//	}
//
//	if posCnt < 1 {
//		runtime.Gosched()
//		return nil, false, posCnt
//	}
//
//	getPosNew = getPos + 1
//	if !atomic.CompareAndSwapUint64(&q.getPos,getPos,getPosNew) {
//		runtime.Gosched()
//		return nil, false, posCnt
//	}
//
//	cache = &q.cache[getPosNew&capMod]
//
//	for {
//		getNo := atomic.LoadUint64(&cache.getNo)
//		putNo := atomic.LoadUint64(&cache.putNo)
//		if getPosNew == getNo && getNo == putNo - q.capacity {
//			val = cache.value
//			cache.value = nil
//			atomic.AddUint64(&cache.getNo,q.capacity)
//			return val, true, posCnt -1
//		} else {
//			runtime.Gosched()
//		}
//	}
//}
//

// nonceHeap is a heap.Interface implementation over 64bit unsigned integers for
// retrieving sorted transactions from the possibly gapped future queue.
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

// txSortedMap is a nonce->transaction hash map with a heap based index to allow
// iterating over the contents in a nonce-incrementing way.
type txSortedMap struct {
	items map[uint64]*transaction.Transaction // Hash map storing the transaction data
	index *nonceHeap                    // Heap of nonces of all the stored transactions (non-strict mode)
	cache []*transaction.Transaction            // Cache of the transactions already sorted
}

// newTxSortedMap creates a new nonce-sorted transaction map.
func newTxSortedMap() *txSortedMap {
	return &txSortedMap{
		items: make(map[uint64]*transaction.Transaction),
		index: new(nonceHeap),
	}
}

// Get retrieves the current transactions associated with the given nonce.
func (m *txSortedMap) Get(nonce uint64) *transaction.Transaction {
	return m.items[nonce]
}

// Put inserts a new transaction into the map, also updating the map's nonce
// index. If a transaction already exists with the same nonce, it's overwritten.
func (m *txSortedMap) Put(tx *transaction.Transaction) {
	nonce := tx.Nonce
	if m.items[nonce] == nil {
		heap.Push(m.index, nonce)
	}
	m.items[nonce], m.cache = tx, nil
}

// Forward removes all transactions from the map with a nonce lower than the
// provided threshold. Every removed transaction is returned for any post-removal
// maintenance.
func (m *txSortedMap) Forward(threshold uint64) []*transaction.Transaction {
	var removed []*transaction.Transaction

	// Pop off heap items until the threshold is reached
	for m.index.Len() > 0 && (*m.index)[0] < threshold {
		nonce := heap.Pop(m.index).(uint64)
		removed = append(removed, m.items[nonce])
		delete(m.items, nonce)
	}
	// If we had a cached order, shift the front
	if m.cache != nil {
		m.cache = m.cache[len(removed):]  //chinese ：缓存交易里删除remved
	}
	return removed
}

// Filter iterates over the list of transactions and removes all of them for which
// the specified function evaluates to true.
// Filter, as opposed to 'filter', re-initialises the heap after the operation is done.
// If you want to do several consecutive filterings, it's therefore better to first
// do a .filter(func1) followed by .Filter(func2) or reheap()
func (m *txSortedMap) Filter(filter func(*transaction.Transaction) bool) []*transaction.Transaction {
	removed := m.filter(filter)
	// If transactions were removed, the heap and cache are ruined
	if len(removed) > 0 {
		m.reheap()
	}
	return removed
}

func (m *txSortedMap) reheap() {
	*m.index = make([]uint64, 0, len(m.items))
	for nonce := range m.items {
		*m.index = append(*m.index, nonce)
	}
	heap.Init(m.index)
	m.cache = nil
}

// filter is identical to Filter, but **does not** regenerate the heap. This method
// should only be used if followed immediately by a call to Filter or reheap()
func (m *txSortedMap) filter(filter func(*transaction.Transaction) bool) []*transaction.Transaction {
	var removed []*transaction.Transaction

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

// Cap places a hard limit on the number of items, returning all transactions
// exceeding that limit.
func (m *txSortedMap) Cap(threshold int) []*transaction.Transaction {
	// Short circuit if the number of items is under the limit
	if len(m.items) <= threshold {
		return nil
	}
	// Otherwise gather and drop the highest nonce'd transactions
	var drops []*transaction.Transaction

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

// Remove deletes a transaction from the maintained map, returning whether the
// transaction was found.
func (m *txSortedMap) Remove(nonce uint64) bool {
	// Short circuit if no transaction is present
	_, ok := m.items[nonce]
	if !ok {
		return false
	}
	// Otherwise delete the transaction and fix the heap index
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

// Ready retrieves a sequentially increasing list of transactions starting at the
// provided nonce that is ready for processing. The returned transactions will be
// removed from the list.
//
// Note, all transactions with nonces lower than start will also be returned to
// prevent getting into and invalid state. This is not something that should ever
// happen but better to be self correcting than failing!
func (m *txSortedMap) Ready(start uint64) []*transaction.Transaction {
	// Short circuit if no transactions are available
	if m.index.Len() == 0 || (*m.index)[0] > start {
		return nil
	}
	// Otherwise start accumulating incremental transactions
	var ready []*transaction.Transaction
	for next := (*m.index)[0]; m.index.Len() > 0 && (*m.index)[0] == next; next++ {
		ready = append(ready, m.items[next])
		delete(m.items, next)
		heap.Pop(m.index)
	}
	m.cache = nil

	return ready
}

// Len returns the length of the transaction map.
func (m *txSortedMap) Len() int {
	return len(m.items)
}

func (m *txSortedMap) flatten() []*transaction.Transaction {
	// If the sorting was not cached yet, create and cache it
	if m.cache == nil {
		m.cache = make([]*transaction.Transaction, 0, len(m.items))
		for _, tx := range m.items {
			m.cache = append(m.cache, tx)
		}
		sort.Sort(TxByNonce(m.cache))
	}
	return m.cache
}

// Flatten creates a nonce-sorted slice of transactions based on the loosely
// sorted internal representation. The result of the sorting is cached in case
// it's requested again before any modifications are made to the contents.
func (m *txSortedMap) Flatten() []*transaction.Transaction {
	// Copy the cache to prevent accidental modifications
	cache := m.flatten()
	txs := make([]*transaction.Transaction, len(cache))
	copy(txs, cache)
	return txs
}

// LastElement returns the last element of a flattened list, thus, the
// transaction with the highest nonce
func (m *txSortedMap) LastElement() *transaction.Transaction {
	cache := m.flatten()
	return cache[len(cache)-1]
}


// txList is a "list" of transactions belonging to an account, sorted by account
// nonce. The same type can be used both for storing contiguous transactions for
// the executable/pending queue; and for storing gapped transactions for the non-
// executable/future queue, with minor behavioral changes.
type txList struct {
	strict bool         // Whether nonces are strictly continuous or not
	txs    *txSortedMap // Heap indexed sorted hash map of the transactions

	costcap *big.Int // Price of the highest costing transaction (reset only if exceeds balance)
	gascap  uint64   // Gas limit of the highest spending transaction (reset only if exceeds block limit)
}

// newTxList create a new transaction list for maintaining nonce-indexable fast,
// gapped, sortable transaction lists.
func newTxList(strict bool) *txList {
	return &txList{
		strict:  strict,
		txs:     newTxSortedMap(),
		costcap: new(big.Int),
	}
}

// Overlaps returns whether the transaction specified has the same nonce as one
// already contained within the list.
func (l *txList) Overlaps(tx *transaction.Transaction) bool {
	return l.txs.Get(tx.Nonce) != nil
}

// Add tries to insert a new transaction into the list, returning whether the
// transaction was accepted, and if yes, any previous transaction it replaced.
//
// If the new transaction is accepted into the list, the lists' cost and gas
// thresholds are also potentially updated.
func (l *txList) Add(tx *transaction.Transaction) (bool, *transaction.Transaction) {
	// If there's an older better transaction, abort
	old := l.txs.Get(tx.Nonce)
	if old != nil {
		return false,nil
		}
	//no achieved for priceBump
	txQuery := TxPoolQuery{}
	l.txs.Put(tx)
	if cost := txQuery.EstimateTxCost(tx);l.costcap.Cmp(cost) <0 {
		l.costcap = cost
	}
	if gas := txQuery.EstimateTxGas(tx);l.gascap < gas {
		l.gascap = gas
	}
	return true, old
}

// Forward removes all transactions from the list with a nonce lower than the
// provided threshold. Every removed transaction is returned for any post-removal
// maintenance.
func (l *txList) Forward(threshold uint64) []*transaction.Transaction {
	return l.txs.Forward(threshold)
}

// Filter removes all transactions from the list with a cost or gas limit higher
// than the provided thresholds. Every removed transaction is returned for any
// post-removal maintenance. Strict-mode invalidated transactions are also
// returned.
//
// This method uses the cached costcap and gascap to quickly decide if there's even
// a point in calculating all the costs or if the balance covers all. If the threshold
// is lower than the costgas cap, the caps will be reset to a new high after removing
// the newly invalidated transactions.
func (l *txList) Filter(costLimit *big.Int, gasLimit uint64) ([]*transaction.Transaction, []*transaction.Transaction) {
	// If all transactions are below the threshold, short circuit
	if l.costcap.Cmp(costLimit) <= 0 && l.gascap <= gasLimit {
		return nil, nil
	}
	l.costcap = new(big.Int).Set(costLimit) // Lower the caps to the thresholds
	l.gascap = gasLimit
	txQuery := TxPoolQuery{}
	// Filter out all the transactions above the account's funds
	removed := l.txs.Filter(func(tx *transaction.Transaction) bool {
		gas := txQuery.EstimateTxGas(tx)
		cost := txQuery.EstimateTxCost(tx)
		return gas > gasLimit || cost.Cmp(costLimit) > 0
	})

	if len(removed) == 0 {
		return nil, nil
	}
	var invalids []*transaction.Transaction
	// If the list was strict, filter anything above the lowest nonce
	if l.strict {
		lowest := uint64(math.MaxUint64)
		for _, tx := range removed {
			if nonce := tx.Nonce; lowest > nonce {
				lowest = nonce
			}
		}
		invalids = l.txs.filter(func(tx *transaction.Transaction) bool { return tx.Nonce > lowest })
	}
	l.txs.reheap()
	return removed, invalids
}

// Cap places a hard limit on the number of items, returning all transactions
// exceeding that limit.
func (l *txList) Cap(threshold int) []*transaction.Transaction {
	return l.txs.Cap(threshold)
}

// Remove deletes a transaction from the maintained list, returning whether the
// transaction was found, and also returning any transaction invalidated due to
// the deletion (strict mode only).
func (l *txList) Remove(tx *transaction.Transaction) (bool, []*transaction.Transaction) {
	// Remove the transaction from the set
	nonce := tx.Nonce
	if removed := l.txs.Remove(nonce); !removed {
		return false, nil
	}
	// In strict mode, filter out non-executable transactions
	if l.strict {
		return true, l.txs.Filter(func(tx *transaction.Transaction) bool { return tx.Nonce > nonce })
	}
	return true, nil
}

// Ready retrieves a sequentially increasing list of transactions starting at the
// provided nonce that is ready for processing. The returned transactions will be
// removed from the list.
//
// Note, all transactions with nonces lower than start will also be returned to
// prevent getting into and invalid state. This is not something that should ever
// happen but better to be self correcting than failing!
func (l *txList) Ready(start uint64) []*transaction.Transaction {
	return l.txs.Ready(start)
}

// Len returns the length of the transaction list.
func (l *txList) Len() int {
	return l.txs.Len()
}

// Empty returns whether the list of transactions is empty or not.
func (l *txList) Empty() bool {
	return l.Len() == 0
}

// Flatten creates a nonce-sorted slice of transactions based on the loosely
// sorted internal representation. The result of the sorting is cached in case
// it's requested again before any modifications are made to the contents.
func (l *txList) Flatten() []*transaction.Transaction {
	return l.txs.Flatten()
}

// LastElement returns the last element of a flattened list, thus, the
// transaction with the highest nonce
func (l *txList) LastElement() *transaction.Transaction {
	return l.txs.LastElement()
}


// priceHeap is a heap.Interface implementation over transactions for retrieving
// price-sorted transactions to discard when the pool fills up.
type priceHeap struct{
	list    []*transaction.Transaction
}

func (h *priceHeap) Len() int      { return len(h.list) }
func (h *priceHeap) Swap(i, j int) { h.list[i], h.list[j] = h.list[j], h.list[i] }

//chinese 两笔交易排序，先对比价格，价格一样，根据Nonce倒序排列，即优先抛掉价格小的，或者nonce大的。
func (h *priceHeap) Less(i, j int) bool {
	switch h.cmp(h.list[i], h.list[j]) {
	case -1:
		return true
	case 1:
		return false
	default:
		return h.list[i].Nonce > h.list[j].Nonce
	}
}
func (h *priceHeap) cmp(a, b *transaction.Transaction) int {
	if a.GasPrice == b.GasPrice{return 0}
	if a.GasPrice < b.GasPrice{
		return 1
	} else if a.GasPrice > b.GasPrice{
		return -1
	} else{
		return 0
	}
}

func (h *priceHeap) Push(x interface{}) {
	tx := x.(*transaction.Transaction)
	h.list = append(h.list, tx)
}

func (h *priceHeap) Pop() interface{} {
	old := h.list
	n := len(old)
	x := old[n-1]
	old[n-1] = nil
	h.list = old[0 : n-1]
	return x
}


type txPricedList struct {
	stales int64
	all              *txLookup  // Pointer to the map of all transactions
	floating         priceHeap  // Heaps of prices of all the stored **remote** transactions
	reheapMu         sync.Mutex // Mutex asserts that only one routine is reheaping the list
}


// newTxPricedList creates a new price-sorted transaction heap.
func newTxPricedList(all *txLookup) *txPricedList {
	return &txPricedList{
		all: all,
	}
}

// Put inserts a new transaction into the heap.
func (l *txPricedList) Put(tx *transaction.Transaction, local bool) {
	if local {
		return
	}
	// Insert every new transaction to the urgent heap first; Discard will balance the heaps
	heap.Push(&l.floating, tx)
}

//Removed chinese 删除N条交易，当删除当交易大于队列中当四分之一的时候，重置队列。
func (l *txPricedList) Removed(count int) {
	stales := atomic.AddInt64(&l.stales, int64(count))
	if int(stales) <= (len(l.floating.list))/4 {
		return
	}
	l.Reheap()
}

// Underpriced checks whether a transaction is cheaper than (or as cheap as) the
// lowest priced (remote) transaction currently being tracked.
func (l *txPricedList) Underpriced(tx *transaction.Transaction) bool {
	// Note: with two queues, being underpriced is defined as being worse than the worst item
	// in all non-empty queues if there is any. If both queues are empty then nothing is underpriced.
	return (l.underpricedFor(&l.floating, tx) || len(l.floating.list) == 0) &&
		( len(l.floating.list) != 0)
}

// underpricedFor checks whether a transaction is cheaper than (or as cheap as) the
// lowest priced (remote) transaction in the given heap.
func (l *txPricedList) underpricedFor(h *priceHeap, tx *transaction.Transaction) bool {
	// Discard stale price points if found at the heap start
	for len(h.list) > 0 {
		head := h.list[0]
		txId,err := head.TxID()
		if err == nil {
			if l.all.GetRemote(txId) == nil { // Removed or migrated
				atomic.AddInt64(&l.stales, -1)
				heap.Pop(h)
				continue
			}
		}
		if l.all.GetRemote(txId) == nil { // Removed or migrated
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
func (l *txPricedList) Discard(slots int, force bool) ([]*transaction.Transaction, bool) {
	drop := make([]*transaction.Transaction, 0, slots) // Remote underpriced transactions to drop
	for slots > 0 {
		// Discard stale transactions if found during cleanup
		tx := heap.Pop(&l.floating).(*transaction.Transaction)
		if txId,err := tx.TxID();err == nil {
			if l.all.GetRemote(txId) == nil { // Removed or migrated
				atomic.AddInt64(&l.stales, -1)
				continue
			}
		}
		drop = append(drop, tx)
		slots -= numSlots(tx)
	}
	// If we still can't make enough room for the new transaction
	if slots > 0 && !force {
		for _, tx := range drop {
			heap.Push(&l.floating, tx)
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
	l.floating.list = make([]*transaction.Transaction, 0, l.all.RemoteCount())
	l.all.Range(func(key transaction.TxID, tx *transaction.Transaction, local bool) bool {
		l.floating.list = append(l.floating.list, tx)
		return true
	}, false, true) // Only iterate remotes
	heap.Init(&l.floating)
}














type TxByNonce []*transaction.Transaction

func (s TxByNonce) Len() int           { return len(s) }
func (s TxByNonce) Less(i, j int) bool { return s[i].Nonce < s[j].Nonce }
func (s TxByNonce) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
