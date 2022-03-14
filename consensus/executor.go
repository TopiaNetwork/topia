package consensus

import (
	"context"

	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	tplog "github.com/TopiaNetwork/topia/log"
	tx "github.com/TopiaNetwork/topia/transaction"
	txpool "github.com/TopiaNetwork/topia/transaction_pool"
)

type consensusExecutor struct {
	log        tplog.Logger
	txPool     txpool.TransactionPool
	marshaler  codec.Marshaler
	epochState EpochState
	csState    consensusStore
	deliver    *messageDeliver
}

func newConsensusExecutor(log tplog.Logger, txPool txpool.TransactionPool, marshaler codec.Marshaler, epochState EpochState, csState consensusStore, deliver *messageDeliver) *consensusExecutor {
	return &consensusExecutor{
		log:        log,
		txPool:     txPool,
		marshaler:  marshaler,
		epochState: epochState,
		csState:    csState,
		deliver:    deliver,
	}
}

func (e *consensusExecutor) start() {

}

func (e *consensusExecutor) makePreparePackedMsg(txs []tx.Transaction) (*PreparePackedMessage, error) {
	latestBlock, err := e.csState.GetLatestBlock()
	if err != nil {
		e.log.Errorf("can't get the latest bock when making prepare packed msg: %v", err)
		return nil, err
	}
	parentBlockHahs, _ := latestBlock.HashBytes(tpcmm.NewBlake2bHasher(0), e.marshaler)

	txRoot := tx.TxRoot(tpcmm.NewBlake2bHasher(0), e.marshaler, txs)

	return &PreparePackedMessage{
		ChainID:         []byte(e.csState.ChainID()),
		Version:         CONSENSUS_VER,
		Epoch:           e.epochState.GetCurrentEpoch(),
		Round:           e.epochState.GetCurrentRound(),
		ParentBlockHash: parentBlockHahs,
		TxRoot:          txRoot,
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

	for _, tx := range pendTxs {

	}

	packedMsg, err := e.makePreparePackedMsg(pendTxs)
	if err != nil {
		return err
	}

	return e.deliver.deliverPreparePackagedMessage(ctx, packedMsg)
}
