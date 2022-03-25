package backend

import (
	"bytes"
	"fmt"

	tpcmm "github.com/TopiaNetwork/topia/common"
	tplgcmm "github.com/TopiaNetwork/topia/ledger/backend/common"
)

func prefixed(prefix, key []byte) []byte {
	return append(tpcmm.BytesCopy(prefix), key...)
}

func endIter(bz []byte) (ret []byte) {
	if len(bz) == 0 {
		panic("endIter expects non-zero bz length")
	}
	ret = tpcmm.BytesCopy(bz)
	for i := len(bz) - 1; i >= 0; i-- {
		if ret[i] < byte(0xFF) {
			ret[i]++
			return
		}
		ret[i] = byte(0x00)
		if i == 0 {
			// Overflow
			return nil
		}
	}
	return nil
}

// IteratePrefix is a convenience function for iterating over a key domain
// restricted by prefix.
func IteratePrefix(dbr tplgcmm.DBReader, prefix []byte) (tplgcmm.Iterator, error) {
	var start, end []byte
	if len(prefix) != 0 {
		start = prefix
		end = endIter(prefix)
	}
	itr, err := dbr.Iterator(start, end)
	if err != nil {
		return nil, err
	}
	return itr, nil
}

type prefixDBIterator struct {
	prefix []byte
	start  []byte
	end    []byte
	source tplgcmm.Iterator
	err    error
}

func newPrefixIterator(prefix, start, end []byte, source tplgcmm.Iterator) *prefixDBIterator {
	return &prefixDBIterator{
		prefix: prefix,
		start:  start,
		end:    end,
		source: source,
	}
}

func (itr *prefixDBIterator) Domain() (start, end []byte) {
	return itr.start, itr.end
}

func (itr *prefixDBIterator) valid() bool {
	if itr.err != nil {
		return false
	}

	key := itr.source.Key()
	if len(key) < len(itr.prefix) || !bytes.Equal(key[:len(itr.prefix)], itr.prefix) {
		itr.err = fmt.Errorf("received invalid key from backend: %x (expected prefix %x)",
			key, itr.prefix)
		return false
	}

	return true
}

func (itr *prefixDBIterator) Next() bool {
	if !itr.source.Next() {
		return false
	}
	key := itr.source.Key()
	if !bytes.HasPrefix(key, itr.prefix) {
		return false
	}
	// Empty keys are not allowed, so if a key exists in the database that exactly matches the
	// prefix we need to skip it.
	if bytes.Equal(key, itr.prefix) {
		return itr.Next()
	}
	return true
}

func (itr *prefixDBIterator) Key() []byte {
	itr.assertIsValid()
	key := itr.source.Key()
	return key[len(itr.prefix):] // we have checked the key in Valid()
}

// Value implements Iterator.
func (itr *prefixDBIterator) Value() []byte {
	itr.assertIsValid()
	return itr.source.Value()
}

func (itr *prefixDBIterator) Error() error {
	if err := itr.source.Error(); err != nil {
		return err
	}
	return itr.err
}

func (itr *prefixDBIterator) Close() error {
	return itr.source.Close()
}

func (itr *prefixDBIterator) assertIsValid() {
	if !itr.valid() {
		panic("iterator is invalid")
	}
}

type BackendRPrefixed struct {
	prefix   []byte
	backendR tplgcmm.DBReader
}

type BackendWPrefixed struct {
	prefix   []byte
	backendW tplgcmm.DBWriter
}

type BackendRWPrefixed struct {
	prefix    []byte
	backendRW tplgcmm.DBReadWriter
}

func NewBackendRPrefixed(prefix []byte, rw tplgcmm.DBReader) tplgcmm.DBReader {
	return &BackendRPrefixed{
		prefix:   prefix,
		backendR: rw,
	}
}

func NewBackendWPrefixed(prefix []byte, rw tplgcmm.DBWriter) tplgcmm.DBWriter {
	return &BackendWPrefixed{
		prefix:   prefix,
		backendW: rw,
	}
}

