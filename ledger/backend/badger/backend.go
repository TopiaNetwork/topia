package badger

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"

	"github.com/dgraph-io/badger/v3"
	bpb "github.com/dgraph-io/badger/v3/pb"
	"github.com/dgraph-io/ristretto/z"
	lru "github.com/hashicorp/golang-lru"

	tplgcmm "github.com/TopiaNetwork/topia/ledger/backend/common"
	"github.com/TopiaNetwork/topia/ledger/backend/version"
	tplog "github.com/TopiaNetwork/topia/log"
)

var (
	versionsFilename = "versions.csv"
)

type BadgerBackend struct {
	log        tplog.Logger
	name       string
	cache      *lru.ARCCache
	db         *badger.DB
	versionMng *versionManager
}

func readVersionsFile(path string) (*versionManager, error) {
	file, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	r := csv.NewReader(file)
	r.FieldsPerRecord = 2
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	var (
		versions []uint64
		lastTs   uint64
	)
	vmap := map[uint64]uint64{}
	for _, row := range rows {
		version, err := strconv.ParseUint(row[0], 10, 64)
		if err != nil {
			return nil, err
		}
		ts, err := strconv.ParseUint(row[1], 10, 64)
		if err != nil {
			return nil, err
		}
		if version == 0 { // 0 maps to the latest timestamp
			lastTs = ts
		}
		versions = append(versions, version)
		vmap[version] = ts
	}
	vMNG := version.NewVersionManager(versions)
	return &versionManager{
		VersionManager: vMNG,
		vmap:           vmap,
		lastTs:         lastTs,
	}, nil
}

func NewBadgerBackend(log tplog.Logger, name string, path string, cacheSize int) *BadgerBackend {
	pathWithName := filepath.Join(path, name+".db")
	if err := os.MkdirAll(pathWithName, 0755); err != nil {
		log.Panicf("can't change the path %s to 0755", pathWithName)
		return nil
	}

	versionMNG, err := readVersionsFile(filepath.Join(path, versionsFilename))
	if err != nil {
		return nil
	}

	opts := badger.DefaultOptions(pathWithName)
	opts.SyncWrites = false
	opts.Logger = nil

	db, err := badger.Open(opts)
	if err != nil {
		log.Panicf("can't open badger: path=%s, err=%v", pathWithName, err)
		return nil
	}

	cache, _ := lru.NewARC(cacheSize)
	return &BadgerBackend{
		log:        log,
		name:       name,
		cache:      cache,
		db:         db,
		versionMng: versionMNG,
	}
}

func (b *BadgerBackend) vesionTxn(version *uint64) *badger.Txn {
	var ts uint64
	if version == nil {
		ts = b.versionMng.lastTs
	} else {
		if tsm, has := b.versionMng.versionTs(*version); has {
			ts = tsm
		} else {
			return nil
		}
	}

	return b.db.NewTransactionAt(ts, true)
}

func (b *BadgerBackend) Get(key []byte, version *uint64) ([]byte, error) {
	txn := b.vesionTxn(version)
	if txn == nil {
		return nil, fmt.Errorf("Can't get badger txn: version=%d", *version)
	}
	defer txn.Discard()

	item, err := txn.Get(key)
	if err == badger.ErrKeyNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	val, err := item.ValueCopy(nil)
	if err == nil && val == nil {
		val = []byte{}
	}

	return val, err
}

func (b *BadgerBackend) Has(key []byte, version *uint64) (bool, error) {
	txn := b.vesionTxn(version)
	if txn == nil {
		return false, fmt.Errorf("Can't get badger txn: version=%d", *version)
	}
	defer txn.Discard()

	_, err := txn.Get(key)
	if err != nil && err != badger.ErrKeyNotFound {
		return false, err
	}

	return (err != badger.ErrKeyNotFound), nil
}

func (b *BadgerBackend) Set(key, value []byte) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
}

func withSync(db *badger.DB, err error) error {
	if err != nil {
		return err
	}
	return db.Sync()
}

func (b *BadgerBackend) SetSync(key, value []byte) error {
	return withSync(b.db, b.Set(key, value))
}

func (b *BadgerBackend) Delete(key []byte) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

func (b *BadgerBackend) DeleteSync(key []byte) error {
	return withSync(b.db, b.Delete(key))
}

func writeVersionsFile(vm *versionManager, path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	w := csv.NewWriter(file)
	rows := [][]string{
		[]string{"0", strconv.FormatUint(vm.lastTs, 10)},
	}
	for it := vm.Iterator(); it.Next(); {
		version := it.Value()
		ts, ok := vm.vmap[version]
		if !ok {
			panic("version not mapped to ts")
		}
		rows = append(rows, []string{
			strconv.FormatUint(it.Value(), 10),
			strconv.FormatUint(ts, 10),
		})
	}
	return w.WriteAll(rows)
}

