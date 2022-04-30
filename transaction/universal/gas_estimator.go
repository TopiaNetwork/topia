package universal

import (
	"math/big"
)

type GasEstimator interface {
	Estimate(txUni *TransactionUniversalWithHead) (*big.Int, error)
}

func NewGasEstimator() GasEstimator {
	return &gasEstimator{}
}

type gasEstimator struct {
}

func (ge *gasEstimator) Estimate(txUni *TransactionUniversalWithHead) (*big.Int, error) {
	switch TransactionUniversalType(txUni.Head.Type) {
	case TransactionUniversalType_Transfer:

	}
	return big.NewInt(0), nil
}
