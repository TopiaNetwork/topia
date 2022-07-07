package transactionpool

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/TopiaNetwork/topia/codec"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txpoolmock "github.com/TopiaNetwork/topia/transaction_pool/mock"
)

func Test_transactionPool_AddLocal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog

	stateService := txpoolmock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := txpoolmock.NewMockBlockService(ctrl)
	network := txpoolmock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	pool.TruncateTxPool()

	assert.Equal(t, int64(0), pool.Count())

	pool.AddTx(Tx1, true)
	assert.Equal(t, int64(1), pool.poolCount)
	assert.Equal(t, int64(Tx1.Size()), pool.Size())
	assert.Equal(t, 1, len(pool.GetLocalTxs()))
	pool.AddTx(Tx2, true)
	assert.Equal(t, int64(2), pool.poolCount)
	assert.Equal(t, int64(Tx1.Size()+Tx2.Size()), pool.Size())
	assert.Equal(t, 2, len(pool.GetLocalTxs()))
}

func Test_TransactionPool_AddLocals(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog
	stateService := txpoolmock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)
	blockService := txpoolmock.NewMockBlockService(ctrl)
	network := txpoolmock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	pool.TruncateTxPool()

	assert.Equal(t, int64(0), pool.Count())

	txs := make([]*txbasic.Transaction, 0)
	txs = append(txs, Tx1)
	txs = append(txs, Tx2)

	pool.addTxs(txs, true)
	assert.Equal(t, int64(2), pool.poolCount)
	assert.Equal(t, int64(TxR1.Size()+TxR2.Size()), pool.Size())
	assert.Equal(t, 2, len(pool.GetLocalTxs()))

}
func Test_transactionPool_AddRemote(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog

	stateService := txpoolmock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := txpoolmock.NewMockBlockService(ctrl)
	network := txpoolmock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	pool.TruncateTxPool()

	assert.Equal(t, int64(0), pool.Count())

	pool.AddTx(TxR1, false)
	assert.Equal(t, int64(1), pool.poolCount)
	assert.Equal(t, int64(TxR1.Size()), pool.Size())
	assert.Equal(t, 1, len(pool.GetRemoteTxs()))
	pool.AddTx(TxR2, false)
	assert.Equal(t, int64(2), pool.poolCount)
	assert.Equal(t, int64(TxR1.Size()+TxR2.Size()), pool.Size())
	assert.Equal(t, 2, len(pool.GetRemoteTxs()))

}

func Test_TransactionPool_AddRemotes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog

	stateService := txpoolmock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := txpoolmock.NewMockBlockService(ctrl)
	network := txpoolmock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	pool.TruncateTxPool()

	assert.Equal(t, int64(0), pool.Count())

	txs := make([]*txbasic.Transaction, 0)
	txs = append(txs, TxR1)
	txs = append(txs, TxR2)

	pool.addTxs(txs, false)
	assert.Equal(t, int64(2), pool.poolCount)
	assert.Equal(t, int64(TxR1.Size()+TxR2.Size()), pool.Size())
	assert.Equal(t, 2, len(pool.GetRemoteTxs()))

}

func Test_transactionPool_SaveTxsData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog

	stateService := txpoolmock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := txpoolmock.NewMockBlockService(ctrl)
	network := txpoolmock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	pool.TruncateTxPool()

	assert.Equal(t, int64(0), pool.Count())

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)

	if err := pool.SaveAllLocalTxsData(pool.config.PathTxsStorage); err != nil {
		t.Error("want", nil, "got", err)
	}

	pool.TruncateTxPool()

	if err := pool.LoadLocalTxsData(pool.config.PathTxsStorage); err != nil {
		t.Error("want", nil, "got", err)

	}
	assert.Equal(t, int64(2), pool.poolCount)

}

func Test_transactionPool_PublishTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog

	stateService := txpoolmock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := txpoolmock.NewMockBlockService(ctrl)
	network := txpoolmock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	pool.TruncateTxPool()

	assert.Equal(t, int64(0), pool.Count())

	if err := pool.txServant.PublishTx(pool.ctx, Tx1); err != nil {
		t.Error("want", nil, "got", err)
	}
}

func Test_transactionPool_ClearLocalFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog

	stateService := txpoolmock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := txpoolmock.NewMockBlockService(ctrl)
	network := txpoolmock.NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	pool.TruncateTxPool()

	assert.Equal(t, int64(0), pool.Count())

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(Tx3, true)
	pool.AddTx(TxR2, false)

	locals := make([]*txbasic.Transaction, 0)
	locals = append(locals, Tx1)
	locals = append(locals, Tx2)
	locals = append(locals, Tx3)

	if err := pool.SaveAllLocalTxsData(pool.config.PathTxsStorage); err != nil {
		t.Error("want", nil, "got", err)
	}
	pathIndex := pool.config.PathTxsStorage + "index.json"
	pathData := pool.config.PathTxsStorage + "data.json"
	_, err := ioutil.ReadFile(pathIndex)
	assert.Equal(t, err, nil)
	_, err = ioutil.ReadFile(pathData)
	assert.Equal(t, err, nil)

	pool.ClearLocalFile(pathIndex)
	_, err = ioutil.ReadFile(pathIndex)
	var isEmpty bool
	if err != nil {
		isEmpty = true
	}
	if !reflect.DeepEqual(true, isEmpty) {
		t.Error("want", true, "got", isEmpty)

	}

	pool.ClearLocalFile(pathData)
	_, err = ioutil.ReadFile(pathData)
	if err != nil {
		isEmpty = true
	}
	if !reflect.DeepEqual(true, isEmpty) {
		t.Error("want", true, "got", isEmpty)

	}

}
