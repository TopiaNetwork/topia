package service

import (
	"context"

	"github.com/TopiaNetwork/topia/transaction"
)

type Transaction struct {
}

func (tx *Transaction) SendTransaction(ctx context.Context, tran *transaction.Transaction) error {
	panic("implement me")
}

func (tx *Transaction) TransactionByID(ctx context.Context, txHash transaction.TxID) (*transaction.Transaction, error) {
	panic("implement me")
}
