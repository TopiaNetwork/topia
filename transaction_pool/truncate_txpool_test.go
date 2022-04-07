package transactionpool

import (
	"github.com/TopiaNetwork/topia/codec"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/transaction/basic"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_transactionPool_truncateQueue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues[Category1].accTxs))
	assert.Equal(t, 0, len(pool.pendings[Category1].accTxs))
	assert.Equal(t, 0, pool.allTxsForLook[Category1].LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook[Category1].RemoteCount())

	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	keyCatLocals = make(map[string]basic.TransactionCategory, 0)
	keyCatRemotes = make(map[string]basic.TransactionCategory, 0)
	txLocals = make([]*basic.Transaction, 0)
	txRemotes = make([]*basic.Transaction, 0)
	var fromlocal, fromremote tpcrtypes.Address

	for i := 1; i <= 200; i++ {
		nonce := uint64(i)
		gasprice := uint64(i * 1000)
		gaslimit := uint64(i * 1000000)
		txlocal = setTxLocal(nonce, gasprice, gaslimit)
		txlocal.Head.TimeStamp = starttime + uint64(i)
		if i > 1 {
			txlocal.Head.FromAddr = append(txlocal.Head.FromAddr, byte(i))
		}
		keylocal, _ = txlocal.HashHex()
		keyCatLocals[keylocal] = basic.TransactionCategory_Topia_Universal
		txLocals = append(txLocals, txlocal)

		txremote = setTxRemote(nonce, gasprice, gaslimit)
		txremote.Head.TimeStamp = starttime + uint64(i)
		if i > 1 {
			txremote.Head.FromAddr = append(txremote.Head.FromAddr, byte(i))
		}
		keyremote, _ = txremote.HashHex()
		keyCatRemotes[keyremote] = basic.TransactionCategory_Topia_Universal
		txRemotes = append(txRemotes, txremote)
		fromlocal = tpcrtypes.Address(txlocal.Head.FromAddr)
		_ = pool.AddTx(txlocal, true)
		assert.Equal(t, 1, pool.queues[Category1].accTxs[fromlocal].txs.Len())
		_ = pool.AddTx(txremote, false)
		fromremote = tpcrtypes.Address(txremote.Head.FromAddr)
		assert.Equal(t, 1, pool.queues[Category1].accTxs[fromremote].txs.Len())
		//fmt.Printf("i:%d,", i)
		//fmt.Println("len(pool.queues[Category1].accTxs)", len(pool.queues[Category1].accTxs))
	}
	assert.Equal(t, 384, len(pool.queues[Category1].accTxs))
	assert.Equal(t, 0, len(pool.pendings[Category1].accTxs))
	assert.Equal(t, 200, pool.allTxsForLook[Category1].LocalCount())
	assert.Equal(t, 184, pool.allTxsForLook[Category1].RemoteCount())

	assert.Equal(t, 200, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 184, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	pool.truncateQueue(Category1)
	assert.Equal(t, 256, len(pool.queues[Category1].accTxs))
	assert.Equal(t, 0, len(pool.pendings[Category1].accTxs))
	assert.Equal(t, 200, pool.allTxsForLook[Category1].LocalCount())
	assert.Equal(t, 56, pool.allTxsForLook[Category1].RemoteCount())

	assert.Equal(t, 200, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 56, len(pool.sortedLists.Pricedlist[Category1].all.remotes))

}

func Test_transactionPool_truncatePending(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues[Category1].accTxs))
	assert.Equal(t, 0, len(pool.pendings[Category1].accTxs))
	assert.Equal(t, 0, pool.allTxsForLook[Category1].LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook[Category1].RemoteCount())

	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	keyCatLocals = make(map[string]basic.TransactionCategory, 0)
	keyCatRemotes = make(map[string]basic.TransactionCategory, 0)
	txLocals = make([]*basic.Transaction, 0)
	txRemotes = make([]*basic.Transaction, 0)
	var fromlocal, fromremote tpcrtypes.Address

	for i := 1; i <= 200; i++ {
		nonce := uint64(i)
		gasprice := uint64(i * 1000)
		gaslimit := uint64(i * 1000000)
		txlocal = setTxLocal(nonce, gasprice, gaslimit)
		txlocal.Head.TimeStamp = starttime + uint64(i)
		if i > 1 {
			txlocal.Head.FromAddr = append(txlocal.Head.FromAddr, byte(i))
		}
		keylocal, _ = txlocal.HashHex()
		keyCatLocals[keylocal] = basic.TransactionCategory_Topia_Universal
		txLocals = append(txLocals, txlocal)

		txremote = setTxRemote(nonce, gasprice, gaslimit)
		txremote.Head.TimeStamp = starttime + uint64(i)
		if i > 1 {
			txremote.Head.FromAddr = append(txremote.Head.FromAddr, byte(i))
		}
		keyremote, _ = txremote.HashHex()
		keyCatRemotes[keyremote] = basic.TransactionCategory_Topia_Universal
		txRemotes = append(txRemotes, txremote)
		fromlocal = tpcrtypes.Address(txlocal.Head.FromAddr)
		_ = pool.AddTx(txlocal, true)
		assert.Equal(t, 1, pool.queues[Category1].accTxs[fromlocal].txs.Len())
		_ = pool.AddTx(txremote, false)
		fromremote = tpcrtypes.Address(txremote.Head.FromAddr)
		assert.Equal(t, 1, pool.queues[Category1].accTxs[fromremote].txs.Len())
		//fmt.Printf("i:%d,", i)
		//fmt.Println("len(pool.queues[Category1].accTxs)", len(pool.queues[Category1].accTxs))
		_ = pool.turnTx(fromlocal, keylocal, txlocal)
		_ = pool.turnTx(fromremote, keyremote, txremote)
		assert.Equal(t, 0, pool.queues[Category1].accTxs[fromlocal].txs.Len())
		assert.Equal(t, 0, pool.queues[Category1].accTxs[fromremote].txs.Len())
	}
	assert.Equal(t, 384, len(pool.queues[Category1].accTxs))
	assert.Equal(t, 384, len(pool.pendings[Category1].accTxs))
	assert.Equal(t, 200, pool.allTxsForLook[Category1].LocalCount())
	assert.Equal(t, 184, pool.allTxsForLook[Category1].RemoteCount())

	assert.Equal(t, 200, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 184, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	pool.truncatePending(Category1)
	assert.Equal(t, 384, len(pool.queues[Category1].accTxs))
	assert.Equal(t, 384, len(pool.pendings[Category1].accTxs))
	assert.Equal(t, 200, pool.allTxsForLook[Category1].LocalCount())
	assert.Equal(t, 184, pool.allTxsForLook[Category1].RemoteCount())

	assert.Equal(t, 200, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 184, len(pool.sortedLists.Pricedlist[Category1].all.remotes))

}
