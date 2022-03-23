package transactionpool

import (
	"container/heap"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TopiaNetwork/topia/account"
	"github.com/TopiaNetwork/topia/transaction"
)

const (
	txSlotSize = 32 * 1024
	txMaxSize  = 4 * txSlotSize
)

//nonceHeap is a heap.Interface implementation over 64bit unsigned integers for
//retrieving sorted transactions from the possibly gapped future queue.
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
	index *nonceHeap                          // Heap of nonces of all the stored transactions (non-strict mode)
	cache []*transaction.Transaction          // Cache of the transactions already sorted
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
		m.cache = m.cache[len(removed):]
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
	strict  bool         // Whether nonces are strictly continuous or not
	txs     *txSortedMap // Heap indexed sorted hash map of the transactions
	servant TransactionPoolServant
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
		return false, nil
	}
	//txQuery := l.servant
	l.txs.Put(tx)
	//if cost := txQuery.EstimateTxCost(tx); l.costcap.Cmp(cost) < 0 {
	//	l.costcap = cost
	//}
	//
	//if gas := txQuery.EstimateTxGas(tx); l.gascap < gas {
	//	l.gascap = gas
	//}

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
	txQuery := l.servant
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

type pendingTxs struct {
	Mu     sync.RWMutex
	accTxs map[account.Address]*txList
}

func newPendingTxs() *pendingTxs {
	txs := &pendingTxs{
		accTxs: make(map[account.Address]*txList),
	}
	return txs
}

type queueTxs struct {
	Mu     sync.RWMutex
	accTxs map[account.Address]*txList
}

func newQueueTxs() *queueTxs {
	txs := &queueTxs{
		accTxs: make(map[account.Address]*txList),
	}
	return txs
}

type accountSet struct {
	accounts map[account.Address]uint8
	cache    *[]account.Address
}

func newAccountSet(addrs ...account.Address) *accountSet {
	as := &accountSet{
		accounts: make(map[account.Address]uint8),
	}
	for _, addr := range addrs {
		as.add(addr)
	}
	return as
}

func (accSet *accountSet) contains(addr account.Address) bool {

	_, exist := accSet.accounts[addr]
	return exist
}
func (accSet *accountSet) containsTx(tx *transaction.Transaction) bool {
	addr := account.Address(hex.EncodeToString(tx.FromAddr))
	return accSet.contains(addr)
}

func (accSet *accountSet) empty() bool {
	return len(accSet.accounts) == 0
}
func (accSet *accountSet) len() int {
	return len(accSet.accounts)
}

func (accSet *accountSet) add(addr account.Address) {
	accSet.accounts[addr] = 1
	accSet.cache = nil
}
func (accSet *accountSet) addTx(tx *transaction.Transaction) {
	addr := account.Address(hex.EncodeToString(tx.FromAddr))
	accSet.add(addr)
}
func (accSet *accountSet) merge(other *accountSet) {
	for addr := range other.accounts {
		accSet.accounts[addr] = 1
	}
	accSet.cache = nil
}

func (accSet *accountSet) RemoveAccount(addr account.Address) {
	delete(accSet.accounts, addr)
}

// flatten returns the list of addresses within this set, also caching it for later
// reuse. The returned slice should not be changed!
func (accSet *accountSet) flatten() []account.Address {
	if accSet.cache == nil {
		accounts := make([]account.Address, 0, len(accSet.accounts))
		for acc := range accSet.accounts {
			accounts = append(accounts, acc)
		}
		accSet.cache = &accounts
	}
	return *accSet.cache
}

type txLookup struct {
	slots   int
	lock    sync.RWMutex
	locals  map[string]*transaction.Transaction
	remotes map[string]*transaction.Transaction
}

func (t *txLookup) Range(f func(key string, tx *transaction.Transaction, local bool) bool, local bool, remote bool) {
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
		locals:  make(map[string]*transaction.Transaction),
		remotes: make(map[string]*transaction.Transaction),
	}
}
func (t *txLookup) Get(key string) *transaction.Transaction {
	t.lock.RLock()
	defer t.lock.RUnlock()
	if tx := t.locals[key]; tx != nil {
		return tx
	}
	return t.remotes[key]
}

// GetLocalTx returns a transaction if it exists in the lookup, or nil if not found.
func (t *txLookup) GetLocalTx(key string) *transaction.Transaction {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.locals[key]
}

// GetRemoteTx returns a transaction if it exists in the lookup, or nil if not found.
func (t *txLookup) GetRemoteTx(key string) *transaction.Transaction {
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

// Slots returns the current number of Quota used in the lookup.
func (t *txLookup) Slots() int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.slots
}

