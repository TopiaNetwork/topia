package consensus

import (
	"context"
	"github.com/TopiaNetwork/topia/execution"
	"github.com/TopiaNetwork/topia/ledger"
	"github.com/TopiaNetwork/topia/state"

	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tplog "github.com/TopiaNetwork/topia/log"
	tx "github.com/TopiaNetwork/topia/transaction"
	txpool "github.com/TopiaNetwork/topia/transaction_pool"
)

type consensusExecutor struct {
	log          tplog.Logger
	txPool       txpool.TransactionPool
	marshaler    codec.Marshaler
	ledger       ledger.Ledger
	exeScheduler execution.Executionscheduler
	deliver      *messageDeliver
}

func newConsensusExecutor(log tplog.Logger, txPool txpool.TransactionPool, marshaler codec.Marshaler, ledger ledger.Ledger, exeScheduler execution.Executionscheduler, deliver *messageDeliver) *consensusExecutor {
	return &consensusExecutor{
		log:          log,
		txPool:       txPool,
		marshaler:    marshaler,
		ledger:       ledger,
		exeScheduler: exeScheduler,
		deliver:      deliver,
	}
}

func (e *consensusExecutor) start() {

}

func (e *consensusExecutor) makePreparePackedMsg(txRoot []byte, txRSRoot []byte, compState state.CompositionState) (*PreparePackedMessage, error) {
	latestBlock, err := compState.GetLatestBlock()
	if err != nil {
		e.log.Errorf("can't get the latest bock when making prepare packed msg: %v", err)
		return nil, err
	}
	parentBlockHahs, _ := latestBlock.HashBytes(tpcmm.NewBlake2bHasher(0), e.marshaler)

	return &PreparePackedMessage{
		ChainID:         []byte(compState.ChainID()),
		Version:         CONSENSUS_VER,
		Epoch:           compState.GetCurrentEpoch(),
		Round:           compState.GetCurrentRound(),
		ParentBlockHash: parentBlockHahs,
		TxRoot:          txRoot,
		TxResultRoot:    txRSRoot,
	}, nil
}

func (e *consensusExecutor) Prepare(ctx context.Context) error {
	pendTxs, err := e.txPool.Pending()
	if err != nil {
		e.log.Errorf("Can't get pending txs: %v", err)
		return err
	}

	if len(pendTxs) == 0 {
		e.log.Debug("Current pending txs'size 0")
		return nil
	}

	txRoot := tx.TxRoot(pendTxs)
	compState := state.CreateCompositionState(e.log, e.ledger)

	var packedTxs execution.PackedTxs

	maxStateVer, err := e.exeScheduler.MaxStateVersion(compState)
	if err != nil {
		e.log.Errorf("Can't get max state version: %v", err)
		return err
	}

	packedTxs.StateVersion = maxStateVer + 1
	packedTxs.TxRoot = txRoot
	packedTxs.TxList = append(packedTxs.TxList, pendTxs...)

	txsRS, err := e.exeScheduler.ExecutePackedTx(ctx, &packedTxs, compState)
	if err != nil {
		e.log.Errorf("Execute state version %d packed txs err: %v", packedTxs.StateVersion, err)
		return err
	}
	txRSRoot := tx.TxResultRoot(txsRS.TxsResult, packedTxs.TxList)

	packedMsg, err := e.makePreparePackedMsg(txRoot, txRSRoot, compState)
	if err != nil {
		return err
	}

	return e.deliver.deliverPreparePackagedMessage(ctx, packedMsg)
}
