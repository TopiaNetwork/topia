package state

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/TopiaNetwork/topia/ledger/backend"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
)

type stateStore struct {
	log      tplog.Logger
	backend  backend.Backend
	lock     sync.Mutex
	storeMap map[string]*StateStoreComposition
}

func NewStateStore(log tplog.Logger, rootPath string, backendType backend.BackendType) *stateStore {
	bsLog := tplog.CreateModuleLogger(tplogcmm.InfoLevel, "StateStore", log)
	backendDB := backend.NewBackend(backendType, bsLog, filepath.Join(rootPath, "statestore"), "statestore")
	return &stateStore{
		log:      log,
		backend:  backendDB,
		storeMap: make(map[string]*StateStoreComposition),
	}
}

func (m *stateStore) CreateNamedStateStore(name string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.storeMap[name]; ok {
		return nil
	}

	ss := newStateStoreComposition(m.log, m.backend, name)
	m.storeMap[name] = ss

	return nil
}

func (m *stateStore) Put(name string, key []byte, value []byte) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if ss, ok := m.storeMap[name]; ok {
		return ss.Put(key, value)
	}

	return fmt.Errorf("Can't find the responding state store: name=%s", name)
}

func (m *stateStore) Delete(name string, key []byte) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if ss, ok := m.storeMap[name]; ok {
		return ss.Delete(key)
	}

	return fmt.Errorf("Can't find the responding state store: name=%s", name)
}

func (m *stateStore) Exists(name string, key []byte) (bool, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if ss, ok := m.storeMap[name]; ok {
		return ss.Exists(key)
	}

	return false, fmt.Errorf("Can't find the responding state store: name=%s", name)
}

func (m *stateStore) Update(name string, key []byte, value []byte) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if ss, ok := m.storeMap[name]; ok {
		return ss.Update(key, value)
	}

	return fmt.Errorf("Can't find the responding state store: name=%s", name)
}

func (m *stateStore) GetState(name string, key []byte) ([]byte, []byte, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if ss, ok := m.storeMap[name]; ok {
		return ss.GetState(key)
	}

	return nil, nil, fmt.Errorf("Can't find the responding state store: name=%s", name)
}

func (m *stateStore) Commit() error {
	return m.backend.Commit()
}
