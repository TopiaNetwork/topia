package transactionpool

import (
	"errors"
	"fmt"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/TopiaNetwork/topia/codec"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
)

func Test_transactionPool_loop_chanRemoveTxHashs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog
	stateService := NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	blockService := NewMockBlockService(ctrl)
	network := NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)

	//assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	//assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	//assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	//assert.Equal(t, 0, pool.allTxsForLook.all[Category1].RemoteCount())
	//
	//assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	//assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	fmt.Println(1)
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	fmt.Println(2)
	assert.Equal(t, 2, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 2, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 2, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	fmt.Println(3)
	hashs := make([]txbasic.TxID, 0)
	hashs = append(hashs, Key1)
	hashs = append(hashs, Key2)
	hashs = append(hashs, KeyR1)
	hashs = append(hashs, KeyR2)
	pool.RemoveTxHashs(hashs)
	var hashs1, hashs2 []txbasic.TxID
	var hash txbasic.TxID
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	for _, tx := range txLocals[1:10] {
		hash, _ = tx.TxID()
		hashs1 = append(hashs1, hash)
		pool.AddTx(tx, false)
	}
	fmt.Println(7)

	for _, tx := range txLocals[20:40] {
		hash, _ = tx.TxID()
		hashs2 = append(hashs2, hash)
		pool.AddTx(tx, false)
	}
	pool.wg.Add(1)
	go pool.loopChanRemoveTxHashs()

	pool.wg.Add(1)
	go func() {
		defer pool.wg.Done()
		pool.chanRmTxs <- hashs1
	}()
	pool.wg.Add(1)
	go func() {
		defer pool.wg.Done()
		pool.chanRmTxs <- hashs2
	}()
	pool.wg.Add(1)
	go func() {
		defer pool.wg.Done()
		pool.ctx.Done()
	}()
	pool.wg.Wait()
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	fmt.Println("test001")

}

func Test_transactionPool_loop_saveAllIfShutDown(t *testing.T) {
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
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	assert.Equal(t, 2, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 2, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 2, pool.allTxsForLook.getRemoteCountByCategory(Category1))

	//pool.wg.Add(1)
	//go pool.chanRemoveTxHashs()

	//check new local files:localTransactions.json,remoteTransactions.json,txPoolConfigs.json
	pool.wg.Add(1)
	go pool.loopSaveAllIfShutDown()

	pool.wg.Add(1)
	go func() {
		pool.wg.Done()
		pool.chanSysShutdown <- errors.New("shut down")
	}()
	pool.wg.Wait()

}

func Test_transactionPool_loop_resetIfNewHead(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	log := TpiaLog
	stateService := NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	blockService := NewMockBlockService(ctrl)
	blockService.EXPECT().GetBlockByHash(tpchaintypes.BlockHash(OldBlock.Head.Hash)).AnyTimes().Return(OldBlock, nil)
	blockService.EXPECT().GetBlockByHash(tpchaintypes.BlockHash(NewBlock.Head.Hash)).AnyTimes().Return(NewBlock, nil)
	network := NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)

	defer pool.wg.Wait()
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	assert.Equal(t, 2, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 2, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 2, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	var hashs []txbasic.TxID
	hashs = append(hashs, Key1)
	hashs = append(hashs, Key2)
	hashs = append(hashs, KeyR1)
	hashs = append(hashs, KeyR2)

	pool.wg.Add(1)
	go pool.loopResetIfBlockAdded()
	newheadevent := &BlockAddedEvent{NewBlock}
	pool.chanBlockAdded <- *newheadevent
}

func Test_transactionPool_removeTxForUptoLifeTime(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog
	stateService := NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	blockService := NewMockBlockService(ctrl)
	network := NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)

	defer pool.wg.Wait()
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	assert.Equal(t, 2, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 2, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 2, pool.allTxsForLook.getRemoteCountByCategory(Category1))

	waitChannel := make(chan struct{})

	//***********change lifTime to trigger**************
	pool.wg.Add(1)
	go pool.loopRemoveTxForUptoLifeTime()

	<-waitChannel
	time.Sleep(10 * time.Second)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))

}

func Test_transactionPool_regularSaveLocalTxs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog
	stateService := NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	blockService := NewMockBlockService(ctrl)
	network := NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)

	defer pool.wg.Wait()
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	assert.Equal(t, 2, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 2, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 2, pool.allTxsForLook.getRemoteCountByCategory(Category1))

	//check new local file:localTransactions.json,
	pool.wg.Add(1)
	go pool.loopRegularSaveLocalTxs()

}

func Test_transactionPool_regularRepublic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	log := TpiaLog
	stateService := NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	blockService := NewMockBlockService(ctrl)
	network := NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)

	defer pool.wg.Wait()
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	assert.Equal(t, 2, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 2, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 2, pool.allTxsForLook.getRemoteCountByCategory(Category1))

	pool.wg.Add(1)
	go pool.loopRegularRepublic()

}

func Test_transactionPool_loop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	log := TpiaLog
	stateService := NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	blockService := NewMockBlockService(ctrl)
	blockService.EXPECT().GetBlockByHash(tpchaintypes.BlockHash(OldBlock.Head.Hash)).AnyTimes().Return(OldBlock, nil)
	blockService.EXPECT().GetBlockByHash(tpchaintypes.BlockHash(NewBlock.Head.Hash)).AnyTimes().Return(NewBlock, nil)
	network := NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)

	defer pool.wg.Wait()

	keyLocals = make([]txbasic.TxID, 0)
	keyRemotes = make([]txbasic.TxID, 0)
	txLocals = make([]*txbasic.Transaction, 0)
	txRemotes = make([]*txbasic.Transaction, 0)
	var fromlocal, fromremote tpcrtypes.Address
	for i := 1; i <= 100; i++ {
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
		fromlocal = tpcrtypes.Address(txlocal.Head.FromAddr)

		txremote = setTxRemote(nonce, gasprice, gaslimit)
		txremote.Head.TimeStamp = starttime + uint64(i)
		if i > 1 {
			txremote.Head.FromAddr = append(txremote.Head.FromAddr, byte(i))
		}
		keyremote, _ = txremote.TxID()
		keyRemotes = append(keyRemotes, keyremote)
		txRemotes = append(txRemotes, txremote)
		fromremote = tpcrtypes.Address(txremote.Head.FromAddr)

		//	fmt.Printf("i == %d:start,\n", i)
		_ = pool.AddTx(txlocal, true)
		assert.Equal(t, 1, pool.queues.getTxListByAddrOfCategory(Category1, fromlocal).txs.Len())
		//	fmt.Printf("i == %d:1,addLocaltx%v:\n", i, err)
		_ = pool.AddTx(txremote, false)
		assert.Equal(t, 1, pool.queues.getTxListByAddrOfCategory(Category1, fromremote).txs.Len())
	}

	assert.Equal(t, 200, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 100, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 100, pool.allTxsForLook.getRemoteCountByCategory(Category1))

	pool.loopChanSelect()

	go func() {
		pool.chanRmTxs <- keyLocals
	}()
	go func() {
		pool.chanRmTxs <- keyRemotes
	}()

	waitChannel := make(chan struct{})

	newheadevent := &BlockAddedEvent{NewBlock}
	pool.chanBlockAdded <- *newheadevent
	pool.chanSysShutdown <- errors.New("shut down")

	<-waitChannel
	time.Sleep(20 * time.Second)

	assert.Equal(t, 200, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 100, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 100, pool.allTxsForLook.getRemoteCountByCategory(Category1))

}
