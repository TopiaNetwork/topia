package transaction

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
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
	TransactionType_Relay
	TransactionType_DataTransfer
)

func TxRoot(txs []Transaction) []byte {
	tree := smt.NewSparseMerkleTree(smt.NewSimpleMap(), smt.NewSimpleMap(), sha256.New())
	for _, tx := range txs {
		txBytes, _ := tx.HashBytes()
		tree.Update(txBytes, txBytes)
	}

	return tree.Root()
}

func (m *Transaction) HashBytes() ([]byte, error) {
	codecType := codec.CodecType_PROTO
	isEth := tpcrtypes.NewFromBytes(m.FromAddr).IsEth()
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

func (m *Transaction) HashHex() (string, error) {
	hashBytes, err := m.HashBytes()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hex.EncodeToString(hashBytes)), nil
}

func (m *Transaction) BasicVerify(ctx context.Context, log tplog.Logger, txServant TansactionServant) VerifyResult {
	return ApplyTransactionVerifiers(ctx, log, m, txServant,
		TransactionChainIDVerifier(),
		TransactionAddressVerifier(),
		TransactionGasVerifier(),
		TransactionNonceVerifier(),
		TransactionSignatureVerifier(),
	)
}

func (m *Transaction) TxAction() TransactionAction {
	txType := TransactionType(m.Type)
	switch txType {
	case TransactionType_Transfer:
		var target []TargetItem
		err := json.Unmarshal(m.Data, &target)
		if err != nil {
			return nil
		}
		return &TransactionTransfer{m, target}
	}

	return nil
}
