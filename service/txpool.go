package service

import (
	"github.com/TopiaNetwork/topia/transaction/basic"
	txpool "github.com/TopiaNetwork/topia/transaction_pool"
)

type TxPoolService interface {
	Pending() ([]*basic.Transaction, error)

	Size() int

	//UpdateTx(tx *basic.Transaction, txID basic.TxID) error
}

type txPoolService struct {
	txpool.TransactionPool
}

func NewTxPoolService(txPool txpool.TransactionPool) TxPoolService {
	return &txPoolService{
		TransactionPool: txPool,
	}
}
