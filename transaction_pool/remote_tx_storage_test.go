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

func Test_transactionPool_AddRemote(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues.getTxsByCategory(Category1).getAll()))
	assert.Equal(t, 0, len(pool.pendings.getTxsByCategory(Category1).getAll()))
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].RemoteCount())

	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	pool.AddRemote(TxR1)
	assert.Equal(t, 1, len(pool.queues.getTxsByCategory(Category1).getAll()))
	assert.Equal(t, 0, len(pool.pendings.getTxsByCategory(Category1).getAll()))
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].LocalCount())
	assert.Equal(t, 1, pool.allTxsForLook.all[Category1].RemoteCount())

	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 1, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
}

func Test_TransactionPool_AddRemotes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues.getTxsByCategory(Category1).getAll()))
	assert.Equal(t, 0, len(pool.pendings.getTxsByCategory(Category1).getAll()))
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].RemoteCount())

	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	txs := make([]*basic.Transaction, 0)
	txs = append(txs, TxR1)
	txs = append(txs, TxR2)
	txsMap := make(map[tpcrtypes.Address][]*basic.Transaction)
	txsMap[From2] = txs
	pool.AddRemotes(txs)
	assert.Equal(t, 1, len(pool.queues.getTxsByCategory(Category1).getAll()))
	assert.Equal(t, 2, pool.queues.getTxsByCategory(Category1).getTxListByAddr(From2).txs.Len())
	assert.Equal(t, 0, len(pool.pendings.getTxsByCategory(Category1).getAll()))
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].LocalCount())
	assert.Equal(t, 2, pool.allTxsForLook.all[Category1].RemoteCount())

	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 2, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
}

func Test_transactionPool_SaveRemoteTxs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues.getTxsByCategory(Category1).getAll()))
	assert.Equal(t, 0, len(pool.pendings.getTxsByCategory(Category1).getAll()))
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].RemoteCount())

	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)

	if err := pool.SaveRemoteTxs(Category1); err != nil {
		t.Error("want", nil, "got", err)
	}
	data, err := ioutil.ReadFile(pool.config.PathRemote[Category1])
	if err != nil {
		t.Error("want", nil, "got", err)
	}
	remotetxs := &remoteTxs{}
	err = json.Unmarshal(data, &remotetxs)
	if err != nil {
		t.Error("want", nil, "got", err)
	}

	for k, v := range remotetxs.Txs {
		want := *pool.allTxsForLook.all[Category1].remotes[k]
		got := *v
		if !reflect.DeepEqual(want, got) {
			t.Error("want", want, "got", got)
		}
	}

}

func Test_transactionPool_LoadRemoteTxs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues.getTxsByCategory(Category1).getAll()))
	assert.Equal(t, 0, len(pool.pendings.getTxsByCategory(Category1).getAll()))
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].RemoteCount())

	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)

	want := pool.allTxsForLook.all[Category1].remotes
	if err := pool.SaveRemoteTxs(Category1); err != nil {
		t.Error("want", nil, "got", err)
	}
	pool = SetNewTransactionPool(Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	if err := pool.LoadRemoteTxs(Category1); err != nil {
		t.Error("want", nil, "got", err)
	}

	got := pool.allTxsForLook.all[Category1].remotes
	for k, v := range want {
		if !assert.Equal(t, *v, *got[k]) {
			t.Error("want", *v, "got", *got[k])
		}

	}
}

func Test_transactionPool_BroadCastTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	network := NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	pool.network = network
	assert.Equal(t, 0, len(pool.queues.getTxsByCategory(Category1).getAll()))
	assert.Equal(t, 0, len(pool.pendings.getTxsByCategory(Category1).getAll()))
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].RemoteCount())

	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	if err := pool.BroadCastTx(Tx1); err != nil {
		t.Error("want", nil, "got", err)
	}
}
