package service

import (
	"context"
	tptypes "github.com/TopiaNetwork/topia/common/types"
	"github.com/TopiaNetwork/topia/transaction"
)

type Transaction struct {
}

func (tx *Transaction) SendTransaction(ctx context.Context, tran *transaction.Transaction) error {
	panic("implement me")
}

func (tx *Transaction) TransactionByID(ctx context.Context, txHash tptypes.TxID) (*transaction.Transaction, error) {
	panic("implement me")
}
