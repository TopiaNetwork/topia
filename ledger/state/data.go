package state

import (
	"errors"
	tplgcmm "github.com/TopiaNetwork/topia/ledger/backend/common"
)

type stateData struct {
	name      string
	backendR  tplgcmm.DBReader
	backendRW tplgcmm.DBReadWriter
}

func newStateData(name string, backendRW tplgcmm.DBReadWriter) *stateData {
	return &stateData{
		name:      name,
		backendRW: backendRW,
	}
}

func newStateDataReadonly(name string, backendR tplgcmm.DBReader) *stateData {
	return &stateData{
		name:     name,
		backendR: backendR,
	}
}

func (s *stateData) Get(key []byte) ([]byte, error) {
	if s.backendR != nil {
		return s.backendR.Get(key)
	} else {
		return s.backendRW.Get(key)
	}
}

func (s *stateData) Set(key []byte, value []byte) error {
	if s.backendR != nil {
		return errors.New("Read only state data store")
	}

	return s.backendRW.Set(key, value)
}

func (s *stateData) Delete(key []byte) error {
	if s.backendR != nil {
		return errors.New("Read only state data store")
	}

	return s.backendRW.Delete(key)
}

func (s *stateData) Has(key []byte) (bool, error) {
	if s.backendR != nil {
		return s.backendR.Has(key)
	} else {
		return s.backendRW.Has(key)
	}
}

func (s *stateData) Iterator(start, end []byte) (tplgcmm.Iterator, error) {
	if s.backendR != nil {
		return s.backendR.Iterator(start, end)
	} else {
		return s.backendRW.Iterator(start, end)
	}
}

func (s *stateData) ReverseIterator(start, end []byte, version *uint64) (tplgcmm.Iterator, error) {
	if s.backendR != nil {
		return s.backendR.ReverseIterator(start, end)
	} else {
		return s.backendRW.ReverseIterator(start, end)
	}
}
