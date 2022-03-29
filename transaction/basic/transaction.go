package basic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/lazyledger/smt"

	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tpcrt "github.com/TopiaNetwork/topia/crypt"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

type TxID string

type TransactionCategory string

const (
	TransactionCategory_Topia_Universal TransactionCategory = "topia_universal"
	TransactionCategory_Eth                                 = "eth"
)

type TransactionVersion uint32

const (
	Transaction_V1                 TransactionVersion = 1
	Transaction_Topia_Universal_V1                    = 1
	Transaction_Eth_V1                                = 1
)

func TxRoot(txs []Transaction) []byte {
	tree := smt.NewSparseMerkleTree(smt.NewSimpleMap(), smt.NewSimpleMap(), sha256.New())
	for _, tx := range txs {
		txBytes, _ := tx.HashBytes()
		tree.Update(txBytes, txBytes)
	}

	return tree.Root()
}

func NewTransaction(log tplog.Logger, privKey tpcrtypes.PrivateKey, head *TransactionHead, data []byte) *Transaction {
	if head == nil {
		panic("Tx head nil")
	}

	if len(data) == 0 {
		panic("Tx data size 0")
	}

	if string(head.Category) == TransactionCategory_Eth && !tpcrtypes.NewFromBytes(head.FromAddr).IsEth() {
		panic("Tx from address is not eth")
	}

	cryptType, err := tpcrtypes.NewFromBytes(head.FromAddr).CryptType()
	if err != nil {
		panic("Can't get crypt type:" + err.Error())
	}
	cryptService := tpcrt.CreateCryptService(log, cryptType)
	signData, err := cryptService.Sign(privKey, data)
	if err != nil {
		panic("Sign err:" + err.Error())
	}

	pubKey, err := cryptService.ConvertToPublic(privKey)
	if err != nil {
		panic("To public key err:" + err.Error())
	}
	signInfo := &tpcrtypes.SignatureInfo{
		SignData:  signData,
		PublicKey: pubKey,
	}

	signBytes, _ := json.Marshal(signInfo)

	return &Transaction{
		Head: &TransactionHead{
			Category:  head.Category,
			Version:   head.Version,
			FromAddr:  head.FromAddr,
			Signature: signBytes,
		},
		Data: &TransactionData{
			Specification: data,
		},
	}
}

func (m *Transaction) CryptType() (tpcrtypes.CryptType, error) {
	return tpcrtypes.NewFromBytes(m.Head.FromAddr).CryptType()
}

func (m *Transaction) HashBytes() ([]byte, error) {
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	txBytes, err := marshaler.Marshal(m)
	if err != nil {
		return nil, err
	}

	hasher := tpcmm.NewBlake2bHasher(0)

	return hasher.Compute(string(txBytes)), nil
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
		TransactionFromAddressVerifier(),
		TransactionSignatureVerifier(),
	)
}
