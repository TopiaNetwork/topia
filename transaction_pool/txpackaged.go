package transactionpool

import (
	"github.com/TopiaNetwork/topia/account"
	"github.com/TopiaNetwork/topia/transaction"
)

type txsPackaged interface {
	CommitTransactions(map[account.Address][]*transaction.Transaction, int)
	CommitTxsForPending(map[account.Address][]*transaction.Transaction) map[account.Address][]*transaction.Transaction
	CommitTxsByPriceAndNonce(map[account.Address][]*transaction.Transaction) map[account.Address][]*transaction.Transaction
}

