package memdb

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/google/btree"

	tplgcmm "github.com/TopiaNetwork/topia/ledger/backend/common"
	tplog "github.com/TopiaNetwork/topia/log"
)

const (
	// The approximate number of items and children per B-tree node. Tuned with benchmarks.
	bTreeDegree = 32
)

// MemDBBackend is an in-memory database backend using a B-tree for storage.
//
// For performance reasons, all given and returned keys and values are pointers to the in-memory
// database, so modifying them will cause the stored values to be modified as well. All DB methods
// already specify that keys and values should be considered read-only, but this is especially
// important with MemDBBackend.
//
// Versioning is implemented by maintaining references to copy-on-write clones of the backing btree.
//
// Note: Currently, transactions do not detect write conflicts, so multiple writers cannot be
// safely committed to overlapping domains. Because of this, the number of open writers is
// limited to 1.
type MemDBBackend struct {
	log         tplog.Logger
	btree       *btree.BTree            // Main contents
	mtx         sync.RWMutex            // Guards version history
	saved       map[uint64]*btree.BTree // Past versions
	vmgr        *tplgcmm.VersionManager // Mirrors version keys
	openWriters int32                   // Open writers
}

type dbTxn struct {
	btree *btree.BTree
	db    *MemDBBackend
}
type dbWriter struct{ dbTxn }

// item is a btree.Item with byte slices as keys and values
type item struct {
	key   []byte
	value []byte
}

// NewDB creates a new in-memory database.
func NewMemDBBackend(log tplog.Logger) *MemDBBackend {
	return &MemDBBackend{
		btree: btree.New(bTreeDegree),
		saved: make(map[uint64]*btree.BTree),
		vmgr:  tplgcmm.NewVersionManager(nil),
	}
}

func (m *MemDBBackend) newTxn(tree *btree.BTree) dbTxn {
	return dbTxn{tree, m}
}

// Close implements DB.
// Close is a noop since for an in-memory database, we don't have a destination to flush
// contents to nor do we want any data loss on invoking Close().
// See the discussion in https://github.com/tendermint/tendermint/libs/pull/56
func (m *MemDBBackend) Close() error {
	return nil
}

// Versions implements DBConnection.
func (m *MemDBBackend) Versions() (tplgcmm.VersionSet, error) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	return m.vmgr, nil
}

// Reader implements DBConnection.
func (m *MemDBBackend) Reader() tplgcmm.DBReader {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	ret := m.newTxn(m.btree)
	return &ret
}

// ReaderAt implements DBConnection.
func (m *MemDBBackend) ReaderAt(version uint64) (tplgcmm.DBReader, error) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	tree, ok := m.saved[version]
	if !ok {
		return nil, tplgcmm.ErrVersionDoesNotExist
	}
	ret := m.newTxn(tree)
	return &ret, nil
}

// Writer implements DBConnection.
func (m *MemDBBackend) Writer() tplgcmm.DBWriter {
	return m.ReadWriter()
}

// ReadWriter implements DBConnection.
func (m *MemDBBackend) ReadWriter() tplgcmm.DBReadWriter {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	atomic.AddInt32(&m.openWriters, 1)
	// Clone creates a copy-on-write extension of the current tree
	return &dbWriter{m.newTxn(m.btree.Clone())}
}

func (m *MemDBBackend) save(target uint64) (uint64, error) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if m.openWriters > 0 {
		return 0, tplgcmm.ErrOpenTransactions
	}

	newVmgr := m.vmgr.Copy()
	target, err := newVmgr.Save(target)
	if err != nil {
		return 0, err
	}
	m.saved[target] = m.btree
	m.vmgr = newVmgr
	return target, nil
}

// SaveVersion implements DBConnection.
func (m *MemDBBackend) SaveNextVersion() (uint64, error) {
	return m.save(0)
}

// SaveNextVersion implements DBConnection.
func (m *MemDBBackend) SaveVersion(target uint64) error {
	if target == 0 {
		return tplgcmm.ErrInvalidVersion
	}
	_, err := m.save(target)
	return err
}

// DeleteVersion implements DBConnection.
func (m *MemDBBackend) DeleteVersion(target uint64) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if _, has := m.saved[target]; !has {
		return tplgcmm.ErrVersionDoesNotExist
	}
	delete(m.saved, target)
	m.vmgr = m.vmgr.Copy()
	m.vmgr.Delete(target)
	return nil
}

