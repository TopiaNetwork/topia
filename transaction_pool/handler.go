package transactionpool

import (
	"context"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	tplog "github.com/TopiaNetwork/topia/log"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
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

func NewTransactionPoolHandler(log tplog.Logger, txPool *transactionPool) *transactionPoolHandler {
	return &transactionPoolHandler{
		log:    log,
		txPool: txPool,
	}
}

func (handler *transactionPoolHandler) ProcessTx(msg *TxMessage) error {
	var tx *txbasic.Transaction
	txId, _ := tx.TxID()
	err := tx.Unmarshal(msg.Data)
	if err != nil {
		handler.log.Error("txmessage data error")
		return err
	}
	if err := handler.txPool.ValidateTx(tx, false); err != nil {
		handler.txPool.txCache.Add(txId, StateTxInValid)
		return err
	}
	category := txbasic.TransactionCategory(tx.Head.Category)
	handler.txPool.newTxListStructs(category)
	if err := handler.txPool.AddTx(tx, false); err != nil {
		return err
	}

	handler.txPool.txCache.Add(txId, StateTxAdded)
	return nil
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
