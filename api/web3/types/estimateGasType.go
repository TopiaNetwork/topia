package types

import (
	"encoding/json"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/currency"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	"github.com/TopiaNetwork/topia/transaction/universal"
	"math/big"
)

type EstimateGasRequestType struct {
	TranArgs transactionArgs
}

type EstimateGasResponseType struct {
	Balance string `json:"balance"`
}

func ConstructGasTransaction(call EstimateGasRequestType) *txbasic.Transaction {
	transactionHead := txbasic.TransactionHead{
		Category: []byte(txbasic.TransactionCategory_Eth),
		ChainID:  null,
		Version:  uint32(txbasic.Transaction_Eth_V1),
		FromAddr: []byte(call.TranArgs.From),
		Nonce:    call.TranArgs.Nonce,
	}
	transactionUniversalHead := universal.TransactionUniversalHead{
		Version:  txbasic.Transaction_Topia_Universal_V1,
		FeePayer: []byte(call.TranArgs.From),
		GasPrice: call.TranArgs.GasPrice,
		GasLimit: call.TranArgs.Gas,
		Type:     uint32(universal.TransactionUniversalType_Transfer),
	}
	targets := []universal.TargetItem{
		{
			currency.TokenSymbol_Native,
			big.NewInt(call.TranArgs.Value),
		},
	}
	txfer := universal.TransactionUniversalTransfer{
		TransactionHead:          transactionHead,
		TransactionUniversalHead: transactionUniversalHead,
		TargetAddr:               tpcrtypes.Address(call.TranArgs.To),
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
