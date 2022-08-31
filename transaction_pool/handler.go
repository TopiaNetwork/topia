package transactionpool

import (
	"context"
	"github.com/TopiaNetwork/topia/codec"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	"runtime"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

type TransactionPoolHandler interface {
	ProcessTx(ctx context.Context, msg *TxMessage) error
	processBlockAddedEvent(context.Context, interface{}) error
}

type transactionPoolHandler struct {
	log      tplog.Logger
	txPool   *transactionPool
	txMsgSub TxMsgSubProcessor
}

func NewTransactionPoolHandler(log tplog.Logger, txPool *transactionPool, txMsgSub TxMsgSubProcessor) *transactionPoolHandler {
	return &transactionPoolHandler{
		log:      log,
		txPool:   txPool,
		txMsgSub: txMsgSub,
	}
}

func (handler *transactionPoolHandler) ProcessTx(ctx context.Context, msg *TxMessage) error {
	return handler.txMsgSub.Process(ctx, msg)
}

func (handler *transactionPoolHandler) processBlockAddedEvent(ctx context.Context, data interface{}) error {
	/*if block, ok := data.(*tpchaintypes.Block); ok {
		handler.txPool.chanBlockAdded <- block
	}*/

	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	if block, ok := data.(*tpchaintypes.Block); ok {
		for _, dataChunkBytes := range block.Data.DataChunks {
			var dataChunk tpchaintypes.BlockDataChunk
			dataChunk.Unmarshal(dataChunkBytes)
			for _, txBytes := range dataChunk.Txs {
				var tx txbasic.Transaction
				marshaler.Unmarshal(txBytes, &tx)
				txID, _ := tx.TxID()
				handler.txPool.RemoveTxByKey(txID)
			}
		}

		runtime.GC()

		return nil
	}

	panic("Unknown sub event data")
}

func (handler *transactionPoolHandler) processBlocksRevertEvent(ctx context.Context, data interface{}) error {
	if blocks, ok := data.([]*tpchaintypes.Block); ok {

		handler.txPool.chanBlocksRevert <- blocks
	}
	return nil
}
