package transactionpool

import (
	"encoding/hex"
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
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	keyLocals = make([]string, 0)
	keyRemotes = make([]string, 0)
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
		keyLocals = append(keyLocals, keylocal)
		txLocals = append(txLocals, txlocal)

		txremote = setTxRemote(nonce, gasprice, gaslimit)
		txremote.Head.TimeStamp = starttime + uint64(i)
		if i > 1 {
			txremote.Head.FromAddr = append(txremote.Head.FromAddr, byte(i))
		}
		keyremote, _ = txremote.HashHex()
		keyRemotes = append(keyRemotes, keyremote)
		txRemotes = append(txRemotes, txremote)
		fromlocal = tpcrtypes.Address(hex.EncodeToString(txlocal.Head.FromAddr))
		_ = pool.AddTx(txlocal, true)
		assert.Equal(t, 1, pool.queue.accTxs[fromlocal].txs.Len())
		_ = pool.AddTx(txremote, false)
		fromremote = tpcrtypes.Address(hex.EncodeToString(txremote.Head.FromAddr))
		assert.Equal(t, 1, pool.queue.accTxs[fromremote].txs.Len())
		//fmt.Printf("i:%d,", i)
		//fmt.Println("len(pool.queue.accTxs)", len(pool.queue.accTxs))
	}
	assert.Equal(t, 384, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 200, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 184, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 200, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 184, len(pool.sortedByPriced.all.remotes))
	pool.truncateQueue()
	assert.Equal(t, 256, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 200, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 56, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 200, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 56, len(pool.sortedByPriced.all.remotes))

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
	keyLocals = make([]string, 0)
	keyRemotes = make([]string, 0)
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
		keyLocals = append(keyLocals, keylocal)
		txLocals = append(txLocals, txlocal)

		txremote = setTxRemote(nonce, gasprice, gaslimit)
		txremote.Head.TimeStamp = starttime + uint64(i)
		if i > 1 {
			txremote.Head.FromAddr = append(txremote.Head.FromAddr, byte(i))
		}
		keyremote, _ = txremote.HashHex()
		keyRemotes = append(keyRemotes, keyremote)
		txRemotes = append(txRemotes, txremote)
		fromlocal = tpcrtypes.Address(hex.EncodeToString(txlocal.Head.FromAddr))
		_ = pool.AddTx(txlocal, true)
		assert.Equal(t, 1, pool.queue.accTxs[fromlocal].txs.Len())
		_ = pool.AddTx(txremote, false)
		fromremote = tpcrtypes.Address(hex.EncodeToString(txremote.Head.FromAddr))
		assert.Equal(t, 1, pool.queue.accTxs[fromremote].txs.Len())
		//fmt.Printf("i:%d,", i)
		//fmt.Println("len(pool.queue.accTxs)", len(pool.queue.accTxs))
		_ = pool.turnTx(fromlocal, keylocal, txlocal)
		_ = pool.turnTx(fromremote, keyremote, txremote)
		assert.Equal(t, 0, pool.queue.accTxs[fromlocal].txs.Len())
		assert.Equal(t, 0, pool.queue.accTxs[fromremote].txs.Len())
	}
	assert.Equal(t, 384, len(pool.queue.accTxs))
	assert.Equal(t, 384, len(pool.pending.accTxs))
	assert.Equal(t, 200, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 184, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 200, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 184, len(pool.sortedByPriced.all.remotes))
	pool.truncatePending()
	assert.Equal(t, 384, len(pool.queue.accTxs))
	assert.Equal(t, 384, len(pool.pending.accTxs))
	assert.Equal(t, 200, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 184, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 200, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 184, len(pool.sortedByPriced.all.remotes))

}
