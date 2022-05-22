package service

import (
	"github.com/TopiaNetwork/topia/transaction/basic"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

type TxPoolService interface {
	Pending() ([]*basic.Transaction, error)

	Size() int64

	//UpdateTx(tx *basic.Transaction, txID basic.TxID, isLocal bool) error
}

type txPoolService struct {
	txpooli.TransactionPool
}

func NewTxPoolService(txPool txpooli.TransactionPool) TxPoolService {
	return &txPoolService{
		TransactionPool: txPool,
	}
}
