package transactionpool

import (
	"context"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tplog "github.com/TopiaNetwork/topia/log"
)

type TransactionPoolHandler interface {
	ProcessTx(msg *TxMessage) error
	processBlockAddedEvent(context.Context, interface{}) error
}

type transactionPoolHandler struct {
	log      tplog.Logger
	txPool   *transactionPool
	txMsgSub TxMessageSubProcessor
}

func NewTransactionPoolHandler(log tplog.Logger, txPool *transactionPool, txMsgSub TxMessageSubProcessor) *transactionPoolHandler {
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
		newChainHead := &BlockAddedEvent{
			block,
		}
		handler.txPool.chanBlockAdded <- *newChainHead
	}
	return nil
}
