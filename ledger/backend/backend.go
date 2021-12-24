package backend

import (
	"github.com/TopiaNetwork/topia/ledger/backend/badger"
	tplgcmm "github.com/TopiaNetwork/topia/ledger/backend/common"
	"github.com/TopiaNetwork/topia/ledger/backend/leveldb"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)

type BackendType int

const (
	BackendType_Unknown BackendType = iota
	BackendType_Leveldb
	BackendType_Rocksdb
	BackendType_Badger
	BackendType_Memdb
)

const (
	DefaultCacheSize = 8192
)

type Backend interface {
	Get([]byte) ([]byte, error)

	Has(key []byte) (bool, error)

	Set([]byte, []byte) error

	// SetSync sets the value for the given key, and flushes it to storage before returning.
	SetSync([]byte, []byte) error

	Delete([]byte) error

	// DeleteSync deletes the key, and flushes the delete to storage before returning.
	DeleteSync([]byte) error

	Iterator(start, end []byte) (tplgcmm.Iterator, error)

	// ReverseIterator returns an iterator over a domain of keys, in descending order. The caller
	// must call Close when done. End is exclusive, and start must be less than end. A nil end
	// iterates from the last key (inclusive), and a nil start iterates to the first key (inclusive).
	// Empty keys are not valid.
	ReverseIterator(start, end []byte) (tplgcmm.Iterator, error)

	// Close closes the database connection.
	Close() error

	// NewBatch creates a batch for atomic updates. The caller must call Batch.Close.
	NewBatch() tplgcmm.Batch

	// Print is used for debugging.
	Print() error

	// Stats returns a map of property values for all keys and the size of the cache.
	Stats() map[string]string
}

func NewBackend(backendType BackendType, log tplog.Logger, path string, name string) Backend {
	bLog := tplog.CreateModuleLogger(tplogcmm.InfoLevel, "LedgerBackend", log)

	switch backendType {
	case BackendType_Leveldb:
		return leveldb.NewLeveldbBackend(bLog, path, name, DefaultCacheSize)
	case BackendType_Rocksdb:
		bLog.Panic("Don't support rocksdb now")
		//return rocksdb.NewRocksdbBackend(bLog, path, name, DefaultCacheSize)
	case BackendType_Badger:
		return badger.NewRockdbBackend(bLog, path, name, DefaultCacheSize)
	case BackendType_Memdb:
		return badger.NewRockdbBackend(bLog, path, name, DefaultCacheSize)
	default:
		bLog.Panicf("Invalid backend type %d", backendType)
	}

	return nil
}
