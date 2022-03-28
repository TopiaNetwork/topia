package transactionpool

import (
	"errors"
	"github.com/TopiaNetwork/topia/codec"
	"github.com/TopiaNetwork/topia/transaction"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_transactionPool_loop_chanRemoveTxHashs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	servant.EXPECT().CurrentBlock().Return(NewBlock).AnyTimes()
	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	defer pool.wg.Wait()
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
	assert.Equal(t, 2, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 2, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 2, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 2, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 2, len(pool.sortedByPriced.all.remotes))
	var hashs []string
	hashs = append(hashs, Key1)
	hashs = append(hashs, Key2)
	hashs = append(hashs, KeyR1)
	hashs = append(hashs, KeyR2)

	pool.wg.Add(1)
	go pool.chanRemoveTxHashs()
	//pool.wg.Add(1)
	//go pool.saveAllIfShutDown()
	//pool.wg.Add(1)
	//go pool.resetIfNewHead()
	//pool.wg.Add(1)
	//go pool.reportTicks()
	//pool.wg.Add(1)
	//go pool.removeTxForUptoLifeTime()
	//pool.wg.Add(1)
	//go pool.regularSaveLocalTxs()
	//pool.wg.Add(1)
	//go pool.regularRepublic()
	pool.chanRmTxs <- hashs
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
}

func Test_transactionPool_loop_saveAllIfShutDown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	servant.EXPECT().CurrentBlock().Return(NewBlock).AnyTimes()

	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant

	newnetwork := NewMockNetwork(ctrl)
	newnetwork.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	pool.network = newnetwork
	defer pool.wg.Wait()
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
	assert.Equal(t, 2, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 2, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 2, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 2, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 2, len(pool.sortedByPriced.all.remotes))
	var hashs []string
	hashs = append(hashs, Key1)
	hashs = append(hashs, Key2)
	hashs = append(hashs, KeyR1)
	hashs = append(hashs, KeyR2)

	//pool.wg.Add(1)
	//go pool.chanRemoveTxHashs()

	//check new local files:localTransactions.json,remoteTransactions.json,txPoolConfigs.json
	pool.wg.Add(1)
	go pool.saveAllIfShutDown()
	//pool.wg.Add(1)
	//go pool.resetIfNewHead()
	//pool.wg.Add(1)
	//go pool.reportTicks()
	//pool.wg.Add(1)
	//go pool.removeTxForUptoLifeTime()
	//pool.wg.Add(1)
	//go pool.regularSaveLocalTxs()
	//pool.wg.Add(1)
	//go pool.regularRepublic()
	pool.chanSysShutDown <- errors.New("shut down")

}

func Test_transactionPool_loop_resetIfNewHead(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	servant.EXPECT().CurrentBlock().Return(NewBlock).AnyTimes()

	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant

	newnetwork := NewMockNetwork(ctrl)
	newnetwork.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	pool.network = newnetwork
	defer pool.wg.Wait()
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
	assert.Equal(t, 2, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 2, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 2, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 2, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 2, len(pool.sortedByPriced.all.remotes))
	var hashs []string
	hashs = append(hashs, Key1)
	hashs = append(hashs, Key2)
	hashs = append(hashs, KeyR1)
	hashs = append(hashs, KeyR2)

	//pool.wg.Add(1)
	//go pool.chanRemoveTxHashs()
	//pool.wg.Add(1)
	//go pool.saveAllIfShutDown()
	pool.wg.Add(1)
	go pool.resetIfNewHead()
	//pool.wg.Add(1)
	//go pool.reportTicks()
	//pool.wg.Add(1)
	//go pool.removeTxForUptoLifeTime()
	//pool.wg.Add(1)
	//go pool.regularSaveLocalTxs()
	//pool.wg.Add(1)
	//go pool.regularRepublic()
	newheadevent := &transaction.ChainHeadEvent{NewBlock}
	pool.chanChainHead <- *newheadevent
}

