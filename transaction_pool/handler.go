package transactionpool

import (
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/transaction/basic"
)

type TransactionPoolHandler interface {
	ProcessTx(msg *TxMessage) error
}

type transactionPoolHandler struct {
	log    tplog.Logger
	txPool *transactionPool
}

func NewTransactionPoolHandler(log tplog.Logger, txPool *transactionPool) *transactionPoolHandler {
	return &transactionPoolHandler{
		log:    log,
		txPool: txPool,
	}
}

func (handler *transactionPoolHandler) ProcessTx(msg *TxMessage) error {
	var tx *basic.Transaction
	err := tx.Unmarshal(msg.Data)
	if err != nil {
		handler.log.Error("txmessage data error")
		return err
	}
	if err := handler.txPool.ValidateTx(tx, false); err != nil {
		return err
	}
	if err := handler.txPool.AddTx(tx, false); err != nil {
		return err
	}
	return nil
}
