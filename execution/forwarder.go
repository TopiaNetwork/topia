package execution

import (
	"context"
	"encoding/json"
	"fmt"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tpnetmsg "github.com/TopiaNetwork/topia/network/message"
	"math/rand"
	"time"

	"github.com/TopiaNetwork/topia/codec"
	tpcmm "github.com/TopiaNetwork/topia/common"
	"github.com/TopiaNetwork/topia/ledger"
	tplog "github.com/TopiaNetwork/topia/log"
	tpnet "github.com/TopiaNetwork/topia/network"
	tpnetcmn "github.com/TopiaNetwork/topia/network/common"
	tpnetprotoc "github.com/TopiaNetwork/topia/network/protocol"
	"github.com/TopiaNetwork/topia/state"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txuni "github.com/TopiaNetwork/topia/transaction/universal"
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
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
	txPool    txpooli.TransactionPool
}

func NewExecutionForwarder(nodeID string,
	log tplog.Logger,
	marshaler codec.Marshaler,
	network tpnet.Network,
	ledger ledger.Ledger,
	txPool txpooli.TransactionPool) ExecutionForwarder {
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

	txID, _ := tx.TxID()

	if tpcmm.IsContainString(forwarder.nodeID, activeExecutors) {
		err := forwarder.txPool.AddTx(tx, true)
		if err != nil {
			forwarder.log.Errorf("Add local tx to pool err: txID %s %v", txID, err)
			return err
		}
	} else {
		ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_PeerList, activeExecutors)
		ctx = context.WithValue(ctx, tpnetcmn.NetContextKey_RouteStrategy, tpnetcmn.RouteStrategy_NearestBucket)
		txBytes, err := forwarder.marshaler.Marshal(tx)
		if err != nil {
			forwarder.log.Errorf("Tx marshal err: txID %s %v", txID, err)
			return err
		}

		respList, err := forwarder.network.SendWithResponse(ctx, tpnetprotoc.ForwardExecute_SyncTx, tpchaintypes.MOD_NAME, txBytes)
		if err != nil {
			forwarder.log.Errorf("Send tx to executors err: txID %s %v", txID, err)
			return err
		}

		var respErrs []string
		for i, resp := range respList {
			errMsg := func() string {
				respErrI := fmt.Sprintf("resp%d, nodeID %s:", i, resp.NodeID)
				if resp.Err != nil {
					respErrI += resp.Err.Error()
					return respErrI
				}
				var respData tpnetmsg.ResponseData
				err = json.Unmarshal(resp.RespData, &respData)
				if err != nil {
					respErrI += err.Error()
					return respErrI
				}
				if respData.ErrMsg != "" {
					respErrI += respData.ErrMsg
					return respErrI
				}

				forwarder.log.Infof("Tx pool successfully add tx %s from %", txID, resp.NodeID)
				return ""
			}()
			if errMsg != "" {
				respErrs = append(respErrs, errMsg)
				continue
			}

			return nil
		}

		forwarder.log.Errorf("All executor can't add tx %s: %v", txID, respErrs)
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
