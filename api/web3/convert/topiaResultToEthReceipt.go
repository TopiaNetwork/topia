package convert

import (
	"encoding/hex"
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/servant"
	tpTypes "github.com/TopiaNetwork/topia/api/web3/types"
	types "github.com/TopiaNetwork/topia/api/web3/types/hexutil"
	"github.com/TopiaNetwork/topia/codec"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	"github.com/TopiaNetwork/topia/transaction/universal"
	"math/big"
	"strconv"
)

func ConvertTopiaResultToEthReceipt(transactionResult txbasic.TransactionResult, apiServant servant.APIServant) map[string]interface{} {
	var resultData universal.TransactionResultUniversal
	var txHash []byte
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	if txbasic.TransactionCategory(transactionResult.GetHead().GetCategory()) == txbasic.TransactionCategory_Eth {
		_ = marshaler.Unmarshal(transactionResult.GetData().GetSpecification(), &resultData)
		txHash = resultData.GetTxHash()
	}
	servant := apiServant
	transaction, _ := servant.GetTransactionByHash(string(txHash))
	block, _ := servant.GetBlockByTxHash(string(txHash))
	var transactionUniversal universal.TransactionUniversal
	json.Unmarshal(transaction.GetData().GetSpecification(), &transactionUniversal)
	var transactionUniversalTransfer universal.TransactionUniversalTransfer
	json.Unmarshal(transactionUniversal.GetData().GetSpecification(), &transactionUniversalTransfer)

	status, _ := strconv.ParseInt(resultData.GetStatus().String(), 10, 64)
	var sta *big.Int
	if status == 1 {
		sta = big.NewInt(0)
	} else {
		sta = big.NewInt(1)
	}

	bloom, _ := hex.DecodeString(
		"0000000000000000000000000000000000000000000000000000000000000000" +
			"0000000000000000000000000000000000000000000000000000000000000000" +
			"0000000000000000000000000000000000000000000000000000000000000000" +
			"0000000000000000000000000000000000000000000000000000000000000000" +
			"0000000000000000000000000000000000000000000000000000000000000000" +
			"0000000000000000000000000000000000000000000000000000000000000000" +
			"0000000000000000000000000000000000000000000000000000000000000000" +
			"0000000000000000000000000000000000000000000000000000000000000000")

	var blockH tpTypes.Hash
	blockH.SetBytes(block.GetHead().GetHash())
	var tranH tpTypes.Hash
	tranH.SetBytes(txHash)
	var from tpTypes.Address
	from.SetBytes(transaction.GetHead().GetFromAddr())
	var conAddr tpTypes.Address
	conAddr.SetBytes(resultData.GetData())
	result := map[string]interface{}{
		"blockHash":         blockH,
		"blockNumber":       types.Uint64(block.GetHead().GetHeight()),
		"transactionHash":   tranH,
		"transactionIndex":  types.Uint64(uint64(1)),
		"from":              from,
		"to":                nil,
		"gasUsed":           types.Uint64(resultData.GetGasUsed()),
		"cumulativeGasUsed": types.Uint64(resultData.GetGasUsed()),
		"contractAddress":   conAddr,
		"logs":              []*tpTypes.Address{},
		"logsBloom":         tpTypes.BytesToBloom(bloom),
		"type":              types.Uint(2),
		"effectiveGasPrice": types.Uint64(0),
		"status":            types.Uint(sta.Uint64()),
	}
	return result
}
