package service

import (
	"github.com/TopiaNetwork/topia/transaction/basic"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

type TxPoolService interface {
	PickTxs() []*basic.Transaction

	Size() int64

	//UpdateTx(tx *basic.Transaction, txID basic.TxID) error
}

type txPoolService struct {
	txpooli.TransactionPool
}

func NewTxPoolService(txPool txpooli.TransactionPool) TxPoolService {
	return &txPoolService{
		TransactionPool: txPool,
	}
}
