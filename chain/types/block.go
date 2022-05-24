package types

import (
	"encoding/hex"
	"fmt"

	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
)

type BlockHash string
type BlockNum uint64

const BLOCK_VER = uint32(1)

func (m *Block) HashBytes() ([]byte, error) {
	hasher := tpcmm.NewBlake2bHasher(0)
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)

	blBytes, err := marshaler.Marshal(m)
	if err != nil {
		return nil, err
	}

	return hasher.Compute(string(blBytes)), nil
}

func (m *Block) BlockNum() BlockNum {
	return BlockNum(m.Head.Height)
}

func (m *Block) BlockHash() (BlockHash, error) {
	hashBytes, err := m.HashBytes()
	if err != nil {
		return "", err
	}

	return BlockHash(hex.EncodeToString(hashBytes)), nil
}

func (m *Block) HashHex() (string, error) {
	hashBytes, err := m.HashBytes()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hex.EncodeToString(hashBytes)), nil
}

func (m *BlockHead) DeepCopy(src *BlockHead) (*BlockHead, error) {
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	blBytes, err := marshaler.Marshal(src)
	if err != nil {
		return nil, err
	}

	var dst BlockHead
	err = marshaler.Unmarshal(blBytes, &dst)
	if err != nil {
		return nil, err
	}

	return &dst, err
}

func (m *BlockResult) HashBytes(hasher tpcmm.Hasher, marshaler codec.Marshaler) ([]byte, error) {
	blBytes, err := marshaler.Marshal(m)
	if err != nil {
		return nil, err
	}

	return hasher.Compute(string(blBytes)), nil
}
