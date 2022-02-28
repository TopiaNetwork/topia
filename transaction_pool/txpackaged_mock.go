package transactionpool

import (
	"github.com/TopiaNetwork/topia/account"
	"github.com/TopiaNetwork/topia/transaction"
)

type TxsPackaged struct {
	TxPool TransactionPool
}

func (pkg *TxsPackaged)TxsPackaged(txs map[account.Address][]*transaction.Transaction,txsType int){
	switch txsType {
	case 0:
		pkg.CommitTxsForPending(txs)
		pkg.TxPool.RemoveTxs(txs)
	case 1:
		pkg.CommitTxsByPriceAndNonce(txs)
		pkg.TxPool.RemoveTxs(txs)
	}
}


// CommitTxsForPending  : Block packaged transactions for pending
func (pkg *TxsPackaged) CommitTxsForPending(txs map[account.Address][]*transaction.Transaction) map[account.Address][]*transaction.Transaction {

	return txs
}
// CommitTxsByPriceAndNonce  : Block packaged transactions sorted by price and nonce
func (pkg *TxsPackaged) CommitTxsByPriceAndNonce(txs map[account.Address][]*transaction.Transaction) map[account.Address][]*transaction.Transaction{

	txset := transaction.NewTxsByPriceAndNonce(pkg.TxPool.Signer,txs)
	txs = make(map[account.Address][]*transaction.Transaction,0)
	for {
		tx := txset.Peek()
		if tx == nil {
			break
		}
		from,_ := transaction.Sender(pkg.TxPool.Signer,tx)
		txs[from] = append(txs[from],tx)
	}
	return txs
}
