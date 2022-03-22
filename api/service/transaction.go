package service

import (
	"context"

	tx "github.com/TopiaNetwork/topia/transaction"
)

type Transaction struct {
}

func (tx *Transaction) SendTransaction(ctx context.Context, tran *tx.Transaction) error {
	panic("implement me")
}

func (tx *Transaction) TransactionByID(ctx context.Context, txHash tx.TxID) (*tx.Transaction, error) {
	panic("implement me")
}
