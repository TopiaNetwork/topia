package transactionpool

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/account"
	"github.com/TopiaNetwork/topia/codec"
	"github.com/TopiaNetwork/topia/transaction"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"reflect"
	"testing"
)

func Test_transactionPool_AddLocal(t *testing.T) {
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
	pool.AddLocal(Tx1)
	assert.Equal(t, 1, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 1, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 1, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
}

func Test_transactionPool_LocalAccounts(t *testing.T) {
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
	txs := make([]*transaction.Transaction, 0)
	txs = append(txs, Tx1)
	txs = append(txs, TxR1)
	pool.AddLocals(txs)
	accounts := make([]account.Address, 0)
	accounts = append(accounts, From1)
	accounts = append(accounts, From2)
	want := accounts
	got := pool.LocalAccounts()
	if !assert.Equal(t, want, got) {
		t.Error("want:", want, "got:", got)
	}

}

func Test_transactionPool_LoadConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant

	pool.config.PathRemote = "newremote.json"
	if err := pool.SaveConfig(); err != nil {
		t.Error("want", nil, "got", err)
	}

	conf, _ := pool.LoadConfig()
	want := pool.config
	got := *conf
	if !assert.Equal(t, want, got) {
		t.Error("want", want, "got", got)
	}
}

func Test_transactionPool_UpdateTxPoolConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant

	pool.config.PathRemote = "newremote.json"
	if err := pool.SaveConfig(); err != nil {
		t.Error("want", nil, "got", err)
	}

	conf, _ := pool.LoadConfig()
	pool.UpdateTxPoolConfig(*conf)
	want := *conf
	got := pool.config
	if !assert.Equal(t, want, got) {
		t.Error("want", *conf, "got", pool.config)
	}

}

func Test_transactionPool_SaveLocalTxs(t *testing.T) {
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

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)

	if err := pool.SaveLocalTxs(); err != nil {
		t.Error("want", nil, "got", err)
	}
	data, err := ioutil.ReadFile(pool.config.PathLocal)
	if err != nil {
		t.Error("want", nil, "got", err)
	}
	var locals map[string]*transaction.Transaction
	err = json.Unmarshal(data, &locals)
	if err != nil {
		t.Error("want", nil, "got", err)
	}
	want := make(map[string]*transaction.Transaction, 0)
	got := make(map[string]*transaction.Transaction, 0)

	for k, v := range pool.allTxsForLook.locals {
		want[k] = v
	}
	for k, v := range locals {
		got[k] = v
	}
	for k, v := range want {
		if reflect.DeepEqual(got[k], v) {
			t.Error("want", v, "got", got[k])
		}
	}

}

func Test_transactionPool_LoadLocalTxs(t *testing.T) {
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
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)

	rawLocaltxs := pool.allTxsForLook.locals
	if err := pool.SaveLocalTxs(); err != nil {
		t.Error("want", nil, "got", err)
	}
	pool = SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	pool.LoadLocalTxs()
	if !assert.Equal(t, pool.queue.accTxs[From1].txs.Len(), 2) ||
		!assert.Equal(t, len(pool.allTxsForLook.locals), 2) {
		t.Error("want", rawLocaltxs, "got", pool.allTxsForLook.locals)
	}

}
