package execution

import (
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type PackedTxs struct {
	StateVersion uint64
	TxRoot       []byte
	TxList       []*txbasic.Transaction
}

type PackedTxsResult struct {
	StateVersion uint64
	TxRSRoot     []byte
	TxsResult    []txbasic.TransactionResult
}
