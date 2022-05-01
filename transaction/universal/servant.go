package universal

import txbasic "github.com/TopiaNetwork/topia/transaction/basic"

type TansactionUniversalServant interface {
	txbasic.TransactionServant
	GetGasEstimator() (GasEstimator, error)
}

type transactionUniversalServant struct {
	txbasic.TransactionServant
}

func NewTansactionUniversalServant(txServant txbasic.TransactionServant) TansactionUniversalServant {
	return &transactionUniversalServant{
		txServant,
	}
}

func (ts *transactionUniversalServant) GetGasEstimator() (GasEstimator, error) {
	return NewGasEstimator(), nil
}
