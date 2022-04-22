package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/TopiaNetwork/topia/api/servant"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	"github.com/TopiaNetwork/topia/transaction/universal"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
)

type GetTransactionByHashRequestType struct {
	TxHash string
}

type GetTransactionByhashResponseType struct {
	BlockHash        []byte   `json:"blockHash"`
	BlockNumber      uint64   `json:"blockNumber"`
	From             []byte   `json:"from"`
	Gas              uint64   `json:"gas"`
	GasPrice         uint64   `json:"gasPrice"`
	GasFeeCap        big.Int  `json:"maxFeePerGas,omitempty"`
	GasTipCap        big.Int  `json:"maxPriorityFeePerGas,omitempty"`
	Hash             []byte   `json:"hash"`
	Input            []byte   `json:"input"`
	Nonce            uint64   `json:"nonce"`
	To               []byte   `json:"to"`
	TransactionIndex uint64   `json:"transactionIndex"`
	Value            *big.Int `json:"value"`
	Type             uint64   `json:"type"`
	Accesses         []byte   `json:"accessList,omitempty"`
	ChainID          *big.Int `json:"chainId,omitempty"`
	V                *big.Int `json:"v"`
	R                *big.Int `json:"r"`
	S                *big.Int `json:"s"`
}

func ConstructGetTransactionByhashResponseType(transaction txbasic.Transaction) *GetTransactionByhashResponseType {
	from := transaction.Head.FromAddr
	hash, _ := transaction.HashBytes()
	//构建v，r，s
	v, r, s := RawSignatureValues(transaction)
	chainId := string(transaction.GetHead().GetChainID())
	chainIdBigInt, _ := new(big.Int).SetString(chainId, 10)
	blockHash, blockNumber := blockInfo(transaction)
	result := &GetTransactionByhashResponseType{
		ChainID:          chainIdBigInt,
		Type:             uint64(0), //eth_legacyTxType
		From:             from,
		Gas:              uint64(0), //TODO:gasUsed不知道
		GasPrice:         GasPrice(&transaction),
		Hash:             hash,
		Input:            []byte{},
		Nonce:            Nonce(&transaction),
		To:               ToAddress(&transaction).Bytes(),
		Value:            Value(&transaction),
		TransactionIndex: transaction.GetHead().GetIndex(),
		BlockHash:        blockHash,
		BlockNumber:      blockNumber,
		V:                v,
		R:                r,
		S:                s,
	}
	return result
}

func ToAddress(tx *txbasic.Transaction) tpcrtypes.Address {
	var toAddress tpcrtypes.Address
	var targetData universal.TransactionUniversalTransfer
	if txbasic.TransactionCategory(tx.Head.Category) == txbasic.TransactionCategory_Topia_Universal {
		var txData universal.TransactionUniversal
		_ = json.Unmarshal(tx.Data.Specification, &txData)
		if txData.Head.Type == uint32(universal.TransactionUniversalType_Transfer) {
			_ = json.Unmarshal(txData.Data.Specification, &targetData)
			toAddress = targetData.TargetAddr
		}
	} else {
		return ""
	}
	return toAddress
}

func GasPrice(tx *txbasic.Transaction) uint64 {
	var gasPrice uint64
	switch txbasic.TransactionCategory(tx.Head.Category) {
	case txbasic.TransactionCategory_Topia_Universal:
		var txUniver universal.TransactionUniversal
		_ = json.Unmarshal(tx.Data.Specification, &txUniver)
		gasPrice = txUniver.Head.GasPrice
		return gasPrice
	default:
		return 0
	}
}

func GasLimit(tx *txbasic.Transaction) uint64 {
	var gasLimit uint64
	if txbasic.TransactionCategory(tx.Head.Category) == txbasic.TransactionCategory_Topia_Universal {
		var txData universal.TransactionUniversal
		_ = json.Unmarshal(tx.Data.Specification, &txData)
		gasLimit = txData.Head.GasLimit
		return gasLimit
	} else {
		return 0
	}
}

func Nonce(tx *txbasic.Transaction) uint64 {
	var nonce uint64
	if txbasic.TransactionCategory(tx.Head.Category) == txbasic.TransactionCategory_Topia_Universal {
		nonce = tx.GetHead().GetNonce()
		return nonce
	} else {
		return 0
	}
}

func Value(tx *txbasic.Transaction) *big.Int {
	var value *big.Int
	if txbasic.TransactionCategory(tx.Head.Category) == txbasic.TransactionCategory_Topia_Universal {
		var txData universal.TransactionUniversal
		_ = json.Unmarshal(tx.Data.Specification, &txData)
		var txDataSpe universal.TransactionUniversalTransfer
		_ = json.Unmarshal(txData.GetData().Specification, &txDataSpe)
		value = txDataSpe.Targets[0].Value
		return value
	} else {
		return big.NewInt(0)
	}
}

func RawSignatureValues(tx txbasic.Transaction) (v, r, s *big.Int) {
	//判断交易的发起人和fee的支付人是否是同一个，如果不是则出错
	var txData universal.TransactionUniversal
	_ = json.Unmarshal(tx.Data.Specification, &txData)

	from := tx.GetHead().GetFromAddr()
	feePayer := txData.GetHead().GetFeePayer()

	if !bytes.Equal(from, feePayer) {
		//报错
	}
	//获取发起人的签名
	sig := tx.GetHead().GetSignature()
	r, s, v = decodeSignature(sig)
	//根据chainId重新计算v
	chainId := string(tx.GetHead().GetChainID())
	chainIdBigInt, _ := new(big.Int).SetString(chainId, 10)
	if chainIdBigInt.Sign() != 0 {
		v = big.NewInt(int64(sig[64] + 35))
		v.Add(v, big.NewInt(2))
	}
	return r, s, v
}

func decodeSignature(sig []byte) (r, s, v *big.Int) {
	//1.检查签名数据的长度
	if len(sig) != SignatureLength {
		panic(fmt.Sprintf("wrong size for signature: got %d, want %d", len(sig), crypto.SignatureLength))
	}
	//2.截取签名数据初始化v，r，s
	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	v = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return r, s, v
}

func blockInfo(tx txbasic.Transaction) (blockHash []byte, blockNumber uint64) {
	//根据txHash获取block
	apiServant := servant.NewAPIServant()
	txHash, _ := tx.HashHex()
	block, _ := apiServant.GetBlockByTxHash(txHash)
	//获取blockHash
	blockHash, _ = block.HashBytes()
	//获取blockNumber
	blockNumber = block.GetHead().GetHeight()
	return blockHash, blockNumber
}
