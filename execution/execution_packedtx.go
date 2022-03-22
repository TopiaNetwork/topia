package execution

import (
	"context"
	"errors"
	"fmt"

	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/state"
	tx "github.com/TopiaNetwork/topia/transaction"
)

type executionPackedTxs struct {
	compState   state.CompositionState
	packedTxs   *PackedTxs
	packedTxsRS *PackedTxsResult
}

func newExecutionPackedTxs(packedTxs *PackedTxs, compState state.CompositionState) *executionPackedTxs {
	return &executionPackedTxs{
		compState: compState,
		packedTxs: packedTxs,
	}
}

func (ept *executionPackedTxs) Execute(log tplog.Logger, ctx context.Context, txServant tx.TansactionServant) (*PackedTxsResult, error) {
	if len(ept.packedTxs.TxList) == 0 {
		return nil, errors.New("Empty packedTxs")
	}

	packedTxsRS := PackedTxsResult{
		StateVersion: ept.packedTxs.StateVersion,
	}

	for _, txItem := range ept.packedTxs.TxList {
		txRS := txItem.TxAction().Execute(ctx, log, txServant)
		if txRS == nil {
			txHexHash, _ := txItem.HashHex()
			err := fmt.Errorf("tx %s execute error", txHexHash)
			log.Errorf("%v", err)
			return &packedTxsRS, err
		}

		if txRS.Status != tx.TransactionResult_OK {
			err := fmt.Errorf("tx %s execute error %s", txRS.ErrString)
			log.Errorf("%v", err)
			return &packedTxsRS, err
		}
	}

	packedTxsRS.TxRSRoot = tx.TxResultRoot(packedTxsRS.TxsResult, ept.packedTxs.TxList)

	return &packedTxsRS, nil
}

func (ept *executionPackedTxs) StateVersion() uint64 {
	return ept.packedTxs.StateVersion
}

func (ept *executionPackedTxs) PackedTxsResult() *PackedTxsResult {
	return ept.packedTxsRS
}