func NewBackendRWPrefixed(prefix []byte, rw tplgcmm.DBReadWriter) tplgcmm.DBReadWriter {
	return &BackendRWPrefixed{
		prefix:    prefix,
		backendRW: rw,
	}
}

func (rp *BackendRPrefixed) Get(key []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, tplgcmm.ErrKeyEmpty
	}
	return rp.backendR.Get(prefixed(rp.prefix, key))
}

func (rp *BackendRPrefixed) Has(key []byte) (bool, error) {
	if len(key) == 0 {
		return false, tplgcmm.ErrKeyEmpty
	}
	return rp.backendR.Has(prefixed(rp.prefix, key))
}

func (rp *BackendRPrefixed) Iterator(start, end []byte) (tplgcmm.Iterator, error) {
	if (start != nil && len(start) == 0) || (end != nil && len(end) == 0) {
		return nil, tplgcmm.ErrKeyEmpty
	}

	var pend []byte
	if end == nil {
		pend = endIter(rp.prefix)
	} else {
		pend = prefixed(rp.prefix, end)
	}
	itr, err := rp.backendR.Iterator(prefixed(rp.prefix, start), pend)
	if err != nil {
		return nil, err
	}
	return newPrefixIterator(rp.prefix, start, end, itr), nil
}

func (rp *BackendRPrefixed) ReverseIterator(start, end []byte) (tplgcmm.Iterator, error) {
	if (start != nil && len(start) == 0) || (end != nil && len(end) == 0) {
		return nil, tplgcmm.ErrKeyEmpty
	}

	var pend []byte
	if end == nil {
		pend = endIter(rp.prefix)
	} else {
		pend = prefixed(rp.prefix, end)
	}
	ritr, err := rp.backendR.ReverseIterator(prefixed(rp.prefix, start), pend)
	if err != nil {
		return nil, err
	}
	return newPrefixIterator(rp.prefix, start, end, ritr), nil
}

func (rp *BackendRPrefixed) Discard() error { return rp.backendR.Discard() }

func (wp *BackendWPrefixed) Set(key []byte, value []byte) error {
	if len(key) == 0 {
		return tplgcmm.ErrKeyEmpty
	}
	return wp.backendW.Set(prefixed(wp.prefix, key), value)
}

func (wp *BackendWPrefixed) Delete(key []byte) error {
	if len(key) == 0 {
		return tplgcmm.ErrKeyEmpty
	}
	return wp.backendW.Delete(prefixed(wp.prefix, key))
}

func (wp *BackendWPrefixed) Commit() error { return wp.backendW.Commit() }

func (wp *BackendWPrefixed) Discard() error { return wp.backendW.Discard() }

func (rwp *BackendRWPrefixed) Get(bytes []byte) ([]byte, error) {
	return NewBackendRPrefixed(rwp.prefix, rwp.backendRW).Get(bytes)
}

func (rwp *BackendRWPrefixed) Has(key []byte) (bool, error) {
	return NewBackendRPrefixed(rwp.prefix, rwp.backendRW).Has(key)
}

func (rwp *BackendRWPrefixed) Set(bytes []byte, bytes2 []byte) error {
	return NewBackendWPrefixed(rwp.prefix, rwp.backendRW).Set(bytes, bytes2)
}

func (rwp *BackendRWPrefixed) Delete(bytes []byte) error {
	return NewBackendWPrefixed(rwp.prefix, rwp.backendRW).Delete(bytes)
}

func (rwp *BackendRWPrefixed) Iterator(start, end []byte) (tplgcmm.Iterator, error) {
	return NewBackendRPrefixed(rwp.prefix, rwp.backendRW).Iterator(start, end)
}

func (rwp *BackendRWPrefixed) ReverseIterator(start, end []byte) (tplgcmm.Iterator, error) {
	return NewBackendRPrefixed(rwp.prefix, rwp.backendRW).ReverseIterator(start, end)
}

func (wp *BackendRWPrefixed) Commit() error { return wp.backendRW.Commit() }

func (wp *BackendRWPrefixed) Discard() error { return wp.backendRW.Discard() }