func (t *txLookup) Add(tx *transaction.Transaction, local bool) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.slots += numSlots(tx)
	if txId, err := tx.TxID(); err == nil {
		if local {
			t.locals[txId] = tx
			fmt.Println("txlookup add tx 00001")
		} else {
			t.remotes[txId] = tx
		}
	}

}
func (t *txLookup) Remove(key string) {
	t.lock.Lock()
	defer t.lock.Unlock()
	tx, ok := t.locals[key]
	fmt.Println("local:", ok)
	if !ok {
		tx, ok = t.remotes[key]
	}
	if !ok {
		//tplog.Logger.Infof("No transaction found to be deleted", "txKey", key)
		return
	}
	t.slots -= numSlots(tx)
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

// priceHeap is a heap.Interface implementation over transactions for retrieving
// price-sorted transactions to discard when the pool fills up.
type priceHeap struct {
	list []*transaction.Transaction
}

func (h *priceHeap) Len() int      { return len(h.list) }
func (h *priceHeap) Swap(i, j int) { h.list[i], h.list[j] = h.list[j], h.list[i] }

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
	if a.GasPrice == b.GasPrice {
		return 0
	}
	if a.GasPrice < b.GasPrice {
		return 1
	} else if a.GasPrice > b.GasPrice {
		return -1
	} else {
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
	stales    int64
	all       *txLookup  // Pointer to the map of all transactions
	remoteTxs priceHeap  // Heaps of prices of all the stored **remote** transactions
	reheapMu  sync.Mutex // Mutex asserts that only one routine is reheaping the list
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
func (l *txPricedList) Underpriced(tx *transaction.Transaction) bool {
	// Note: with two queues, being underpriced is defined as being worse than the worst item
	// in all non-empty queues if there is any. If both queues are empty then nothing is underpriced.
	return (l.underpricedFor(&l.remoteTxs, tx) || len(l.remoteTxs.list) == 0) &&
		(len(l.remoteTxs.list) != 0)
}

// underpricedFor checks whether a transaction is cheaper than (or as cheap as) the
// lowest priced (remote) transaction in the given heap.
func (l *txPricedList) underpricedFor(h *priceHeap, tx *transaction.Transaction) bool {
	// Discard stale price points if found at the heap start
	for len(h.list) > 0 {
		head := h.list[0]
		txId, err := head.TxID()
		if err == nil {
			if l.all.GetRemoteTx(txId) == nil { // Removed or migrated
				atomic.AddInt64(&l.stales, -1)
				heap.Pop(h)
				continue
			}
		}
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
func (l *txPricedList) Discard(slots int, force bool) ([]*transaction.Transaction, bool) {
	drop := make([]*transaction.Transaction, 0, slots) // Remote underpriced transactions to drop
	for slots > 0 {
		// Discard stale transactions if found during cleanup
		tx := heap.Pop(&l.remoteTxs).(*transaction.Transaction)
		if txId, err := tx.TxID(); err == nil {
			if l.all.GetRemoteTx(txId) == nil { // Removed or migrated
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
	l.remoteTxs.list = make([]*transaction.Transaction, 0, l.all.RemoteCount())
	l.all.Range(func(key string, tx *transaction.Transaction, local bool) bool {
		l.remoteTxs.list = append(l.remoteTxs.list, tx)
		return true
	}, false, true) // Only iterate remotes
	heap.Init(&l.remoteTxs)
}

// numSlots calculates the number of slots needed for a single transaction.
func numSlots(tx *transaction.Transaction) int {
	return int((tx.Size() + txSlotSize - 1) / txSlotSize)
}

type TxByNonce []*transaction.Transaction

func (s TxByNonce) Len() int           { return len(s) }
func (s TxByNonce) Less(i, j int) bool { return s[i].Nonce < s[j].Nonce }
func (s TxByNonce) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// CntAccountItem is an item  we manage in a priority queue for cnt.
type CntAccountItem struct {
	accountAddr account.Address
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

// addressByHeartbeat is an account address tagged with its last activity timestamp.
type txByHeartbeat struct {
	tx                 string
	ActivationInterval time.Time
}

type txsByHeartbeat []txByHeartbeat

func (t txsByHeartbeat) Len() int { return len(t) }
func (t txsByHeartbeat) Less(i, j int) bool {
	return t[i].ActivationInterval.Before(t[j].ActivationInterval)
}
func (t txsByHeartbeat) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
