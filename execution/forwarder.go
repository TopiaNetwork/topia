package execution

import (
	"context"
	"fmt"
	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/ledger"
	tpnet "github.com/TopiaNetwork/topia/network"
	tpnetcmn "github.com/TopiaNetwork/topia/network/common"
	tpnetprotoc "github.com/TopiaNetwork/topia/network/protocol"
	"github.com/TopiaNetwork/topia/state"
	txuni "github.com/TopiaNetwork/topia/transaction/universal"
	txpool "github.com/TopiaNetwork/topia/transaction_pool"
	"math/rand"
	"time"

	tplog "github.com/TopiaNetwork/topia/log"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

type ExecutionForwarder interface {
	ForwardTxSync(ctx context.Context, tx *txbasic.Transaction) (*txbasic.TransactionResult, error)

	ForwardTxAsync(ctx context.Context, tx *txbasic.Transaction) error
}

type executionForwarder struct {
	nodeID    string
	log       tplog.Logger
	marshaler codec.Marshaler
	network   tpnet.Network
	ledger    ledger.Ledger
	txPool    txpool.TransactionPool
}

func NewExecutionForwarder(nodeID string,
	log tplog.Logger,
	marshaler codec.Marshaler,
	network tpnet.Network,
	ledger ledger.Ledger,
	txPool txpool.TransactionPool) ExecutionForwarder {
	return &executionForwarder{
		nodeID:    nodeID,
		log:       log,
		marshaler: marshaler,
		network:   network,
		ledger:    ledger,
		txPool:    txPool,
	}

}

func (forwarder executionForwarder) sendTx(ctx context.Context, tx *txbasic.Transaction) error {
	compStateRN := state.CreateCompositionStateReadonly(forwarder.log, forwarder.ledger)
	activeExecutors, _ := compStateRN.GetActiveExecutorIDs()

	if tpcmm.IsContainString(forwarder.nodeID, activeExecutors) {
		err := forwarder.txPool.AddTx(tx, true)
		if err != nil {
			forwarder.log.Errorf("Add local tx to pool err: %v", err)
			return err
		}
	} else {
		ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_PeerList, activeExecutors)
		ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
		txBytes, err := forwarder.marshaler.Marshal(tx)
		if err != nil {
			forwarder.log.Errorf("Tx marshal err: err=%v", err)
			return err
		}

		err = forwarder.network.Send(ctx, tpnetprotoc.AsyncSendProtocolID, txpool.MOD_NAME, txBytes)
		if err != nil {
			forwarder.log.Errorf("Send tx to executors err: err=%v", err)
			return err
		}
	}

	return nil
}

func (forwarder executionForwarder) ForwardTxSync(ctx context.Context, tx *txbasic.Transaction) (*txbasic.TransactionResult, error) {
	err := forwarder.sendTx(ctx, tx)
	if err != nil {
		return nil, err
	}

	txID, _ := tx.TxID()
	txHashBytes, _ := tx.HashBytes()
	blockStore := forwarder.ledger.GetBlockStore()
	count := 0
	startAt := time.Now()
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		count++
		select {
		case <-timer.C:
			txResult, err := blockStore.GetTransactionResultByID(txID)
			if err != nil {
				jitter := 100*time.Millisecond + time.Duration(rand.Int63n(int64(time.Second))) // nolint: gosec
				backoff := 100 * time.Duration(count) * time.Millisecond
				timer.Reset(jitter + backoff)
				continue
			}

			return txResult, nil
		case <-ctx.Done():
			switch txbasic.TransactionCategory(tx.Head.Category) {
			case txbasic.TransactionCategory_Topia_Universal:
				err = fmt.Errorf("Send sucessfully, but wait for result time out: tx %s waiting time %s", txID, time.Since(startAt).String())
				txHead := tx.GetHead()

				txUniRS := &txuni.TransactionResultUniversal{
					Version:   txHead.Version,
					TxHash:    txHashBytes,
					GasUsed:   0,
					ErrString: []byte(err.Error()),
					Status:    txuni.TransactionResultUniversal_Err,
				}
				txUniRSBytes, err := forwarder.marshaler.Marshal(txUniRS)
				if err != nil {
					return nil, err
				}

				return &txbasic.TransactionResult{
					Head: &txbasic.TransactionResultHead{
						Category: txHead.Category,
						Version:  txHead.Version,
						ChainID:  txHead.ChainID,
					},
					Data: &txbasic.TransactionResultData{
						Specification: txUniRSBytes,
					},
				}, nil
			}
		}
	}
}

func (forwarder executionForwarder) ForwardTxAsync(ctx context.Context, tx *txbasic.Transaction) error {
	return forwarder.sendTx(ctx, tx)
}
