package service

import (
	"context"
	"github.com/TopiaNetwork/topia/transaction/basic"
)

type Transaction struct {
}

func (tx *Transaction) SendTransaction(ctx context.Context, tran *basic.Transaction) error {
	panic("implement me")
}

func (tx *Transaction) TransactionByID(ctx context.Context, txHash basic.TxID) (*basic.Transaction, error) {
	panic("implement me")
}
