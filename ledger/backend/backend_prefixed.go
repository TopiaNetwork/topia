package backend

import (
	"bytes"
	"errors"
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

type prefixDBIterator struct {
	prefix []byte
	start  []byte
	end    []byte
	source tplgcmm.Iterator
	valid  bool
	err    error
}

func newPrefixIterator(prefix, start, end []byte, source tplgcmm.Iterator) (*prefixDBIterator, error) {
	pitrInvalid := &prefixDBIterator{
		prefix: prefix,
		start:  start,
		end:    end,
		source: source,
		valid:  false,
	}

	if source.Valid() && bytes.Equal(source.Key(), prefix) {
		source.Next()
	}

	if !source.Valid() || !bytes.HasPrefix(source.Key(), prefix) {
		return pitrInvalid, nil
	}

	return &prefixDBIterator{
		prefix: prefix,
		start:  start,
		end:    end,
		source: source,
		valid:  true,
	}, nil
}

func (itr *prefixDBIterator) Domain() (start []byte, end []byte) {
	return itr.start, itr.end
}

func (itr *prefixDBIterator) Valid() bool {
	if !itr.valid || itr.err != nil || !itr.source.Valid() {
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

// Next implements Iterator.
func (itr *prefixDBIterator) Next() {
	itr.assertIsValid()
	itr.source.Next()

	if !itr.source.Valid() || !bytes.HasPrefix(itr.source.Key(), itr.prefix) {
		itr.valid = false

	} else if bytes.Equal(itr.source.Key(), itr.prefix) {
		itr.Next()
	}
}

func (itr *prefixDBIterator) Key() []byte {
	itr.assertIsValid()
	key := itr.source.Key()
	return key[len(itr.prefix):] // we have checked the key in Valid()
}

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
	if !itr.Valid() {
		panic("iterator is invalid")
	}
}

type BackendPrefixed struct {
	prefix  []byte
	backend Backend
}

func NewBackendPrefixed(prefix []byte, backend Backend) Backend {
	return &BackendPrefixed{
		prefix:  prefix,
		backend: backend,
	}
}

func (b *BackendPrefixed) Get(bytes []byte, version *uint64) ([]byte, error) {
	return b.Get(prefixed(b.prefix, bytes), version)
}

func (b *BackendPrefixed) Has(key []byte, version *uint64) (bool, error) {
	return b.Has(prefixed(b.prefix, key), version)
}

func (b *BackendPrefixed) Set(bytes []byte, bytes2 []byte) error {
	return b.Set(prefixed(b.prefix, bytes), bytes2)
}

func (b *BackendPrefixed) SetSync(bytes []byte, bytes2 []byte) error {
	return b.SetSync(prefixed(b.prefix, bytes), bytes2)
}

func (b *BackendPrefixed) Delete(bytes []byte) error {
	return b.Delete(prefixed(b.prefix, bytes))
}

func (b *BackendPrefixed) DeleteSync(bytes []byte) error {
	return b.DeleteSync(prefixed(b.prefix, bytes))
}

func (b *BackendPrefixed) Iterator(start, end []byte, version *uint64) (tplgcmm.Iterator, error) {
	if start == nil || len(start) == 0 || (end != nil && len(end) == 0) {
		return nil, errors.New("invalid input")
	}

	var endN []byte
	if end == nil {
		endN = endIter(b.prefix)
	} else {
		endN = prefixed(b.prefix, end)
	}

	iter, err := b.backend.Iterator(prefixed(b.prefix, start), endN, version)
	if err != nil {
		return nil, err
	}

	return newPrefixIterator(b.prefix, start, end, iter)
}

func (b *BackendPrefixed) ReverseIterator(start, end []byte, version *uint64) (tplgcmm.Iterator, error) {
	if start == nil || len(start) == 0 || (end != nil && len(end) == 0) {
		return nil, errors.New("invalid input")
	}

	var endN []byte
	if end == nil {
		endN = endIter(b.prefix)
	} else {
		endN = prefixed(b.prefix, end)
	}

	iter, err := b.backend.ReverseIterator(prefixed(b.prefix, start), endN, version)
	if err != nil {
		return nil, err
	}

	return newPrefixIterator(b.prefix, start, end, iter)
}

func (b *BackendPrefixed) Close() error {
	return b.backend.Close()
}

func (b *BackendPrefixed) NewBatch() tplgcmm.Batch {
	return b.backend.NewBatch()
}

func (b *BackendPrefixed) Print() error {
	return b.backend.Print()
}

func (b *BackendPrefixed) Stats() map[string]string {
	return b.backend.Stats()
}

func (b *BackendPrefixed) Versions() (tplgcmm.VersionSet, error) {
	return b.backend.Versions()
}

func (b *BackendPrefixed) SaveNextVersion() (uint64, error) {
	return b.backend.SaveNextVersion()
}

func (b *BackendPrefixed) SaveVersion(version uint64) error {
	return b.backend.SaveVersion(version)
}

func (b *BackendPrefixed) DeleteVersion(uint64) error {
	//TODO implement me
	panic("implement me")
}

func (b *BackendPrefixed) LastVersion() uint64 {
	//TODO implement me
	panic("implement me")
}

func (b *BackendPrefixed) Commit() error {
	return b.backend.Commit()
}
