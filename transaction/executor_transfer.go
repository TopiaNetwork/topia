package transaction

import (
	"context"

	tplog "github.com/TopiaNetwork/topia/log"
)

type TransactionExecutorActionTran struct {
	log tplog.Logger
}

func (exet *TransactionExecutorActionTran) doExecute(ctx context.Context, tx *Transaction) ReturnCode {
	//TODO implement me
	panic("implement me")
}