func (b *BadgerBackend) Close() error {
	writeVersionsFile(b.versionMng, filepath.Join(b.db.Opts().Dir, versionsFilename))
	return b.db.Close()
}

func (b *BadgerBackend) Print() error {
	return nil
}

func (b *BadgerBackend) iteratorOpts(start, end []byte, opts badger.IteratorOptions, version *uint64) (*BadgerBackendIterator, error) {
	if (start != nil && len(start) == 0) || (end != nil && len(end) == 0) {
		return nil, errors.New("invalid input params")
	}
	txn := b.vesionTxn(version)
	iter := txn.NewIterator(opts)
	iter.Rewind()
	iter.Seek(start)
	if opts.Reverse && iter.Valid() && bytes.Equal(iter.Item().Key(), start) {
		// If we're going in reverse, our starting point was "end",
		// which is exclusive.
		iter.Next()
	}
	return &BadgerBackendIterator{
		reverse: opts.Reverse,
		start:   start,
		end:     end,

		txn:  txn,
		iter: iter,
	}, nil
}

func (b *BadgerBackend) Iterator(start, end []byte, version *uint64) (tplgcmm.Iterator, error) {
	opts := badger.DefaultIteratorOptions
	return b.iteratorOpts(start, end, opts, version)
}

func (b *BadgerBackend) ReverseIterator(start, end []byte, version *uint64) (tplgcmm.Iterator, error) {
	opts := badger.DefaultIteratorOptions
	opts.Reverse = true
	return b.iteratorOpts(end, start, opts, version)
}

func (b *BadgerBackend) Stats() map[string]string {
	return nil
}

func (b *BadgerBackend) Versions() (tplgcmm.VersionSet, error) {
	return b.versionMng, nil
}

func (b *BadgerBackend) save(version uint64) (uint64, error) {
	b.versionMng = b.versionMng.Copy()
	return b.versionMng.Save(version)
}

func (b *BadgerBackend) SaveNextVersion() (uint64, error) {
	return b.save(0)
}

func (b *BadgerBackend) SaveVersion(version uint64) error {
	_, err := b.save(version)
	return err
}

func (b *BadgerBackend) DeleteVersion(version uint64) error {
	if !b.versionMng.Exists(version) {
		return fmt.Errorf("version %d not exist", version)
	}
	b.versionMng = b.versionMng.Copy()
	b.versionMng.Delete(version)

	return nil
}

