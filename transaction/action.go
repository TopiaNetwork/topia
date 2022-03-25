package transaction

import (
	"context"

	tplog "github.com/TopiaNetwork/topia/log"
)

type Verifiable interface {
	Verify(ctx context.Context, log tplog.Logger, txServant TansactionServant) VerifyResult
}

type Executable interface {
	Execute(ctx context.Context, log tplog.Logger, txServant TansactionServant) *TransactionResult
}

type TransactionAction interface {
	Verifiable
	Executable
}
