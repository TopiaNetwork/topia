package execution

import tx "github.com/TopiaNetwork/topia/transaction"

type PackedTxs struct {
	StateVersion uint64
	TxRoot       []byte
	TxList       []tx.Transaction
}

type PackedTxsResult struct {
	StateVersion uint64
	TxRSRoot     []byte
	TxsResult    []tx.TransactionResult
}
