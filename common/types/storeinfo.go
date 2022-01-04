package types

import "github.com/TopiaNetwork/topia/transaction"

type BlockResultStoreInfo struct {
	BlockResult
	TxResults map[transaction.TxID]*transaction.TransactionResult
}
