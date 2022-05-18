package types

import (
	"context"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type TxInterface interface {
	SendTransaction(ctx context.Context, tran *txbasic.Transaction) error

	TransactionByID(ctx context.Context, txHash txbasic.TxID) (*txbasic.Transaction, error)
}
