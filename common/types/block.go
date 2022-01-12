package types

import (
	"fmt"

	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
)

type BlockHash string
type BlockNum uint64

const BLOCK_VER = uint32(1)

func (m *Block) HashBytes(hasher tpcmm.Hasher, marshaler codec.Marshaler) ([]byte, error) {
	blBytes, err := marshaler.Marshal(m)
	if err != nil {
		return nil, err
	}

	return hasher.Compute(string(blBytes)), nil
}

func (m *Block) HashHex(hasher tpcmm.Hasher, marshaler codec.Marshaler) (string, error) {
	hashBytes, err := m.HashBytes(hasher, marshaler)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hashBytes), nil
}
