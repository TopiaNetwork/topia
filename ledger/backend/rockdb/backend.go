package rockdb

import (
	"os"
	"path/filepath"
	"runtime"

	lru "github.com/hashicorp/golang-lru"
	"github.com/tecbot/gorocksdb"

	tplgcmm "github.com/TopiaNetwork/topia/ledger/backend/common"
	tplog "github.com/TopiaNetwork/topia/log"
)

type RocksdbBackend struct {
	log    tplog.Logger
	name   string
	cache  *lru.ARCCache
	db     *gorocksdb.DB
	ro     *gorocksdb.ReadOptions
	wo     *gorocksdb.WriteOptions
	woSync *gorocksdb.WriteOptions
}

func NewRocksdbBackend(log tplog.Logger, name string, path string, cacheSize int) *RocksdbBackend {
	pathWithName := filepath.Join(path, name+".db")
	if err := os.MkdirAll(pathWithName, 0755); err != nil {
		log.Panicf("can't change the path %s to 0755", pathWithName)
		return nil
	}

	bbto := gorocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetBlockCache(gorocksdb.NewLRUCache(1 << 30))
	bbto.SetFilterPolicy(gorocksdb.NewBloomFilter(10))

	opts := gorocksdb.NewDefaultOptions()
	opts.SetBlockBasedTableFactory(bbto)
	opts.SetCreateIfMissing(true)
	opts.IncreaseParallelism(runtime.NumCPU())
	// 1.5GB maximum memory use for writebuffer.
	opts.OptimizeLevelStyleCompaction(512 * 1024 * 1024)

	db, err := gorocksdb.OpenDb(opts, filepath.Join(path, name+".db"))
	if err != nil {
		log.Panicf("Create leveldb %s error %v, dbPath=%s", name, err, path)
		return nil
	}

	ro := gorocksdb.NewDefaultReadOptions()
	wo := gorocksdb.NewDefaultWriteOptions()
	woSync := gorocksdb.NewDefaultWriteOptions()
	woSync.SetSync(true)

	cache, _ := lru.NewARC(cacheSize)
	return &RocksdbBackend{
		log:    log,
		name:   name,
		cache:  cache,
		db:     db,
		ro:     ro,
		wo:     wo,
		woSync: woSync,
	}
}

func (b *RocksdbBackend) Get(bytes []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (b *RocksdbBackend) Has(key []byte) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (b *RocksdbBackend) Set(bytes []byte, bytes2 []byte) error {
	//TODO implement me
	panic("implement me")
}

func (b *RocksdbBackend) SetSync(bytes []byte, bytes2 []byte) error {
	//TODO implement me
	panic("implement me")
}

func (b *RocksdbBackend) Delete(bytes []byte) error {
	//TODO implement me
	panic("implement me")
}

func (b *RocksdbBackend) DeleteSync(bytes []byte) error {
	//TODO implement me
	panic("implement me")
}

func (b *RocksdbBackend) Iterator(start, end []byte) (tplgcmm.Iterator, error) {
	//TODO implement me
	panic("implement me")
}

func (b *RocksdbBackend) ReverseIterator(start, end []byte) (tplgcmm.Iterator, error) {
	//TODO implement me
	panic("implement me")
}

func (b *RocksdbBackend) Close() error {
	//TODO implement me
	panic("implement me")
}

func (b *RocksdbBackend) NewBatch() tplgcmm.Batch {
	//TODO implement me
	panic("implement me")
}

func (b *RocksdbBackend) Print() error {
	//TODO implement me
	panic("implement me")
}

func (b *RocksdbBackend) Stats() map[string]string {
	//TODO implement me
	panic("implement me")
}
