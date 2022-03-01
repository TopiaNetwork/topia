package transactionpool

import (
	tplog "github.com/TopiaNetwork/topia/log"
	"github.com/TopiaNetwork/topia/transaction"
	"github.com/sirupsen/logrus"
)

type TransactionPoolHandler interface {
	ProcessTx(msg *TxMessage) error
}

type transactionPoolHandler struct {
	log 		tplog.Logger
	txPool 		TransactionPool
}

func NewTransactionPoolHandler(log tplog.Logger, txPool TransactionPool) *transactionPoolHandler {
	return &transactionPoolHandler{
		log: log,
		txPool: txPool,
	}
}

func (handler *transactionPoolHandler) ProcessTx(msg *TxMessage) error {
	var tx *transaction.Transaction
	err := tx.Unmarshal(msg.Data)
	if err != nil {
		logrus.Errorf("txmessage data error")
		return err
	}
	handler.txPool.AddTx(tx,false)
	return nil
}

func(handler *transactionPoolHandler) BroadCastTx(tx *transaction.Transaction) (*TxMessage,error) {
	var msg *TxMessage
	data := tx.GetData()
	msg.Data = data
	_,err := msg.Marshal()
	if err != nil {
		return nil,err
	}
	return msg,nil
}
