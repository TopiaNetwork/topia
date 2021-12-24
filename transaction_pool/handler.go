package transactionpool

import tplog "github.com/TopiaNetwork/topia/log"

type TransactionPoolHandler interface {
	ProcessTx(msg *TxMessage) error
}

type transactionPoolHandler struct {
	log tplog.Logger
}

func NewTransactionPoolHandler(log tplog.Logger) *transactionPoolHandler {
	return &transactionPoolHandler{
		log: log,
	}
}

func (handler *transactionPoolHandler) ProcessTx(msg *TxMessage) error {
	panic("implement me")
}
