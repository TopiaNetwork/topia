package transaction

import (
	"math/big"

	tpcmm "github.com/TopiaNetwork/topia/common"
)

type TargetItem struct {
	Symbol tpcmm.TokenSymbol
	Value  *big.Int
}

type TransactionTransfer struct {
	Transaction
	Target []TargetItem
}
