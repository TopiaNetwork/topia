package action

import (
	"context"

	tplog "github.com/TopiaNetwork/topia/log"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type Verifiable interface {
	Verify(ctx context.Context, log tplog.Logger, nodeID string, txServant txbasic.TransactionServant) txbasic.VerifyResult
}

type Executable interface {
	Execute(ctx context.Context, log tplog.Logger, nodeID string, txServant txbasic.TransactionServant) *txbasic.TransactionResult
}

type TransactionAction interface {
	Verifiable
	Executable
}
