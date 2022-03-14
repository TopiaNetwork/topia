package transaction

import "context"

type TransactionAction interface {
	Verify(ctx context.Context)
	Execute(ctx context.Context)
}
