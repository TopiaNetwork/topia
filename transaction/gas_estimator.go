package transaction

import "math/big"

type GasEstimator interface {
	Estimate(tx *Transaction) (*big.Int, error)
}

func NewGasEstimator() GasEstimator {
	return &gasEstimator{}
}

type gasEstimator struct {
}

func (ge *gasEstimator) Estimate(tx *Transaction) (*big.Int, error) {
	return big.NewInt(0), nil
}
