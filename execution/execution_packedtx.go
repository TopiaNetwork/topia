package execution

import (
	"context"
	"errors"
	"fmt"
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/state"
	txfactory "github.com/TopiaNetwork/topia/transaction"
	"github.com/TopiaNetwork/topia/transaction/basic"
)

type executionPackedTxs struct {
	nodeID      string
	compState   state.CompositionState
	packedTxs   *PackedTxs
	packedTxsRS *PackedTxsResult
}

func newExecutionPackedTxs(nodeID string, packedTxs *PackedTxs, compState state.CompositionState) *executionPackedTxs {
	return &executionPackedTxs{
		nodeID:    nodeID,
		compState: compState,
		packedTxs: packedTxs,
	}
}

func (ept *executionPackedTxs) Execute(log tplog.Logger, ctx context.Context, txServant basic.TransactionServant) (*PackedTxsResult, error) {
	if len(ept.packedTxs.TxList) == 0 {
		return nil, errors.New("Empty packedTxs")
	}

	packedTxsRS := PackedTxsResult{
		StateVersion: ept.packedTxs.StateVersion,
	}

	log.Infof("Execution of packed txs start executing tx: state version %d, tx count %d, self node %s", ept.packedTxs.StateVersion, len(ept.packedTxs.TxList), ept.nodeID)
	for _, txItem := range ept.packedTxs.TxList {
		txRS := txfactory.CreatTransactionAction(txItem).Execute(ctx, log, ept.nodeID, txServant)

		if txRS == nil {
			txHexHash, _ := txItem.HashHex()
			err := fmt.Errorf("tx %s execute error", txHexHash)
			log.Errorf("%v", err)
			return &packedTxsRS, err
		}

		packedTxsRS.TxsResult = append(packedTxsRS.TxsResult, *txRS)
	}
	log.Infof("Execution of packed txs finish executing tx: state version %d, tx count %d, self node %s", ept.packedTxs.StateVersion, len(ept.packedTxs.TxList), ept.nodeID)

	packedTxsRS.TxRSRoot = basic.TxResultRoot(packedTxsRS.TxsResult, ept.packedTxs.TxList)

	return &packedTxsRS, nil
}

func (ept *executionPackedTxs) StateVersion() uint64 {
	return ept.packedTxs.StateVersion
}

func (ept *executionPackedTxs) PackedTxsResult() *PackedTxsResult {
	return ept.packedTxsRS
}
