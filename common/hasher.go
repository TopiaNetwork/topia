package common

import (
	"hash"

	"golang.org/x/crypto/blake2b"
)

type Hasher interface {
	Compute(string) []byte
	Size() int
}

type blake2bHasher struct {
	size int
	hash hash.Hash
}

func NewBlake2bHasher(size int) Hasher {
	sizeT := size
	if size < 0 {
		panicf("invalid blake2b hasher size: %d", size)
	}

	if size == 0 {
		sizeT = blake2b.Size256
	}

	h, err := blake2b.New(sizeT, nil)
	if err != nil {
		panicf("blake2b new err:%v", err)
	}
	return &blake2bHasher{
		size,
		h,
	}
}

func (b2bHasher *blake2bHasher) Compute(s string) []byte {
	_, _ = b2bHasher.hash.Write([]byte(s))
	return b2bHasher.hash.Sum(nil)
}

func (b2bHasher *blake2bHasher) Size() int {
	return b2bHasher.hash.Size()
}
