package transaction

import (
	"context"

	tplog "github.com/TopiaNetwork/topia/log"
)

type TransactionExecutorActionBuildinInV struct {
	log tplog.Logger
}

func (exet *TransactionExecutorActionBuildinInV) doExecute(ctx context.Context, tx *Transaction) ReturnCode {
	//TODO implement me
	panic("implement me")
}
