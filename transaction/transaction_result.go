package transaction

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/lazyledger/smt"

	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
)

func TxResultRoot(txResults []TransactionResult, txs []Transaction) []byte {
	tree := smt.NewSparseMerkleTree(smt.NewSimpleMap(), smt.NewSimpleMap(), sha256.New())
	for i, txR := range txResults {
		txBytes, _ := txR.HashBytes(txs[i].FromAddr)
		tree.Update(txBytes, txBytes)
	}

	return tree.Root()
}

func (m *TransactionResult) HashBytes(fromAddr []byte) ([]byte, error) {
	codecType := codec.CodecType_PROTO
	isEth := tpcrtypes.NewFromBytes(fromAddr).IsEth()
	if isEth {
		codecType = codec.CodecType_RLP
	}

	marshaler := codec.CreateMarshaler(codecType)

	blBytes, err := marshaler.Marshal(m)
	if err != nil {
		return nil, err
	}

	hasher := tpcmm.NewBlake2bHasher(0)

	return hasher.Compute(string(blBytes)), nil
}

func (m *TransactionResult) HashHex(fromAddr []byte) (string, error) {
	hashBytes, err := m.HashBytes(fromAddr)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hex.EncodeToString(hashBytes)), nil
}
