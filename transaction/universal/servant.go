package universal

import (
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type TransactionUniversalServant interface {
	txbasic.TransactionServant
	GetGasEstimator() (GasEstimator, error)
}

type transactionUniversalServant struct {
	txbasic.TransactionServant
}

func NewTransactionUniversalServant(txServant txbasic.TransactionServant) TransactionUniversalServant {
	return &transactionUniversalServant{
		txServant,
	}
}

func (ts *transactionUniversalServant) GetGasEstimator() (GasEstimator, error) {
	return NewGasEstimator(ts.TransactionServant), nil
}
