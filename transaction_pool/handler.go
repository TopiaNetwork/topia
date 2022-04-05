package transactionpool

import (
	"github.com/sirupsen/logrus"

	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/transaction/basic"
)

type TransactionPoolHandler interface {
	ProcessTx(msg *TxMessage) error
}

type transactionPoolHandler struct {
	log    tplog.Logger
	txPool TransactionPool
}

func NewTransactionPoolHandler(log tplog.Logger, txPool TransactionPool) *transactionPoolHandler {
	return &transactionPoolHandler{
		log:    log,
		txPool: txPool,
	}
}

func (handler *transactionPoolHandler) ProcessTx(msg *TxMessage) error {
	var tx *basic.Transaction
	err := tx.Unmarshal(msg.Data)
	if err != nil {
		logrus.Errorf("txmessage data error")
		return err
	}
	handler.txPool.AddTx(tx, false)
	return nil
}
