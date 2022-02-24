package transactionpool

import (
	"github.com/TopiaNetwork/topia/transaction"
	"math/big"
)

type txPoolQuery interface {
	EstimateTxCost(tx *transaction.Transaction) (*big.Int, error)
	EstimateTxGas(tx *transaction.Transaction) (uint64,error)

}
