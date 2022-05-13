package convert

import (
	"encoding/json"
	"fmt"
	"github.com/TopiaNetwork/topia/api/servant"
	tpTypes "github.com/TopiaNetwork/topia/api/web3/types"
	types "github.com/TopiaNetwork/topia/api/web3/types/hexutil"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	"github.com/TopiaNetwork/topia/transaction/universal"
	"math/big"
)

func TopiaTransactionToEthTransaction(tx txbasic.Transaction, apiServant servant.APIServant) (*tpTypes.EthRpcTransaction, error) {
	v, r, s := RawSignatureValues(tx)
	chainId := tx.GetHead().GetChainID()
	chainIdBigInt := new(big.Int).SetBytes(chainId)
	bh, blockNumber := blockInfo(tx, apiServant)
	gasPrice := new(big.Int).SetUint64(GasPrice(&tx))
	index := tx.GetHead().GetIndex()
	height := new(big.Int).SetUint64(blockNumber)

	var from tpTypes.Address
	from.SetBytes(tx.Head.FromAddr)
	var to tpTypes.Address
	to.SetBytes(ToAddress(&tx).Bytes())
	var hash tpTypes.Hash
	hashByte, _ := tx.HashBytes()
	hash.SetBytes(hashByte)
	var blockHash tpTypes.Hash
	blockHash.SetBytes(bh)
	result := &tpTypes.EthRpcTransaction{
		ChainID:          (*types.Big)(chainIdBigInt),
		Type:             types.Uint64(0),
		From:             from,
		Gas:              types.Uint64(GasLimit(&tx)),
		GasPrice:         (*types.Big)(gasPrice),
		Hash:             hash,
		Input:            types.Bytes{},
		Nonce:            types.Uint64(Nonce(&tx)),
		To:               to,
		Value:            (*types.Big)(Value(&tx)),
		TransactionIndex: (*types.Uint64)(&index),
		BlockHash:        blockHash,
		BlockNumber:      (*types.Big)(height),
		V:                (*types.Big)(v),
		R:                (*types.Big)(r),
		S:                (*types.Big)(s),
	}
	return result, nil
}

func AddPrefix(s string) string {
	return "0x" + s
}

func ToAddress(tx *txbasic.Transaction) tpcrtypes.Address {
	var toAddress tpcrtypes.Address
	var targetData universal.TransactionUniversalTransfer
	if txbasic.TransactionCategory(tx.Head.Category) == txbasic.TransactionCategory_Eth {
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
	case txbasic.TransactionCategory_Eth:
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
	if txbasic.TransactionCategory(tx.Head.Category) == txbasic.TransactionCategory_Eth {
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
	if txbasic.TransactionCategory(tx.Head.Category) == txbasic.TransactionCategory_Eth {
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
	var txData universal.TransactionUniversal
	_ = json.Unmarshal(tx.Data.Specification, &txData)

	sig := tx.GetHead().GetSignature()
	r, s, v = decodeSignature(sig)

	chainId := tx.GetHead().GetChainID()
	chainIdBigInt := new(big.Int).SetBytes(chainId)
	if chainIdBigInt.Sign() != 0 {
		v = big.NewInt(int64(sig[64] + 35))
		v.Add(v, big.NewInt(2))
	}
	return v, r, s
}

func decodeSignature(sig []byte) (r, s, v *big.Int) {

	if len(sig) != SignatureLength {
		panic(fmt.Sprintf("wrong size for signature: got %d, want %d", len(sig), SignatureLength))
	}

	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	v = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return r, s, v
}

func blockInfo(tx txbasic.Transaction, apiServant servant.APIServant) (blockHash []byte, blockNumber uint64) {

	servant := apiServant
	txHash, _ := tx.HashHex()
	block, _ := servant.GetBlockByTxHash(txHash)

	blockHash, _ = block.HashBytes()

	blockNumber = block.GetHead().GetHeight()
	return blockHash, blockNumber
}
