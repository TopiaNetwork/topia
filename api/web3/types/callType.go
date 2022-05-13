package types

import (
	"encoding/json"
	types "github.com/TopiaNetwork/topia/api/web3/types/hexutil"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	"github.com/TopiaNetwork/topia/transaction/universal"
	"strconv"
)

type CallRequestType struct {
	TranArgs    TransactionArgs
	BlockNumber interface{}
}

type CallResponseType struct {
	Balance string `json:"balance"`
}

type TransactionArgs struct {
	From                 *Address      `json:"from"`
	To                   *Address      `json:"to"`
	Gas                  *types.Uint64 `json:"gas"`
	GasPrice             *types.Big    `json:"gasPrice"`
	MaxFeePerGas         *types.Big    `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *types.Big    `json:"maxPriorityFeePerGas"`
	Value                *types.Big    `json:"value"`
	Nonce                *types.Uint64 `json:"nonce"`

	// We accept "data" and "input" for backwards-compatibility reasons.
	// "input" is the newer name and should be preferred by clients.
	// Issue detail: https://github.com/ethereum/go-ethereum/issues/15628
	Data  *types.Bytes `json:"data"`
	Input *types.Bytes `json:"input"`

	// Introduced by AccessListTxType transaction.
	AccessList *AccessList `json:"accessList,omitempty"`
	ChainID    *types.Big  `json:"chainId,omitempty"`
}

func ConstructTransaction(call CallRequestType) *txbasic.Transaction {
	var gasPrice, gasLimit uint64
	if call.TranArgs.Gas == nil {
		gasLimit = 0
	} else {
		gasLimit, _ = strconv.ParseUint(call.TranArgs.Gas.String(), 10, 64)
	}
	if call.TranArgs.GasPrice == nil {
		gasPrice = 0
	} else {
		gasPrice, _ = strconv.ParseUint(call.TranArgs.GasPrice.String(), 10, 64)
	}

	var from []byte
	if call.TranArgs.From != nil {
		from = call.TranArgs.From.Bytes()
	} else {
		from = []byte{}
	}
	transactionUniversalHead := universal.TransactionUniversalHead{
		Version:  txbasic.Transaction_Topia_Universal_V1,
		FeePayer: from,
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Type:     uint32(universal.TransactionUniversalType_ContractInvoke),
	}
	var txData []byte
	if call.TranArgs.Data != nil {
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
		FromAddr: from,
	}
	txDataBytes, _ := json.Marshal(txUni)
	return &txbasic.Transaction{
		Head: &transactionHead,
		Data: &txbasic.TransactionData{
			Specification: txDataBytes,
		},
	}
}