func Test_transactionPool_loop_reportTicks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	servant.EXPECT().CurrentBlock().Return(NewBlock).AnyTimes()

	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant

	newnetwork := NewMockNetwork(ctrl)
	newnetwork.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	pool.network = newnetwork
	defer pool.wg.Wait()
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
	assert.Equal(t, 2, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 2, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 2, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 2, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 2, len(pool.sortedByPriced.all.remotes))
	var hashs []string
	hashs = append(hashs, Key1)
	hashs = append(hashs, Key2)
	hashs = append(hashs, KeyR1)
	hashs = append(hashs, KeyR2)

	//pool.wg.Add(1)
	//go pool.chanRemoveTxHashs()
	//pool.wg.Add(1)
	//go pool.saveAllIfShutDown()
	//pool.wg.Add(1)
	//go pool.resetIfNewHead()
	pool.wg.Add(1)
	go pool.reportTicks()
	//pool.wg.Add(1)
	//go pool.removeTxForUptoLifeTime()
	//pool.wg.Add(1)
	//go pool.regularSaveLocalTxs()
	//pool.wg.Add(1)
	//go pool.regularRepublic()

}

func Test_transactionPool_removeTxForUptoLifeTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	servant.EXPECT().CurrentBlock().Return(NewBlock).AnyTimes()

	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant

	newnetwork := NewMockNetwork(ctrl)
	newnetwork.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	pool.network = newnetwork
	defer pool.wg.Wait()
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
	assert.Equal(t, 2, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 2, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 2, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 2, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 2, len(pool.sortedByPriced.all.remotes))
	var hashs []string
	hashs = append(hashs, Key1)
	hashs = append(hashs, Key2)
	hashs = append(hashs, KeyR1)
	hashs = append(hashs, KeyR2)

	//pool.wg.Add(1)
	//go pool.chanRemoveTxHashs()
	//pool.wg.Add(1)
	//go pool.saveAllIfShutDown()
	//pool.wg.Add(1)
	//go pool.resetIfNewHead()
	//pool.wg.Add(1)
	//go pool.reportTicks()

	//change lifTime to trigger
	pool.wg.Add(1)
	go pool.removeTxForUptoLifeTime()
	//pool.wg.Add(1)
	//go pool.regularSaveLocalTxs()
	//pool.wg.Add(1)
	//go pool.regularRepublic()

	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
}

func Test_transactionPool_regularSaveLocalTxs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	servant.EXPECT().CurrentBlock().Return(NewBlock).AnyTimes()

	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant

	newnetwork := NewMockNetwork(ctrl)
	newnetwork.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	pool.network = newnetwork
	defer pool.wg.Wait()
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
	assert.Equal(t, 2, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 2, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 2, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 2, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 2, len(pool.sortedByPriced.all.remotes))
	var hashs []string
	hashs = append(hashs, Key1)
	hashs = append(hashs, Key2)
	hashs = append(hashs, KeyR1)
	hashs = append(hashs, KeyR2)

	//pool.wg.Add(1)
	//go pool.chanRemoveTxHashs()
	//pool.wg.Add(1)
	//go pool.saveAllIfShutDown()
	//pool.wg.Add(1)
	//go pool.resetIfNewHead()
	//pool.wg.Add(1)
	//go pool.reportTicks()

	//change lifTime to trigger
	//pool.wg.Add(1)
	//go pool.removeTxForUptoLifeTime()

	//check new local file:localTransactions.json,
	pool.wg.Add(1)
	go pool.regularSaveLocalTxs()
	//pool.wg.Add(1)
	//go pool.regularRepublic()

}

func Test_transactionPool_regularRepublic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	servant.EXPECT().CurrentBlock().Return(NewBlock).AnyTimes()

	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant

	newnetwork := NewMockNetwork(ctrl)
	newnetwork.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	pool.network = newnetwork
	defer pool.wg.Wait()
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
	assert.Equal(t, 2, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 2, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 2, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 2, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 2, len(pool.sortedByPriced.all.remotes))
	var hashs []string
	hashs = append(hashs, Key1)
	hashs = append(hashs, Key2)
	hashs = append(hashs, KeyR1)
	hashs = append(hashs, KeyR2)

	//pool.wg.Add(1)
	//go pool.chanRemoveTxHashs()
	//pool.wg.Add(1)
	//go pool.saveAllIfShutDown()
	//pool.wg.Add(1)
	//go pool.resetIfNewHead()
	//pool.wg.Add(1)
	//go pool.reportTicks()

	//change lifTime to trigger
	//pool.wg.Add(1)
	//go pool.removeTxForUptoLifeTime()

	//check new local file:localTransactions.json,
	//pool.wg.Add(1)
	//go pool.regularSaveLocalTxs()
	pool.wg.Add(1)
	go pool.regularRepublic()

}
