package transactionpool

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/codec"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/transaction/basic"
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
	pool := SetNewTransactionPool(Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].RemoteCount())
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	pool.AddLocal(Tx1)
	assert.Equal(t, 1, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 1, pool.allTxsForLook.all[Category1].LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].RemoteCount())
	assert.Equal(t, 1, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
}

func Test_transactionPool_LocalAccounts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].RemoteCount())
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	txs := make([]*basic.Transaction, 0)
	txs = append(txs, Tx1)
	txs = append(txs, TxR1)
	pool.AddLocals(txs)
	accounts := make([]tpcrtypes.Address, 0)
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
	pool := SetNewTransactionPool(Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant

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
	pool := SetNewTransactionPool(Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant

	//pool.config.PathRemote[Category1] = "newremote.json"
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
	pool := SetNewTransactionPool(Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].RemoteCount())
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)

	if err := pool.SaveLocalTxs(Category1); err != nil {
		t.Error("want", nil, "got", err)
	}
	data, err := ioutil.ReadFile(pool.config.PathLocal[Category1])
	if err != nil {
		t.Error("want", nil, "got", err)
	}
	var locals map[string]*basic.Transaction
	err = json.Unmarshal(data, &locals)
	if err != nil {
		t.Error("want", nil, "got", err)
	}
	wants := make(map[string]*basic.Transaction, 0)
	gots := make(map[string]*basic.Transaction, 0)

	for k, v := range pool.allTxsForLook.all[Category1].locals {
		wants[k] = v
	}
	for k, v := range locals {
		gots[k] = v
	}

	for k, v := range wants {
		got := gots[k]
		want := v
		if !reflect.DeepEqual(got, want) {
			t.Error("want", want, "got", got)
		}
	}
}

func Test_transactionPool_LoadLocalTxs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].RemoteCount())
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)

	rawLocaltxs := pool.allTxsForLook.all[Category1].locals
	if err := pool.SaveLocalTxs(Category1); err != nil {
		t.Error("want", nil, "got", err)
	}
	pool = SetNewTransactionPool(Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	pool.LoadLocalTxs(Category1)
	if !assert.Equal(t, pool.queues.getTxListByAddrOfCategory(Category1, From1).txs.Len(), 2) ||
		!assert.Equal(t, len(pool.allTxsForLook.all[Category1].locals), 2) {
		t.Error("want", rawLocaltxs, "got", pool.allTxsForLook.all[Category1].locals)
	}

}
