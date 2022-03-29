package universal

import txbasic "github.com/TopiaNetwork/topia/transaction/basic"

type TansactionUniversalServant interface {
	txbasic.TansactionServant
	GetGasEstimator() (GasEstimator, error)
}

type transactionUniversalServant struct {
	txbasic.TansactionServant
}

func NewTansactionUniversalServant(txServant txbasic.TansactionServant) TansactionUniversalServant {
	return &transactionUniversalServant{
		txServant,
	}
}

func (ts *transactionUniversalServant) GetGasEstimator() (GasEstimator, error) {
	return NewGasEstimator(), nil
}
