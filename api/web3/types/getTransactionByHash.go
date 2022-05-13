package types

import (
	types "github.com/TopiaNetwork/topia/api/web3/types/hexutil"
)

type GetTransactionByHashRequestType struct {
	TxHash string
}

const SignatureLength = 65

type EthRpcTransaction struct {
	BlockHash        Hash          `json:"blockHash"`
	BlockNumber      *types.Big    `json:"blockNumber"`
	From             Address       `json:"from"`
	Gas              types.Uint64  `json:"gas"`
	GasPrice         *types.Big    `json:"gasPrice"`
	GasFeeCap        *types.Big    `json:"maxFeePerGas,omitempty"`
	GasTipCap        *types.Big    `json:"maxPriorityFeePerGas,omitempty"`
	Hash             Hash          `json:"hash"`
	Input            types.Bytes   `json:"input"`
	Nonce            types.Uint64  `json:"nonce"`
	To               Address       `json:"to"`
	TransactionIndex *types.Uint64 `json:"transactionIndex"`
	Value            *types.Big    `json:"value"`
	Type             types.Uint64  `json:"type"`
	Accesses         string        `json:"accessList,omitempty"`
	ChainID          *types.Big    `json:"chainId,omitempty"`
	V                *types.Big    `json:"v"`
	R                *types.Big    `json:"r"`
	S                *types.Big    `json:"s"`
}
