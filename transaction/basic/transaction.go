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

func TxRoot(txs []*Transaction) []byte {
	tree := smt.NewSparseMerkleTree(smt.NewSimpleMap(), smt.NewSimpleMap(), sha256.New())
	for _, tx := range txs {
		txBytes, _ := tx.HashBytes()
		tree.Update(txBytes, txBytes)
	}

	return tree.Root()
}

func TxRootByBytes(txsBytes [][]byte) []byte {
	tree := smt.NewSparseMerkleTree(smt.NewSimpleMap(), smt.NewSimpleMap(), sha256.New())
	for _, txBytes := range txsBytes {
		txHashBytes := tpcmm.NewBlake2bHasher(0).Compute(string(txBytes))
		tree.Update(txHashBytes, txHashBytes)
	}

	return tree.Root()
}

func TxRootWithRtn(txs []Transaction) ([]byte, [][]byte) {
	var txsBytes [][]byte
	tree := smt.NewSparseMerkleTree(smt.NewSimpleMap(), smt.NewSimpleMap(), sha256.New())
	for _, tx := range txs {
		txBytes, _ := tx.HashBytes()
		tree.Update(txBytes, txBytes)
		txsBytes = append(txsBytes, txBytes)
	}

	return tree.Root(), txsBytes
}

func NewTransaction(log tplog.Logger, cryptService tpcrt.CryptService, privKey tpcrtypes.PrivateKey, nonce uint64, txCategory TransactionCategory, txVersion TransactionVersion, data []byte) *Transaction {
	if privKey == nil {
		panic("Tx private key nil")
	}

	if len(data) == 0 {
		panic("Tx data size 0")
	}

	txFromPubKey, err := cryptService.ConvertToPublic(privKey)
	if err != nil {
		panic("Can't convert public key from fromPriKey: " + err.Error())
	}
	txFromAddr, err := cryptService.CreateAddress(txFromPubKey)
	if err != nil {
		panic("Can't convert public key from fromPubKey: " + err.Error())
	}

	if txCategory == TransactionCategory_Eth && !txFromAddr.IsEth() {
		panic("Tx from address is not eth")
	}

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
			Category:  []byte(txCategory),
			Version:   uint32(txVersion),
			FromAddr:  txFromAddr.Bytes(),
			Nonce:     nonce,
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

func (m *Transaction) TxID() (TxID, error) {
	hashBytes, err := m.HashBytes()
	if err != nil {
		return "", err
	}

	return TxID(hex.EncodeToString(hashBytes)), nil
}

func (m *Transaction) HashHex() (string, error) {
	hashBytes, err := m.HashBytes()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hex.EncodeToString(hashBytes)), nil
}

func (m *Transaction) BasicVerify(ctx context.Context, log tplog.Logger, txServant TransactionServant) VerifyResult {
	return ApplyTransactionVerifiers(ctx, log, m, txServant,
		TransactionChainIDVerifier(),
		TransactionFromAddressVerifier(),
		TransactionSignatureVerifier(),
	)
}

// TxDifference returns a new set which is the difference between a and b.
func TxDifference(a, b []*Transaction) []*Transaction {
	keep := make([]*Transaction, 0, len(a))
	remove := make(map[string]struct{})
	for _, tx := range b {
		if txId, err := tx.HashHex(); err != nil {
			remove[txId] = struct{}{}
		}
	}
	for _, tx := range a {
		if txId, err := tx.HashHex(); err != nil {
			if _, ok := remove[txId]; !ok {
				keep = append(keep, tx)
			}
		}
	}
	return keep
}
