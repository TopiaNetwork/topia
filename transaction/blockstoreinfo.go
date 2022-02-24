package transaction

import "github.com/TopiaNetwork/topia/common/types"

type BlockResultStoreInfo struct {
	types.BlockResult
	TxResults map[TxID]*TransactionResult
}

