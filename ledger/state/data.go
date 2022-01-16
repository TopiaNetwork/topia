package state

import (
	bacs "github.com/TopiaNetwork/topia/ledger/backend"
	tplgcmm "github.com/TopiaNetwork/topia/ledger/backend/common"
)

type stateData struct {
	name    string
	backend bacs.Backend
}

func newStateData(name string, backend bacs.Backend) *stateData {
	return &stateData{
		name:    name,
		backend: bacs.NewBackendPrefixed([]byte(name), backend),
	}
}

func (s *stateData) Get(key []byte, version *uint64) ([]byte, error) {
	return s.backend.Get(key, version)
}

func (s *stateData) Set(key []byte, value []byte) error {
	return s.backend.Set(key, value)
}

func (s *stateData) Delete(key []byte) error {
	return s.backend.Delete(key)
}

func (s *stateData) SaveVersion(version uint64) error {
	return s.backend.SaveVersion(version)
}

func (s *stateData) DeleteVersion(version uint64) error {
	return s.backend.DeleteVersion(version)
}

func (s *stateData) Has(key []byte, version *uint64) (bool, error) {
	return s.backend.Has(key, version)
}

func (s *stateData) Iterator(start, end []byte, version *uint64) (tplgcmm.Iterator, error) {
	return s.backend.Iterator(start, end, version)
}

func (s *stateData) ReverseIterator(start, end []byte, version *uint64) (tplgcmm.Iterator, error) {
	return s.backend.ReverseIterator(start, end, version)
}
