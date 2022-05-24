package backend

import (
	"github.com/TopiaNetwork/topia/ledger/backend/badger"
	tplgcmm "github.com/TopiaNetwork/topia/ledger/backend/common"
	"github.com/TopiaNetwork/topia/ledger/backend/leveldb"
	"github.com/TopiaNetwork/topia/ledger/backend/memdb"
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
	// Reader opens a read-only transaction at the current working version.
	Reader() tplgcmm.DBReader

	// ReaderAt opens a read-only transaction at a specified version.
	// Returns ErrVersionDoesNotExist for invalid versions.
	ReaderAt(uint64) (tplgcmm.DBReader, error)

	// ReadWriter opens a read-write transaction at the current version.
	ReadWriter() tplgcmm.DBReadWriter

	// Writer opens a write-only transaction at the current version.
	Writer() tplgcmm.DBWriter

	//PendingTxCount return pending transaction count
	PendingTxCount() int32

	// Versions returns all saved versions as an immutable set which is safe for concurrent access.
	Versions() (tplgcmm.VersionSet, error)

	// SaveNextVersion saves the current contents of the database and returns the next version ID,
	// which will be `Versions().Last()+1`.
	// Returns an error if any open DBWriter transactions exist.
	// TODO: rename to something more descriptive?
	SaveNextVersion() (uint64, error)

	// SaveVersion attempts to save database at a specific version ID, which must be greater than or
	// equal to what would be returned by `SaveNextVersion`.
	// Returns an error if any open DBWriter transactions exist.
	SaveVersion(uint64) error

	// DeleteVersion deletes a saved version. Returns ErrVersionDoesNotExist for invalid versions.
	DeleteVersion(uint64) error

	// Revert reverts the DB state to the last saved version; if none exist, this clears the DB.
	// Returns an error if any open DBWriter transactions exist.
	Revert() error

	// Close closes the database connection.
	Close() error
}

func NewBackend(backendType BackendType, log tplog.Logger, path string, name string) Backend {
	bLog := tplog.CreateModuleLogger(tplogcmm.InfoLevel, "LedgerBackend", log)

	switch backendType {
	case BackendType_Leveldb:
		return leveldb.NewLeveldbBackend(bLog, name, path, DefaultCacheSize)
	case BackendType_Rocksdb:
		bLog.Panic("Don't support rocksdb now")
		//return rocksdb.NewRocksdbBackend(bLog, path, name, DefaultCacheSize)
	case BackendType_Badger:
		return badger.NewBadgerBackend(bLog, name, path, DefaultCacheSize)
	case BackendType_Memdb:
		return memdb.NewMemDBBackend(bLog)
	default:
		bLog.Panicf("Invalid backend type %d", backendType)
	}

	return nil
}
