package badger

import (
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger"
	lru "github.com/hashicorp/golang-lru"

	tplgcmm "github.com/TopiaNetwork/topia/ledger/backend/common"
	tplog "github.com/TopiaNetwork/topia/log"
)

type BadgerBackend struct {
	log   tplog.Logger
	name  string
	cache *lru.ARCCache
	db    *badger.DB
}

func NewRockdbBackend(log tplog.Logger, name string, path string, cacheSize int) *BadgerBackend {
	pathWithName := filepath.Join(path, name+".db")
	if err := os.MkdirAll(pathWithName, 0755); err != nil {
		log.Panicf("can't change the path %s to 0755", pathWithName)
		return nil
	}

	opts := badger.DefaultOptions(pathWithName)
	opts.SyncWrites = false // note that we have Sync methods
	opts.Logger = nil       // badger is too chatty by default

	db, err := badger.Open(opts)
	if err != nil {
		log.Panicf("can't open badger: path=%s, err=%v", pathWithName, err)
		return nil
	}

	cache, _ := lru.NewARC(cacheSize)
	return &BadgerBackend{
		log:   log,
		name:  name,
		cache: cache,
		db:    db,
	}
}

func (b *BadgerBackend) Get(bytes []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (b *BadgerBackend) Has(key []byte) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (b *BadgerBackend) Set(bytes []byte, bytes2 []byte) error {
	//TODO implement me
	panic("implement me")
}

func (b *BadgerBackend) SetSync(bytes []byte, bytes2 []byte) error {
	//TODO implement me
	panic("implement me")
}

func (b *BadgerBackend) Delete(bytes []byte) error {
	//TODO implement me
	panic("implement me")
}

func (b *BadgerBackend) DeleteSync(bytes []byte) error {
	//TODO implement me
	panic("implement me")
}

func (b *BadgerBackend) Iterator(start, end []byte) (tplgcmm.Iterator, error) {
	//TODO implement me
	panic("implement me")
}

func (b *BadgerBackend) ReverseIterator(start, end []byte) (tplgcmm.Iterator, error) {
	//TODO implement me
	panic("implement me")
}

func (b *BadgerBackend) Close() error {
	//TODO implement me
	panic("implement me")
}

func (b *BadgerBackend) NewBatch() tplgcmm.Batch {
	//TODO implement me
	panic("implement me")
}

func (b *BadgerBackend) Print() error {
	//TODO implement me
	panic("implement me")
}

func (b *BadgerBackend) Stats() map[string]string {
	//TODO implement me
	panic("implement me")
}