func (b *BadgerBackend) Revert() error {
	// Revert from latest commit timestamp to last "saved" timestamp
	// if no versions exist, use 0 as it precedes any possible commit timestamp
	var target uint64
	last := b.versionMng.Last()
	if last == 0 {
		target = 0
	} else {
		var has bool
		if target, has = b.versionMng.versionTs(last); !has {
			return errors.New("bad version history")
		}
	}
	lastTs := b.versionMng.lastTs
	if target == lastTs {
		return nil
	}

	// Badger provides no way to rollback committed data, so we undo all changes
	// since the target version using the Stream API
	stream := b.db.NewStreamAt(lastTs)
	// Skips unchanged keys
	stream.ChooseKey = func(item *badger.Item) bool { return item.Version() > target }
	// Scans for value at target version
	stream.KeyToList = func(key []byte, itr *badger.Iterator) (*bpb.KVList, error) {
		kv := bpb.KV{Key: key}
		// advance down to <= target version
		itr.Next() // we have at least one newer version
		for itr.Valid() && bytes.Equal(key, itr.Item().Key()) && itr.Item().Version() > target {
			itr.Next()
		}
		if itr.Valid() && bytes.Equal(key, itr.Item().Key()) && !itr.Item().IsDeletedOrExpired() {
			var err error
			kv.Value, err = itr.Item().ValueCopy(nil)
			if err != nil {
				return nil, err
			}
		}
		return &bpb.KVList{Kv: []*bpb.KV{&kv}}, nil
	}
	txn := b.db.NewTransactionAt(lastTs, true)
	defer txn.Discard()
	stream.Send = func(buf *z.Buffer) error {
		kvl, err := badger.BufferToKVList(buf)
		if err != nil {
			return err
		}
		// nil Value indicates a deleted entry
		for _, kv := range kvl.Kv {
			if kv.Value == nil {
				err = txn.Delete(kv.Key)
				if err != nil {
					return err
				}
			} else {
				err = txn.Set(kv.Key, kv.Value)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	err := stream.Orchestrate(context.Background())
	if err != nil {
		return err
	}
	return txn.CommitAt(lastTs, nil)
}

func (b *BadgerBackend) LastVersion() uint64 {
	return b.versionMng.Last()
}

func (b *BadgerBackend) Commit() error {
	txn := b.vesionTxn(nil)
	if txn == nil {
		return errors.New("Can't get the last bedger txt")
	}

	b.versionMng.updateCommitTs(txn.ReadTs())

	return txn.CommitAt(b.versionMng.lastTs, nil)
}

func (b *BadgerBackend) NewBatch() tplgcmm.Batch {
	wb := &BadgerBackendBatch{
		db:         b.db,
		wb:         b.db.NewWriteBatch(),
		firstFlush: make(chan struct{}, 1),
	}
	wb.firstFlush <- struct{}{}
	return wb
}

var _ tplgcmm.Batch = (*BadgerBackendBatch)(nil)

type BadgerBackendBatch struct {
	db *badger.DB
	wb *badger.WriteBatch

	// Calling db.Flush twice panics, so we must keep track of whether we've
	// flushed already on our own. If Write can receive from the firstFlush
	// channel, then it's the first and only Flush call we should do.
	//
	// Upstream bug report:
	// https://github.com/dgraph-io/badger/issues/1394
	firstFlush chan struct{}
}

func (b *BadgerBackendBatch) Set(key, value []byte) error {
	return b.wb.Set(key, value)
}

func (b *BadgerBackendBatch) Delete(key []byte) error {
	return b.wb.Delete(key)
}

func (b *BadgerBackendBatch) Write() error {
	select {
	case <-b.firstFlush:
		return b.wb.Flush()
	default:
		return fmt.Errorf("batch already flushed")
	}
}

func (b *BadgerBackendBatch) WriteSync() error {
	return withSync(b.db, b.Write())
}

func (b *BadgerBackendBatch) Close() error {
	select {
	case <-b.firstFlush: // a Flush after Cancel panics too
	default:
	}
	b.wb.Cancel()
	return nil
}

type BadgerBackendIterator struct {
	reverse    bool
	start, end []byte

	txn  *badger.Txn
	iter *badger.Iterator

	lastErr error
}

func (i *BadgerBackendIterator) Close() error {
	i.iter.Close()
	i.txn.Discard()
	return nil
}

func (i *BadgerBackendIterator) Domain() (start, end []byte) { return i.start, i.end }
func (i *BadgerBackendIterator) Error() error                { return i.lastErr }

func (i *BadgerBackendIterator) Next() {
	if !i.Valid() {
		panic("iterator is invalid")
	}
	i.iter.Next()
}

func (i *BadgerBackendIterator) Valid() bool {
	if !i.iter.Valid() {
		return false
	}
	if len(i.end) > 0 {
		key := i.iter.Item().Key()
		if c := bytes.Compare(key, i.end); (!i.reverse && c >= 0) || (i.reverse && c < 0) {
			// We're at the end key, or past the end.
			return false
		}
	}
	return true
}

func (i *BadgerBackendIterator) Key() []byte {
	if !i.Valid() {
		panic("iterator is invalid")
	}
	// Note that we don't use KeyCopy, so this is only valid until the next
	// call to Next.
	return i.iter.Item().KeyCopy(nil)
}

func (i *BadgerBackendIterator) Value() []byte {
	if !i.Valid() {
		panic("iterator is invalid")
	}
	val, err := i.iter.Item().ValueCopy(nil)
	if err != nil {
		i.lastErr = err
	}
	return val
}

type versionManager struct {
	*version.VersionManager
	vmap   map[uint64]uint64
	lastTs uint64
}

func (vm *versionManager) versionTs(ver uint64) (uint64, bool) {
	ts, has := vm.vmap[ver]
	return ts, has
}

// updateCommitTs increments the lastTs if equal to readts.
func (vm *versionManager) updateCommitTs(readts uint64) {
	if vm.lastTs == readts {
		vm.lastTs++
	}
}

// Atomically accesses the last commit timestamp used as a version marker.
func (vm *versionManager) lastCommitTs() uint64 {
	return atomic.LoadUint64(&vm.lastTs)
}

func (vm *versionManager) Copy() *versionManager {
	vmap := map[uint64]uint64{}
	for ver, ts := range vm.vmap {
		vmap[ver] = ts
	}
	return &versionManager{
		VersionManager: vm.VersionManager.Copy(),
		vmap:           vmap,
		lastTs:         vm.lastCommitTs(),
	}
}

func (vm *versionManager) Save(target uint64) (uint64, error) {
	id, err := vm.VersionManager.Save(target)
	if err != nil {
		return 0, err
	}
	vm.vmap[id] = vm.lastTs // non-atomic, already guarded by the vmgr mutex
	return id, nil
}

func (vm *versionManager) Delete(target uint64) {
	vm.VersionManager.Delete(target)
	delete(vm.vmap, target)
}
