package meta

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/TopiaNetwork/topia/ledger/backend"
	tplgcmm "github.com/TopiaNetwork/topia/ledger/backend/common"
	tplgstore "github.com/TopiaNetwork/topia/ledger/store"
	tplog "github.com/TopiaNetwork/topia/log"
)

type IterMetaDataCBFunc func(key []byte, val []byte)

type MetaStore interface {
	tplgstore.Store

	GetData(name string, key []byte) ([]byte, error)

	LatestVersion() (uint64, error)

	IterateAllMetaDataCB(name string, iterCBFunc IterMetaDataCBFunc) error
}

type metaStore struct {
	log       tplog.Logger
	backend   backend.Backend
	backendRW tplgcmm.DBReadWriter
	lock      sync.RWMutex
	storeMap  map[string]*metaStoreSpecification
}

func NewMetaStore(log tplog.Logger, backendDB backend.Backend) MetaStore {
	return &metaStore{
		log:       log,
		backend:   backendDB,
		backendRW: backendDB.ReadWriter(),
		storeMap:  make(map[string]*metaStoreSpecification),
	}
}

func (m *metaStore) AddNamedStateStore(name string, cacheSize int) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.storeMap[name]; ok {
		return nil
	}

	m.storeMap[name] = newMetaStoreSpecification(m.log, m.backendRW, name, cacheSize)

	return nil
}

func (m *metaStore) Root(name string) ([]byte, error) {
	panic("Unsupported method root for metaStore")
}

func (m *metaStore) Put(name string, key []byte, value []byte) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if ss, ok := m.storeMap[name]; ok {
		return ss.Set(key, value)
	}

	return fmt.Errorf("Can't find the responding meta store: name=%s", name)
}

func (m *metaStore) Delete(name string, key []byte) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if ss, ok := m.storeMap[name]; ok {
		return ss.Delete(key)
	}

	return fmt.Errorf("Can't find the responding meta store: name=%s", name)
}

func (m *metaStore) Exists(name string, key []byte) (bool, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if ss, ok := m.storeMap[name]; ok {
		return ss.Has(key)
	}

	return false, fmt.Errorf("Can't find the responding meta store: name=%s", name)
}

func (m *metaStore) Update(name string, key []byte, value []byte) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if ss, ok := m.storeMap[name]; ok {
		return ss.Set(key, value)
	}

	return fmt.Errorf("Can't find the responding meta store: name=%s", name)
}

func (m *metaStore) GetData(name string, key []byte) ([]byte, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if ss, ok := m.storeMap[name]; ok {
		return ss.Get(key)
	}

	return nil, fmt.Errorf("Can't find the responding meta store: name=%s", name)
}

func (m *metaStore) IterateAllMetaDataCB(name string, iterCBFunc IterMetaDataCBFunc) error {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if ss, ok := m.storeMap[name]; ok {
		return ss.IterateAllStateDataCB(iterCBFunc)
	}

	return fmt.Errorf("Can't find the responding meta store: name=%s", name)
}

func (m *metaStore) Clone(other tplgstore.Store) error {
	m.lock.RLock()
	defer m.lock.RUnlock()

	otherSS, ok := other.(*metaStore)
	if !ok {
		return fmt.Errorf("Expected Other is metaStore，actual %s", reflect.TypeOf(other).Name())
	}

	dataIt, err := m.backendRW.Iterator(nil, nil)

	if err != nil {
		return err
	}
	defer dataIt.Close()

	for dataIt.Next() {
		otherSS.backendRW.Set(dataIt.Key(), dataIt.Value())
	}

	return nil
}

func (m *metaStore) LatestVersion() (uint64, error) {
	verSet, err := m.backend.Versions()
	if err != nil {
		m.log.Errorf("Can't get version set: %v", err)
		return 0, err
	}

	return verSet.Last(), nil
}

func (m *metaStore) Commit() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.backendRW.Commit()

	lastVer, _ := m.LatestVersion()
	m.backend.SaveVersion(lastVer + 1)

	return nil
}

func (m *metaStore) Rollback() error {
	return m.backend.Revert()
}

func (m *metaStore) Stop() error {
	return m.backendRW.Discard()
}

func (m *metaStore) Close() error {
	return m.backend.Close()
}
