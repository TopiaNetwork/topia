package transaction

import (
	"crypto/sha256"
	"fmt"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/lazyledger/smt"
)

type TxID string

type TransactionType uint32

const (
	TransactionType_Unknown TransactionType = iota
	TransactionType_Transfer
	TransactionType_ContractDeploy
	TransactionType_ContractInvoke
	TransactionType_NativeInvoke
	TransactionType_Pay
	TransactionType_Relay
	TransactionType_DataTransfer
)

func TxRoot(hasher tpcmm.Hasher, marshaler codec.Marshaler, txs []Transaction) []byte {
	tree := smt.NewSparseMerkleTree(smt.NewSimpleMap(), smt.NewSimpleMap(), sha256.New())
	for _, tx := range txs {
		txBytes, _ := tx.HashBytes(hasher, marshaler)
		tree.Update(txBytes, txBytes)
	}

	return tree.Root()
}

func (m *Transaction) HashBytes(hasher tpcmm.Hasher, marshaler codec.Marshaler) ([]byte, error) {
	blBytes, err := marshaler.Marshal(m)
	if err != nil {
		return nil, err
	}

	return hasher.Compute(string(blBytes)), nil
}

func (m *Transaction) HashHex(hasher tpcmm.Hasher, marshaler codec.Marshaler) (string, error) {
	hashBytes, err := m.HashBytes(hasher, marshaler)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hashBytes), nil
}
