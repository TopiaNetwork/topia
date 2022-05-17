package transactionpool

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/TopiaNetwork/topia/codec"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

func Test_transactionPool_truncateQueue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].RemoteCount())

	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	keyLocals = make([]txbasic.TxID, 0)
	keyRemotes = make([]txbasic.TxID, 0)
	txLocals = make([]*txbasic.Transaction, 0)
	txRemotes = make([]*txbasic.Transaction, 0)

	for i := 1; i <= 400; i++ {
		nonce := uint64(i)
		gasprice := uint64(i * 1000)
		gaslimit := uint64(i * 1000000)
		txlocal = setTxLocal(nonce, gasprice, gaslimit)
		txlocal.Head.TimeStamp = starttime + uint64(i)
		if i > 1 {
			txlocal.Head.FromAddr = append(txlocal.Head.FromAddr, byte(i))
		}
		keylocal, _ = txlocal.TxID()
		keyLocals = append(keyLocals, keylocal)
		txLocals = append(txLocals, txlocal)

		txremote = setTxRemote(nonce, gasprice, gaslimit)
		txremote.Head.TimeStamp = starttime + uint64(i)
		if i > 1 {
			txremote.Head.FromAddr = append(txremote.Head.FromAddr, byte(i))
		}
		keyremote, _ = txremote.TxID()
		keyRemotes = append(keyRemotes, keyremote)
		txRemotes = append(txRemotes, txremote)
		_ = pool.AddTx(txlocal, true)
		_ = pool.AddTx(txremote, false)
	}
	assert.Equal(t, 449, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 400, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 192, pool.allTxsForLook.all[Category1].RemoteCount())

	assert.Equal(t, 400, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 192, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	pool.truncateQueueByCategory(Category1)
	assert.Equal(t, 257, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 400, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].RemoteCount())

	assert.Equal(t, 400, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))

}

func Test_transactionPool_truncatePending(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].RemoteCount())

	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	keyLocals = make([]txbasic.TxID, 0)
	keyRemotes = make([]txbasic.TxID, 0)
	txLocals = make([]*txbasic.Transaction, 0)
	txRemotes = make([]*txbasic.Transaction, 0)
	var fromlocal, fromremote tpcrtypes.Address

	for i := 1; i <= 400; i++ {
		nonce := uint64(i)
		gasprice := uint64(i * 1000)
		gaslimit := uint64(i * 1000000)
		txlocal = setTxLocal(nonce, gasprice, gaslimit)
		txlocal.Head.TimeStamp = starttime + uint64(i)
		if i > 1 {
			txlocal.Head.FromAddr = append(txlocal.Head.FromAddr, byte(i))
		}
		keylocal, _ = txlocal.TxID()
		keyLocals = append(keyLocals, keylocal)
		txLocals = append(txLocals, txlocal)

		txremote = setTxRemote(nonce, gasprice, gaslimit)
		txremote.Head.TimeStamp = starttime + uint64(i)
		if i > 1 {
			txremote.Head.FromAddr = append(txremote.Head.FromAddr, byte(i))
		}
		keyremote, _ = txremote.TxID()
		keyRemotes = append(keyRemotes, keyremote)
		txRemotes = append(txRemotes, txremote)
		fromlocal = tpcrtypes.Address(txlocal.Head.FromAddr)
		_ = pool.AddTx(txlocal, true)
		assert.Equal(t, 1, pool.queues.getTxListByAddrOfCategory(Category1, fromlocal).txs.Len())
		_ = pool.AddTx(txremote, false)
		fromremote = tpcrtypes.Address(txremote.Head.FromAddr)

		_ = pool.turnTx(fromlocal, keylocal, txlocal)

		_ = pool.turnTx(fromremote, keyremote, txremote)
	}
	assert.Equal(t, 449, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 449, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 400, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 192, pool.allTxsForLook.all[Category1].RemoteCount())

	assert.Equal(t, 400, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 192, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	pool.truncatePendingByCategory(Category1)
	assert.Equal(t, 449, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 449, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 400, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 192, pool.allTxsForLook.all[Category1].RemoteCount())

	assert.Equal(t, 400, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 192, len(pool.sortedLists.Pricedlist[Category1].all.remotes))

}
