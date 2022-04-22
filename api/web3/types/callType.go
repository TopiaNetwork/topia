package types

import (
	"encoding/json"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	"github.com/TopiaNetwork/topia/transaction/universal"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type CallRequestType struct {
	TranArgs    transactionArgs
	BlockNumber string
}

type CallResponseType struct {
	Balance string `json:"balance"`
}

type transactionArgs struct {
	From                 string `json:"from"`
	To                   string `json:"to"`
	Gas                  uint64 `json:"gas"`
	GasPrice             uint64 `json:"gasPrice"`
	MaxFeePerGas         string `json:"maxFeePerGas"`
	MaxPriorityFeePerGas string `json:"maxPriorityFeePerGas"`
	Value                int64  `json:"value"`
	Nonce                uint64 `json:"nonce"`

	Data  string `json:"data"`
	Input string `json:"input"`

	ChainID *hexutil.Big `json:"chainId,omitempty"`
}

func ConstructTransaction(call CallRequestType) *txbasic.Transaction {
	transactionUniversalHead := universal.TransactionUniversalHead{
		Version:  txbasic.Transaction_Topia_Universal_V1,
		FeePayer: []byte(call.TranArgs.From),
		GasPrice: call.TranArgs.GasPrice,
		GasLimit: call.TranArgs.Gas,
		Type:     uint32(universal.TransactionUniversalType_ContractInvoke),
	}
	var txData []byte
	if call.TranArgs.Data != "" {
		txData, _ = json.Marshal(call.TranArgs.Data)
	} else {
		txData, _ = json.Marshal(call.TranArgs.Input)
	}
	txUni := universal.TransactionUniversal{
		Head: &transactionUniversalHead,
		Data: &universal.TransactionUniversalData{
			Specification: txData,
		},
	}

	transactionHead := txbasic.TransactionHead{
		Category: []byte(txbasic.TransactionCategory_Eth),
		ChainID:  null,
		Version:  uint32(txbasic.Transaction_Eth_V1),
		FromAddr: []byte(call.TranArgs.From),
		Nonce:    call.TranArgs.Nonce,
	}
	txDataBytes, _ := json.Marshal(txUni)
	return &txbasic.Transaction{
		Head: &transactionHead,
		Data: &txbasic.TransactionData{
			Specification: txDataBytes,
		},
	}
}
