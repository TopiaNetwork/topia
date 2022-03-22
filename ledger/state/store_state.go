package state

import (
	"errors"
	"fmt"
	"sync"

	"github.com/TopiaNetwork/topia/ledger/backend"
	tplgcmm "github.com/TopiaNetwork/topia/ledger/backend/common"
	tplog "github.com/TopiaNetwork/topia/log"
)

type Flag int

const (
	Flag_Unknown   Flag = 0x00
	Flag_ReadOnly       = 0x01
	Flag_WriteOnly      = 0x10
)

type StateStore interface {
	AddNamedStateStore(name string) error

	Put(name string, key []byte, value []byte) error

	Delete(name string, key []byte) error

	Exists(name string, key []byte) (bool, error)

	Update(name string, key []byte, value []byte) error

	GetState(name string, key []byte) ([]byte, []byte, error)

	StateLatestVersion() (uint64, error)

	StateVersions() ([]uint64, error)

	Commit() error

	Rollback() error

	Stop() error

	Close() error
}

type stateStore struct {
	log       tplog.Logger
	backend   backend.Backend
	backendR  tplgcmm.DBReader
	backendRW tplgcmm.DBReadWriter
	lock      sync.RWMutex
	storeMap  map[string]*StateStoreComposition
}

func NewStateStore(log tplog.Logger, backendDB backend.Backend, flag Flag) StateStore {
	if Flag_ReadOnly|Flag_WriteOnly == flag {
		return &stateStore{
			log:       log,
			backend:   backendDB,
			backendRW: backendDB.ReadWriter(),
			storeMap:  make(map[string]*StateStoreComposition),
		}
	} else if Flag_ReadOnly == flag {
		return &stateStore{
			log:      log,
			backend:  backendDB,
			backendR: backendDB.Reader(),
			storeMap: make(map[string]*StateStoreComposition),
		}
	} else {
		log.Panicf("Invalid state store flag")
		return nil
	}
}

func (m *stateStore) AddNamedStateStore(name string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.storeMap[name]; ok {
		return nil
	}

	var ss *StateStoreComposition
	if m.backendR != nil {
		ss = newStateStoreCompositionReadOnly(m.log, m.backendR, name)
	} else {
		ss = newStateStoreComposition(m.log, m.backendRW, name)
	}
	m.storeMap[name] = ss

	return nil
}

func (m *stateStore) Put(name string, key []byte, value []byte) error {
	if m.backendR != nil {
		return errors.New("Can't put because of read only state store")
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	if ss, ok := m.storeMap[name]; ok {
		return ss.Put(key, value)
	}

	return fmt.Errorf("Can't find the responding state store: name=%s", name)
}

func (m *stateStore) Delete(name string, key []byte) error {
	if m.backendR != nil {
		return errors.New("Can't delete because of read only state store")
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	if ss, ok := m.storeMap[name]; ok {
		return ss.Delete(key)
	}

	return fmt.Errorf("Can't find the responding state store: name=%s", name)
}

func (m *stateStore) Exists(name string, key []byte) (bool, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if ss, ok := m.storeMap[name]; ok {
		return ss.Exists(key)
	}

	return false, fmt.Errorf("Can't find the responding state store: name=%s", name)
}

func (m *stateStore) Update(name string, key []byte, value []byte) error {
	if m.backendR != nil {
		return errors.New("Can't update because of read only state store")
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	if ss, ok := m.storeMap[name]; ok {
		return ss.Update(key, value)
	}

	return fmt.Errorf("Can't find the responding state store: name=%s", name)
}

func (m *stateStore) GetState(name string, key []byte) ([]byte, []byte, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if ss, ok := m.storeMap[name]; ok {
		return ss.GetState(key)
	}

	return nil, nil, fmt.Errorf("Can't find the responding state store: name=%s", name)
}

func (m *stateStore) StateLatestVersion() (uint64, error) {
	verSet, err := m.backend.Versions()
	if err != nil {
		m.log.Errorf("Can't get version set: %v", err)
		return 0, err
	}

	return verSet.Last(), nil
}

func (m *stateStore) StateVersions() ([]uint64, error) {
	verSet, err := m.backend.Versions()
	if err != nil {
		m.log.Errorf("Can't get version set: %v", err)
		return nil, err
	}

	var versions []uint64
	verIt := verSet.Iterator()
	for verIt.Next() {
		ver := verIt.Value()
		versions = append(versions, ver)
	}

	return versions, err
}

func (m *stateStore) Commit() error {
	if m.backendR != nil {
		return errors.New("Can't commit because of read only state store")
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	m.backendRW.Commit()

	lastVer, _ := m.StateLatestVersion()
	m.backend.SaveVersion(lastVer + 1)

	return nil
}

func (m *stateStore) Rollback() error {
	if m.backendR != nil {
		return errors.New("Can't rollback because of read only state store")
	}

	return m.backend.Revert()
}

func (m *stateStore) Stop() error {
	if m.backendR != nil {
		return m.backendR.Discard()
	}

	return m.backendRW.Discard()
}

func (m *stateStore) Close() error {
	return m.backend.Close()
}
