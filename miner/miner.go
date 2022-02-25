package miner

import (
	"github.com/TopiaNetwork/topia/account"
	"github.com/TopiaNetwork/topia/transaction"
	transactionpool "github.com/TopiaNetwork/topia/transaction_pool"
	"github.com/btcsuite/btcd/blockchain"
)



type Miner struct {
	TxPool    	*transactionpool.TransactionPool
	BlockChain blockchain.BlockChain
}
//txsType: 0:pending,1:txsByPriceAndNonce
func (m *Miner) commitTransactions(txs map[account.Address][]*transaction.Transaction,txsType int){
	switch txsType {
	case 0:
		m.CommitTxsForPending(txs)
		m.TxPool.RemoveTxs(txs)
	case 1:
		m.CommitTxsByPriceAndNonce(txs)
		m.TxPool.RemoveTxs(txs)
	}
}

// CommitTxsForPending  : Block packaged transactions for pending
func (m *Miner) CommitTxsForPending(txs map[account.Address][]*transaction.Transaction) map[account.Address][]*transaction.Transaction {

	return txs
}
// CommitTxsByPriceAndNonce  : Block packaged transactions sorted by price and nonce
func (m *Miner) CommitTxsByPriceAndNonce(txs map[account.Address][]*transaction.Transaction) map[account.Address][]*transaction.Transaction{

	txset := transaction.NewTxsByPriceAndNonce(m.TxPool.Signer,txs)
	txs = make(map[account.Address][]*transaction.Transaction,0)
	for {
		tx := txset.Peek()
		if tx == nil {
			break
		}
		from,_ := transaction.Sender(m.TxPool.Signer,tx)
		txs[from] = append(txs[from],tx)
	}
	return txs
}
