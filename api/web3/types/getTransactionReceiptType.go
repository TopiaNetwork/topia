package types

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/servant"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	"github.com/TopiaNetwork/topia/transaction/universal"
)

type GetTransactionReceiptRequestType struct {
	TxHash string
}

type GetTransactionReceiptResponseType struct {
	BlockHash         interface{} `json:"blockHash"`
	BlockNumber       interface{} `json:"blockNumber"`
	TransactionHash   interface{} `json:"transactionHash"`
	TransactionIndex  interface{} `json:"transactionIndex"`
	From              interface{} `json:"from"`
	To                interface{} `json:"to"`
	GasUsed           interface{} `json:"gasUsed"`
	CumulativeGasUsed interface{} `json:"cumulativeGasUsed"`
	ContractAddress   interface{} `json:"contractAddress"`
	Logs              interface{} `json:"logs"`
	LogsBloom         interface{} `json:"logsBloom"`
	Type              interface{} `json:"type"`
	EffectiveGasPrice interface{} `json:"effectiveGasPrice"`
	Status            interface{} `json:"status"`
}

func ConstructGetTransactionReceiptResponseType(transactionResult txbasic.TransactionResult) *GetTransactionReceiptResponseType {
	var resultData universal.TransactionResultUniversal
	var txHash string
	if txbasic.TransactionCategory(transactionResult.GetHead().GetCategory()) == txbasic.TransactionCategory_Eth {
		_ = json.Unmarshal(transactionResult.GetData().Specification, &resultData)
		txHash = string(resultData.GetTxHash())
	}
	apiServant := servant.NewAPIServant()
	transaction, _ := apiServant.GetTransactionByHash(txHash)
	block, _ := apiServant.GetBlockByTxHash(txHash)
	var transactionData universal.TransactionUniversal
	_ = json.Unmarshal(transaction.GetData().GetSpecification(), transactionData)
	var transactionUniversalData universal.TransactionUniversalTransfer
	_ = json.Unmarshal(transactionData.GetData().GetSpecification(), transactionUniversalData)

	result := &GetTransactionReceiptResponseType{
		BlockHash:         block.GetHead().GetHash(),
		BlockNumber:       block.GetHead().GetHeight(),
		TransactionHash:   txHash,
		TransactionIndex:  transaction.GetHead().GetIndex(),
		From:              transaction.GetHead().GetFromAddr(),
		To:                transactionUniversalData.TargetAddr,
		GasUsed:           resultData.GetGasUsed(),
		CumulativeGasUsed: 0, //累加计算
		ContractAddress:   []byte{},
		Logs:              []byte{},
		LogsBloom:         []byte{},
		Type:              0,
		EffectiveGasPrice: 0, //需要计算
		Status:            resultData.GetStatus(),
	}
	return result
}
