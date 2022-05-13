package types

import (
	"encoding/json"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/currency"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	"github.com/TopiaNetwork/topia/transaction/universal"
	"math/big"
	"strconv"
)

type EstimateGasRequestType struct {
	TranArgs TransactionArgs
	Height   string
}

type EstimateGasResponseType struct {
	Balance string `json:"balance"`
}

func ConstructGasTransaction(call EstimateGasRequestType) *txbasic.Transaction {
	var from []byte
	if call.TranArgs.From != nil {
		from = call.TranArgs.From.Bytes()
	} else {
		from = []byte{}
	}
	transactionHead := txbasic.TransactionHead{
		Category: []byte(txbasic.TransactionCategory_Eth),
		ChainID:  null,
		Version:  uint32(txbasic.Transaction_Eth_V1),
		FromAddr: from,
	}
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

	transactionUniversalHead := universal.TransactionUniversalHead{
		Version:  txbasic.Transaction_Topia_Universal_V1,
		FeePayer: from,
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Type:     uint32(universal.TransactionUniversalType_Transfer),
	}
	var value *big.Int
	if call.TranArgs.Value != nil {
		value = call.TranArgs.Value.ToInt()
	} else {
		value = big.NewInt(0)
	}
	targets := []universal.TargetItem{
		{
			currency.TokenSymbol_Native,
			value,
		},
	}
	var to string
	if call.TranArgs.To != nil {
		to = call.TranArgs.To.String()
	} else {
		to = ""
	}
	txfer := universal.TransactionUniversalTransfer{
		TransactionHead:          transactionHead,
		TransactionUniversalHead: transactionUniversalHead,
		TargetAddr:               tpcrtypes.Address(to),
		Targets:                  targets,
	}

	txferData, _ := json.Marshal(txfer)

	txUni := universal.TransactionUniversal{
		Head: &txfer.TransactionUniversalHead,
		Data: &universal.TransactionUniversalData{
			Specification: txferData,
		},
	}
	txDataBytes, _ := json.Marshal(txUni)
	return &txbasic.Transaction{
		Data: &txbasic.TransactionData{
			Specification: txDataBytes,
		},
	}
}
