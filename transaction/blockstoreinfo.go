package transaction

import (
	"github.com/TopiaNetwork/topia/chain/types"
)

type BlockResultStoreInfo struct {
	types.BlockResult
	TxResults map[string]*TransactionResult
}