func (m *MemDBBackend) Revert() error {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	if m.openWriters > 0 {
		return tplgcmm.ErrOpenTransactions
	}

	last := m.vmgr.Last()
	if last == 0 {
		m.btree = btree.New(bTreeDegree)
		return nil
	}
	var has bool
	m.btree, has = m.saved[last]
	if !has {
		return fmt.Errorf("bad version history: version %v not saved", last)
	}
	for ver, _ := range m.saved {
		if ver > last {
			delete(m.saved, ver)
		}
	}
	return nil
}

func (m *MemDBBackend) PendingTxCount() int32 {
	return m.openWriters
}

// Get implements DBReader.
func (tx *dbTxn) Get(key []byte) ([]byte, error) {
	if tx.btree == nil {
		return nil, tplgcmm.ErrTransactionClosed
	}
	if len(key) == 0 {
		return nil, tplgcmm.ErrKeyEmpty
	}
	i := tx.btree.Get(newKey(key))
	if i != nil {
		return i.(*item).value, nil
	}
	return nil, nil
}

// Has implements DBReader.
func (tx *dbTxn) Has(key []byte) (bool, error) {
	if tx.btree == nil {
		return false, tplgcmm.ErrTransactionClosed
	}
	if len(key) == 0 {
		return false, tplgcmm.ErrKeyEmpty
	}
	return tx.btree.Has(newKey(key)), nil
}

// Set implements DBWriter.
func (tx *dbWriter) Set(key []byte, value []byte) error {
	if tx.btree == nil {
		return tplgcmm.ErrTransactionClosed
	}
	if err := tplgcmm.ValidateKv(key, value); err != nil {
		return err
	}
	tx.btree.ReplaceOrInsert(newPair(key, value))
	return nil
}

// Delete implements DBWriter.
func (tx *dbWriter) Delete(key []byte) error {
	if tx.btree == nil {
		return tplgcmm.ErrTransactionClosed
	}
	if len(key) == 0 {
		return tplgcmm.ErrKeyEmpty
	}
	tx.btree.Delete(newKey(key))
	return nil
}

// Iterator implements DBReader.
// Takes out a read-lock on the database until the iterator is closed.
func (tx *dbTxn) Iterator(start, end []byte) (tplgcmm.Iterator, error) {
	if tx.btree == nil {
		return nil, tplgcmm.ErrTransactionClosed
	}
	if (start != nil && len(start) == 0) || (end != nil && len(end) == 0) {
		return nil, tplgcmm.ErrKeyEmpty
	}
	return newMemDBIterator(tx, start, end, false), nil
}

// ReverseIterator implements DBReader.
// Takes out a read-lock on the database until the iterator is closed.
func (tx *dbTxn) ReverseIterator(start, end []byte) (tplgcmm.Iterator, error) {
	if tx.btree == nil {
		return nil, tplgcmm.ErrTransactionClosed
	}
	if (start != nil && len(start) == 0) || (end != nil && len(end) == 0) {
		return nil, tplgcmm.ErrKeyEmpty
	}
	return newMemDBIterator(tx, start, end, true), nil
}

// Commit implements DBWriter.
func (tx *dbWriter) Commit() error {
	if tx.btree == nil {
		return tplgcmm.ErrTransactionClosed
	}
	tx.db.mtx.Lock()
	defer tx.db.mtx.Unlock()
	tx.db.btree = tx.btree
	return tx.Discard()
}

// Discard implements DBReader.
func (tx *dbTxn) Discard() error {
	if tx.btree != nil {
		tx.btree = nil
	}
	return nil
}

// Discard implements DBWriter.
func (tx *dbWriter) Discard() error {
	if tx.btree != nil {
		defer atomic.AddInt32(&tx.db.openWriters, -1)
	}
	return tx.dbTxn.Discard()
}

// Print prints the database contents.
func (m *MemDBBackend) Print() error {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	m.btree.Ascend(func(i btree.Item) bool {
		item := i.(*item)
		fmt.Printf("[%X]:\t[%X]\n", item.key, item.value)
		return true
	})
	return nil
}

// Stats implements DBConnection.
func (m *MemDBBackend) Stats() map[string]string {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	stats := make(map[string]string)
	stats["database.type"] = "memDB"
	stats["database.size"] = fmt.Sprintf("%d", m.btree.Len())
	return stats
}

// Less implements btree.Item.
func (i *item) Less(other btree.Item) bool {
	// this considers nil == []byte{}, but that's ok since we handle nil endpoints
	// in iterators specially anyway
	return bytes.Compare(i.key, other.(*item).key) == -1
}

// newKey creates a new key item.
func newKey(key []byte) *item {
	return &item{key: key}
}

// newPair creates a new pair item.
func newPair(key, value []byte) *item {
	return &item{key: key, value: value}
}
