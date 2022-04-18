package basic

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/lazyledger/smt"

	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
)

func TxResultRoot(txResults []TransactionResult, txs []Transaction) []byte {
	tree := smt.NewSparseMerkleTree(smt.NewSimpleMap(), smt.NewSimpleMap(), sha256.New())
	for _, txR := range txResults {
		txBytes, _ := txR.HashBytes()
		tree.Update(txBytes, txBytes)
	}

	return tree.Root()
}

func (m *TransactionResult) HashBytes() ([]byte, error) {
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)

	blBytes, err := marshaler.Marshal(m)
	if err != nil {
		return nil, err
	}

	hasher := tpcmm.NewBlake2bHasher(0)

	return hasher.Compute(string(blBytes)), nil
}

func (m *TransactionResult) HashHex() (string, error) {
	hashBytes, err := m.HashBytes()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hex.EncodeToString(hashBytes)), nil
}
