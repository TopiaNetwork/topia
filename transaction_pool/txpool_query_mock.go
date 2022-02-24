package transactionpool

import (
	"github.com/TopiaNetwork/topia/transaction"
	"math/big"
)

type TxPoolQuery struct {
	cost        *big.Int
	gas         uint64
}

func (TxPoolQuery) EstimateTxGas(tx *transaction.Transaction) uint64{
	return tx.GasPrice * tx.GasLimit
	//no achieved
}
func (TxPoolQuery) EstimateTxCost(tx *transaction.Transaction) *big.Int{
	return new(big.Int).SetBytes(tx.Value)
	//no achieved
}
