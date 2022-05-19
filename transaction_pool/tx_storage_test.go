package transactionpool

import (
	"encoding/json"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/TopiaNetwork/topia/codec"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

func Test_transactionPool_AddLocal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog
	stateService := NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	blockService := NewMockBlockService(ctrl)
	network := NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].RemoteCount())
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	pool.AddLocal(Tx1)
	assert.Equal(t, 1, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 1, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].RemoteCount())
	assert.Equal(t, 1, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
}

func Test_transactionPool_AddRemote(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog
	stateService := NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	blockService := NewMockBlockService(ctrl)
	network := NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].RemoteCount())

	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	pool.AddRemote(TxR1)
	assert.Equal(t, 1, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 1, pool.allTxsForLook.all[Category1].RemoteCount())

	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 1, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
}

func Test_TransactionPool_AddRemotes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog
	stateService := NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	blockService := NewMockBlockService(ctrl)
	network := NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].RemoteCount())

	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	txs := make([]*txbasic.Transaction, 0)
	txs = append(txs, TxR1)
	txs = append(txs, TxR2)
	txsMap := make(map[tpcrtypes.Address][]*txbasic.Transaction)
	txsMap[From2] = txs
	pool.AddRemotes(txs)
	assert.Equal(t, 1, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 2, pool.queues.getTxListByAddrOfCategory(Category1, From2).txs.Len())
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 2, pool.allTxsForLook.all[Category1].RemoteCount())

	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 2, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
}

func Test_transactionPool_SaveRemoteTxs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog
	stateService := NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	blockService := NewMockBlockService(ctrl)
	network := NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
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
	log := TpiaLog
	stateService := NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	blockService := NewMockBlockService(ctrl)
	network := NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	//when run SetNewTransactionPool LoadRemoteTxs is load 2 txs from saved txs
	assert.Equal(t, 1, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 2, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	assert.Equal(t, 0, len(pool.sortedLists.getAllLocalTxsByCategory(Category1)))
	assert.Equal(t, 2, len(pool.sortedLists.getAllRemoteTxsByCategory(Category1)))

}

func Test_transactionPool_PublishTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog
	stateService := NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	blockService := NewMockBlockService(ctrl)
	network := NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].RemoteCount())

	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	if err := pool.txServant.PublishTx(pool.ctx, Tx1); err != nil {
		t.Error("want", nil, "got", err)
	}
}
