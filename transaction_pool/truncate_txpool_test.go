package transactionpool

import (
	"encoding/hex"
	"github.com/TopiaNetwork/topia/account"
	"github.com/TopiaNetwork/topia/codec"
	"github.com/TopiaNetwork/topia/transaction"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_transactionPool_truncateQueue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))

	var nonce, gasprice, gaslimit uint64
	var txLocal, txRemote *transaction.Transaction
	txsLocal := make([]*transaction.Transaction, 0)
	txsRemote := make([]*transaction.Transaction, 0)
	var fromLocal, fromRemote account.Address
	var keyLocal, keyRemote string
	fromsLocal := make([]account.Address, 0)
	fromsRemote := make([]account.Address, 0)
	keysLocal := make([]string, 0)
	keysRemote := make([]string, 0)
	for i := 1; i <= 104; i++ {
		nonce = uint64(i)
		gasprice = uint64(1001 * i)
		gaslimit = uint64(10001)
		txLocal = settransactionlocal(nonce, gasprice, gaslimit)
		txLocal.FromAddr = append(txLocal.FromAddr, byte(i))
		txsLocal = append(txsLocal, txLocal)
		txRemote = settransactionremote(nonce, gasprice, gaslimit)
		txRemote.FromAddr = append(txRemote.FromAddr, byte(i))
		txsRemote = append(txsRemote, txRemote)
		fromLocal = account.Address(hex.EncodeToString(txLocal.FromAddr))
		fromsLocal = append(fromsLocal, fromLocal)
		fromRemote = account.Address(hex.EncodeToString(txRemote.FromAddr))
		fromsRemote = append(fromsRemote, fromRemote)
		keyLocal, _ = txLocal.TxID()
		keyRemote, _ = txRemote.TxID()
		keysLocal = append(keysLocal, keyLocal)
		keysRemote = append(keysRemote, keyRemote)
		_ = pool.AddTx(txLocal, true)
		assert.Equal(t, 1, pool.queue.accTxs[fromLocal].txs.Len())
		_ = pool.AddTx(txRemote, false)
		assert.Equal(t, 1, pool.queue.accTxs[fromRemote].txs.Len())

	}
	assert.Equal(t, 192, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 104, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 88, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 104, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 88, len(pool.sortedByPriced.all.remotes))
	pool.truncateQueue()
	assert.Equal(t, 128, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 104, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 24, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 104, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 24, len(pool.sortedByPriced.all.remotes))

}

func Test_transactionPool_truncatePending(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))

	var nonce, gasprice, gaslimit uint64
	var txLocal, txRemote *transaction.Transaction
	txsLocal := make([]*transaction.Transaction, 0)
	txsRemote := make([]*transaction.Transaction, 0)
	var fromLocal, fromRemote account.Address
	var keyLocal, keyRemote string
	fromsLocal := make([]account.Address, 0)
	fromsRemote := make([]account.Address, 0)
	keysLocal := make([]string, 0)
	keysRemote := make([]string, 0)
	for i := 1; i <= 104; i++ {
		nonce = uint64(i)
		gasprice = uint64(1001 * i)
		gaslimit = uint64(10001)
		txLocal = settransactionlocal(nonce, gasprice, gaslimit)
		txLocal.FromAddr = append(txLocal.FromAddr, byte(i))
		txsLocal = append(txsLocal, txLocal)
		txRemote = settransactionremote(nonce, gasprice, gaslimit)
		txRemote.FromAddr = append(txRemote.FromAddr, byte(i))
		txsRemote = append(txsRemote, txRemote)
		fromLocal = account.Address(hex.EncodeToString(txLocal.FromAddr))
		fromsLocal = append(fromsLocal, fromLocal)
		fromRemote = account.Address(hex.EncodeToString(txRemote.FromAddr))
		fromsRemote = append(fromsRemote, fromRemote)
		keyLocal, _ = txLocal.TxID()
		keyRemote, _ = txRemote.TxID()
		keysLocal = append(keysLocal, keyLocal)
		keysRemote = append(keysRemote, keyRemote)
		_ = pool.AddTx(txLocal, true)
		assert.Equal(t, 1, pool.queue.accTxs[fromLocal].txs.Len())
		_ = pool.AddTx(txRemote, false)
		assert.Equal(t, 1, pool.queue.accTxs[fromRemote].txs.Len())
		_ = pool.turnTx(fromLocal, keyLocal, txLocal)
		_ = pool.turnTx(fromRemote, keyRemote, txRemote)
		assert.Equal(t, 0, pool.queue.accTxs[fromLocal].txs.Len())
		assert.Equal(t, 0, pool.queue.accTxs[fromRemote].txs.Len())
	}
	assert.Equal(t, 192, len(pool.queue.accTxs))
	assert.Equal(t, 192, len(pool.pending.accTxs))
	assert.Equal(t, 104, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 88, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 104, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 88, len(pool.sortedByPriced.all.remotes))
	pool.truncatePending()
	assert.Equal(t, 192, len(pool.queue.accTxs))
	assert.Equal(t, 192, len(pool.pending.accTxs))
	assert.Equal(t, 104, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 88, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 104, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 88, len(pool.sortedByPriced.all.remotes))

}
