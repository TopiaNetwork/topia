package common

import (
	cryrand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"math/rand"
)

type SeedRandReader struct {
	seedNumber int64
}

func NewSeedRandReader(seed []byte) (*SeedRandReader, error) {
	if len(seed) == 0 {
		return nil, errors.New("seed buf size 0")
	}

	seedHash := sha256.Sum256(seed)
	seedNumber := binary.BigEndian.Uint64(seedHash[:])

	return &SeedRandReader{
		seedNumber: int64(seedNumber),
	}, nil
}

func (srr *SeedRandReader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, errors.New("buf size 0")
	}

	randomizer := rand.New(rand.NewSource(srr.seedNumber))

	return randomizer.Read(p)
}

func NewRandReader(seed string) (io.Reader, error) {
	if len(seed) == 0 {
		return cryrand.Reader, nil
	}

	return NewSeedRandReader([]byte(seed))
}
