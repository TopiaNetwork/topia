package leveldb

import (
	tplgcmm "github.com/TopiaNetwork/topia/ledger/backend/common"
	"os"
	"path/filepath"

	lru "github.com/hashicorp/golang-lru"
	"github.com/syndtr/goleveldb/leveldb"

	tplog "github.com/TopiaNetwork/topia/log"
)

type LeveldbBackend struct {
	log   tplog.Logger
	name  string
	cache *lru.ARCCache
	db    *leveldb.DB
}

func NewLeveldbBackend(log tplog.Logger, name string, path string, cacheSize int) *LeveldbBackend {
	pathWithName := filepath.Join(path, name+".db")
	if err := os.MkdirAll(pathWithName, 0755); err != nil {
		log.Panicf("can't change the path %s to 0755", pathWithName)
		return nil
	}

	db, err := leveldb.OpenFile(filepath.Join(path, name+".db"), nil)
	if err != nil {
		log.Panicf("Create leveldb %s error %v, dbPath=%s", name, err, pathWithName)
		return nil
	}

	cache, _ := lru.NewARC(cacheSize)
	return &LeveldbBackend{
		log:   log,
		name:  name,
		cache: cache,
		db:    db,
	}
}

func (l *LeveldbBackend) Reader() tplgcmm.DBReader {
	//TODO implement me
	panic("implement me")
}

func (l *LeveldbBackend) ReaderAt(u uint64) (tplgcmm.DBReader, error) {
	//TODO implement me
	panic("implement me")
}

func (l *LeveldbBackend) ReadWriter() tplgcmm.DBReadWriter {
	//TODO implement me
	panic("implement me")
}

func (l *LeveldbBackend) Writer() tplgcmm.DBWriter {
	//TODO implement me
	panic("implement me")
}

func (l *LeveldbBackend) PendingTxCount() int32 {
	//TODO implement me
	panic("implement me")
}

func (l *LeveldbBackend) Versions() (tplgcmm.VersionSet, error) {
	//TODO implement me
	panic("implement me")
}

func (l *LeveldbBackend) SaveNextVersion() (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (l *LeveldbBackend) SaveVersion(u uint64) error {
	//TODO implement me
	panic("implement me")
}

func (l *LeveldbBackend) DeleteVersion(u uint64) error {
	//TODO implement me
	panic("implement me")
}

func (l *LeveldbBackend) Revert() error {
	//TODO implement me
	panic("implement me")
}

func (l *LeveldbBackend) Close() error {
	//TODO implement me
	panic("implement me")
}
