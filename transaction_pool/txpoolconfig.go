package transactionpool

import (
	txpooli "github.com/TopiaNetwork/topia/transaction_pool/interface"
)

func (pool *transactionPool) SetTxPoolConfig(conf txpooli.TransactionPoolConfig) {
	conf = (conf).Check()
	pool.config = conf
	return
}
