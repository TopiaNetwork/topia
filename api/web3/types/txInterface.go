package types

import (
	"context"
	"github.com/TopiaNetwork/topia/transaction/basic"
)

type TxInterface interface {
	SendTransaction(ctx context.Context, tran *basic.Transaction) error

	TransactionByID(ctx context.Context, txHash basic.TxID) (*basic.Transaction, error)
}
