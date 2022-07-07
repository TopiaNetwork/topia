package transactionpool

import (
	"context"

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
	if block, ok := data.(*tpchaintypes.Block); ok {
		handler.txPool.chanBlockAdded <- block
	}
	return nil
}

func (handler *transactionPoolHandler) processBlocksRevertEvent(ctx context.Context, data interface{}) error {
	if blocks, ok := data.([]*tpchaintypes.Block); ok {

		handler.txPool.chanBlocksRevert <- blocks
	}
	return nil
}
