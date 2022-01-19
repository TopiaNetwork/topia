package leveldb

import (
	"os"
	"path/filepath"

	lru "github.com/hashicorp/golang-lru"
	"github.com/syndtr/goleveldb/leveldb"

	tplgcmm "github.com/TopiaNetwork/topia/ledger/backend/common"
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

func (b *LeveldbBackend) Get(bytes []byte, version *uint64) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (b *LeveldbBackend) Has(key []byte, version *uint64) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (b *LeveldbBackend) Set(bytes []byte, bytes2 []byte) error {
	//TODO implement me
	panic("implement me")
}

func (b *LeveldbBackend) SetSync(bytes []byte, bytes2 []byte) error {
	//TODO implement me
	panic("implement me")
}

func (b *LeveldbBackend) Delete(bytes []byte) error {
	//TODO implement me
	panic("implement me")
}

func (b *LeveldbBackend) DeleteSync(bytes []byte) error {
	//TODO implement me
	panic("implement me")
}

func (b *LeveldbBackend) Iterator(start, end []byte, version *uint64) (tplgcmm.Iterator, error) {
	//TODO implement me
	panic("implement me")
}

func (b *LeveldbBackend) ReverseIterator(start, end []byte, version *uint64) (tplgcmm.Iterator, error) {
	//TODO implement me
	panic("implement me")
}

func (b *LeveldbBackend) Close() error {
	//TODO implement me
	panic("implement me")
}

func (b *LeveldbBackend) NewBatch() tplgcmm.Batch {
	//TODO implement me
	panic("implement me")
}

func (b *LeveldbBackend) Print() error {
	//TODO implement me
	panic("implement me")
}

func (b *LeveldbBackend) Stats() map[string]string {
	//TODO implement me
	panic("implement me")
}

func (b *LeveldbBackend) Versions() (tplgcmm.VersionSet, error) {
	//TODO implement me
	panic("implement me")
}

func (b *LeveldbBackend) SaveNextVersion() (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (b *LeveldbBackend) SaveVersion(uint64) error {
	//TODO implement me
	panic("implement me")
}

func (b *LeveldbBackend) DeleteVersion(uint64) error {
	//TODO implement me
	panic("implement me")
}

func (b *LeveldbBackend) LastVersion() uint64 {
	//TODO implement me
	panic("implement me")
}

func (b *LeveldbBackend) Commit() error {
	//TODO implement me
	panic("implement me")
}
