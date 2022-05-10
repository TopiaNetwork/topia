package action

import (
	"context"
	"math/big"

	tplog "github.com/TopiaNetwork/topia/log"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type Verifiable interface {
	Verify(ctx context.Context, log tplog.Logger, nodeID string, txServant txbasic.TransactionServant) txbasic.VerifyResult
}

type Executable interface {
	Execute(ctx context.Context, log tplog.Logger, nodeID string, txServant txbasic.TransactionServant) *txbasic.TransactionResult
}

type Estimable interface {
	Estimate(ctx context.Context, log tplog.Logger, nodeID string, txServant txbasic.TransactionServant) (*big.Int, error)
}

type TransactionAction interface {
	Verifiable
	Estimable
	Executable
}
