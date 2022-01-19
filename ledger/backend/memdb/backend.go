package memdb

import (
	"sync"

	"github.com/google/btree"

	tplgcmm "github.com/TopiaNetwork/topia/ledger/backend/common"
	tplog "github.com/TopiaNetwork/topia/log"
)

const (
	// The approximate number of items and children per B-tree node. Tuned with benchmarks.
	bTreeDegree = 32
)

type MemBackend struct {
	log   tplog.Logger
	name  string
	mtx   sync.RWMutex
	btree *btree.BTree
}

func NewLMemBackend(log tplog.Logger, name string, path string, cacheSize int) *MemBackend {
	return &MemBackend{
		log:   log,
		name:  name,
		btree: btree.New(bTreeDegree),
	}
}

func (b *MemBackend) Get(bytes []byte, version *uint64) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (b *MemBackend) Has(key []byte, version *uint64) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (b *MemBackend) Set(bytes []byte, bytes2 []byte) error {
	//TODO implement me
	panic("implement me")
}

func (b *MemBackend) SetSync(bytes []byte, bytes2 []byte) error {
	//TODO implement me
	panic("implement me")
}

func (b *MemBackend) Delete(bytes []byte) error {
	//TODO implement me
	panic("implement me")
}

func (b *MemBackend) DeleteSync(bytes []byte) error {
	//TODO implement me
	panic("implement me")
}

func (b *MemBackend) Iterator(start, end []byte, version *uint64) (tplgcmm.Iterator, error) {
	//TODO implement me
	panic("implement me")
}

func (b *MemBackend) ReverseIterator(start, end []byte, version *uint64) (tplgcmm.Iterator, error) {
	//TODO implement me
	panic("implement me")
}

func (b *MemBackend) Close() error {
	//TODO implement me
	panic("implement me")
}

func (b *MemBackend) NewBatch() tplgcmm.Batch {
	//TODO implement me
	panic("implement me")
}

func (b *MemBackend) Print() error {
	//TODO implement me
	panic("implement me")
}

func (b *MemBackend) Stats() map[string]string {
	//TODO implement me
	panic("implement me")
}

func (b *MemBackend) Versions() (tplgcmm.VersionSet, error) {
	//TODO implement me
	panic("implement me")
}

func (b *MemBackend) SaveNextVersion() (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (b *MemBackend) SaveVersion(uint64) error {
	//TODO implement me
	panic("implement me")
}

func (b *MemBackend) DeleteVersion(uint64) error {
	//TODO implement me
	panic("implement me")
}

func (b *MemBackend) LastVersion() uint64 {
	//TODO implement me
	panic("implement me")
}

func (b *MemBackend) Commit() error {
	//TODO implement me
	panic("implement me")
}
