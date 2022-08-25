package transactionpool

import (
	"io/ioutil"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"

	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

func Test_transactionPool_AddLocal(t *testing.T) {
	pool1.TruncateTxPool()

	assert.Equal(t, int64(0), pool1.Count())

	pool1.AddTx(Tx1, true)
	assert.Equal(t, int64(1), pool1.poolCount)
	assert.Equal(t, int64(Tx1.Size()), pool1.Size())
	assert.Equal(t, 1, len(pool1.GetLocalTxs()))
	pool1.AddTx(Tx2, true)
	assert.Equal(t, int64(2), pool1.poolCount)
	assert.Equal(t, int64(Tx1.Size()+Tx2.Size()), pool1.Size())
	assert.Equal(t, 2, len(pool1.GetLocalTxs()))
}

func Test_TransactionPool_AddLocals(t *testing.T) {
	pool1.TruncateTxPool()

	assert.Equal(t, int64(0), pool1.Count())

	txs := make([]*txbasic.Transaction, 0)
	txs = append(txs, Tx1)
	txs = append(txs, Tx2)

	pool1.addTxs(txs, true)
	assert.Equal(t, int64(2), pool1.poolCount)
	assert.Equal(t, int64(Tx1.Size()+Tx2.Size()), pool1.Size())
	assert.Equal(t, 2, len(pool1.GetLocalTxs()))

}
func Test_transactionPool_AddRemote(t *testing.T) {
	pool1.TruncateTxPool()

	assert.Equal(t, int64(0), pool1.Count())

	pool1.AddTx(TxR1, false)
	assert.Equal(t, int64(1), pool1.poolCount)
	assert.Equal(t, int64(TxR1.Size()), pool1.Size())
	assert.Equal(t, 1, len(pool1.GetRemoteTxs()))
	pool1.AddTx(TxR2, false)
	assert.Equal(t, int64(2), pool1.poolCount)
	assert.Equal(t, int64(TxR1.Size()+TxR2.Size()), pool1.Size())
	assert.Equal(t, 2, len(pool1.GetRemoteTxs()))

}

func Test_TransactionPool_AddRemotes(t *testing.T) {
	pool1.TruncateTxPool()

	assert.Equal(t, int64(0), pool1.Count())

	txs := make([]*txbasic.Transaction, 0)
	txs = append(txs, TxR1)
	txs = append(txs, TxR2)

	pool1.addTxs(txs, false)
	assert.Equal(t, int64(2), pool1.poolCount)
	assert.Equal(t, int64(TxR1.Size()+TxR2.Size()), pool1.Size())
	assert.Equal(t, 2, len(pool1.GetRemoteTxs()))

}

func Test_transactionPool_SaveTxsData(t *testing.T) {
	pool1.TruncateTxPool()

	assert.Equal(t, int64(0), pool1.Count())

	pool1.AddTx(Tx1, true)
	pool1.AddTx(Tx2, true)
	pool1.AddTx(TxR1, false)
	pool1.AddTx(TxR2, false)

	if err := pool1.SaveAllLocalTxsData(pool1.config.PathTxsStorage); err != nil {
		t.Error("want", nil, "got", err)
	}

	pool1.pending = newAccTxs()
	pool1.prepareTxs = newAccTxs()
	pool1.allWrappedTxs = newAllLookupTxs()
	pool1.pendingNonces = newAccountNonce(pool1.txServant.getStateQuery())
	atomic.StoreInt64(&pool1.pendingCount, 0)
	atomic.StoreInt64(&pool1.pendingSize, 0)
	atomic.StoreInt64(&pool1.poolCount, 0)
	atomic.StoreInt64(&pool1.poolSize, 0)
	pool1.txCache.Purge()
	if err := pool1.LoadLocalTxsData(pool1.config.PathTxsStorage); err != nil {
		t.Error("want", nil, "got", err)

	}
	assert.Equal(t, int64(2), pool1.poolCount)

}

func Test_transactionPool_PublishTx(t *testing.T) {
	pool1.TruncateTxPool()

	assert.Equal(t, int64(0), pool1.Count())

	if err := pool1.txServant.PublishTx(pool1.ctx, Tx1); err != nil {
		t.Error("want", nil, "got", err)
	}
}

func Test_transactionPool_ClearLocalFile(t *testing.T) {
	pool1.TruncateTxPool()

	assert.Equal(t, int64(0), pool1.Count())

	pool1.AddTx(Tx1, true)
	pool1.AddTx(Tx2, true)
	pool1.AddTx(Tx3, true)
	pool1.AddTx(TxR2, false)

	locals := make([]*txbasic.Transaction, 0)
	locals = append(locals, Tx1)
	locals = append(locals, Tx2)
	locals = append(locals, Tx3)

	if err := pool1.SaveAllLocalTxsData(pool1.config.PathTxsStorage); err != nil {
		t.Error("want", nil, "got", err)
	}
	pathIndex := pool1.config.PathTxsStorage + "index.json"
	pathData := pool1.config.PathTxsStorage + "data.json"
	_, err := ioutil.ReadFile(pathIndex)
	assert.Equal(t, err, nil)
	_, err = ioutil.ReadFile(pathData)
	assert.Equal(t, err, nil)

	pool1.ClearLocalFile(pathIndex)
	_, err = ioutil.ReadFile(pathIndex)
	var isEmpty bool
	if err != nil {
		isEmpty = true
	}
	if !reflect.DeepEqual(true, isEmpty) {
		t.Error("want", true, "got", isEmpty)

	}

	pool1.ClearLocalFile(pathData)
	_, err = ioutil.ReadFile(pathData)
	if err != nil {
		isEmpty = true
	}
	if !reflect.DeepEqual(true, isEmpty) {
		t.Error("want", true, "got", isEmpty)
	}

}
