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

func (m *MemBackend) Reader() tplgcmm.DBReader {
	//TODO implement me
	panic("implement me")
}

func (m *MemBackend) ReaderAt(u uint64) (tplgcmm.DBReader, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MemBackend) ReadWriter() tplgcmm.DBReadWriter {
	//TODO implement me
	panic("implement me")
}

func (m *MemBackend) Writer() tplgcmm.DBWriter {
	//TODO implement me
	panic("implement me")
}

func (m *MemBackend) PendingTxCount() int32 {
	//TODO implement me
	panic("implement me")
}

func (m *MemBackend) Versions() (tplgcmm.VersionSet, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MemBackend) SaveNextVersion() (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MemBackend) SaveVersion(u uint64) error {
	//TODO implement me
	panic("implement me")
}

func (m *MemBackend) DeleteVersion(u uint64) error {
	//TODO implement me
	panic("implement me")
}

func (m *MemBackend) Revert() error {
	//TODO implement me
	panic("implement me")
}

func (m *MemBackend) Close() error {
	//TODO implement me
	panic("implement me")
}
