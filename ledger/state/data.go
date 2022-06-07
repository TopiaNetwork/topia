package state

import (
	"errors"

	"github.com/coocood/freecache"

	tplgcmm "github.com/TopiaNetwork/topia/ledger/backend/common"
)

type stateData struct {
	name      string
	backendR  tplgcmm.DBReader
	backendRW tplgcmm.DBReadWriter
	cache     *freecache.Cache
}

func newStateData(name string, backendRW tplgcmm.DBReadWriter, cacheSize int) *stateData {
	cache := (*freecache.Cache)(nil)
	if cacheSize > 0 {
		cache = freecache.NewCache(cacheSize)
	} else {
		panic("CacheSize must be bigger than 0 when creating state data!")
	}

	return &stateData{
		name:      name,
		backendRW: backendRW,
		cache:     cache,
	}
}

func newStateDataReadonly(name string, backendR tplgcmm.DBReader) *stateData {
	return &stateData{
		name:     name,
		backendR: backendR,
	}
}

func (s *stateData) Get(key []byte) ([]byte, error) {
	if val, err := s.cache.Get(key); err == nil {
		return val, nil
	}

	if s.backendR != nil {
		return s.backendR.Get(key)
	} else {
		return s.backendRW.Get(key)
	}
}

func (s *stateData) addToCache(key, val []byte) {
	if err := s.cache.Set(key, val, -1); err != nil {
		s.cache.Del(key)
	}
}

func (s *stateData) Set(key []byte, value []byte) error {
	if s.backendR != nil {
		return errors.New("Read only state data store")
	}

	if err := s.backendRW.Set(key, value); err != nil {
		return err
	}

	s.addToCache(key, value)

	return nil
}

func (s *stateData) Delete(key []byte) error {
	if s.backendR != nil {
		return errors.New("Read only state data store")
	}

	s.cache.Del(key)

	return s.backendRW.Delete(key)
}

func (s *stateData) Has(key []byte) (bool, error) {
	if _, err := s.cache.Get(key); err == nil {
		return true, nil
	}

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
