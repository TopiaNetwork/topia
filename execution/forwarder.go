package execution

import (
	"context"

	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type ExecutionForwarder interface {
	ForwardTxSync(ctx context.Context, tx *txbasic.Transaction) (*txbasic.TransactionResult, error)

	ForwardTxAsync(ctx context.Context, tx *txbasic.Transaction) error
}
