package transactionpool

import (
	"errors"
	"github.com/TopiaNetwork/topia/codec"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/transaction/basic"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_transactionPool_loop_chanRemoveTxHashs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	servant.EXPECT().CurrentBlock().Return(NewBlock).AnyTimes()
	log := TpiaLog
	pool := SetNewTransactionPool(Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant

	assert.Equal(t, 0, len(pool.queues[Category1].accTxs))
	assert.Equal(t, 0, len(pool.pendings[Category1].accTxs))
	assert.Equal(t, 0, pool.allTxsForLook[Category1].LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook[Category1].RemoteCount())

	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	assert.Equal(t, 2, len(pool.queues[Category1].accTxs))
	assert.Equal(t, 0, len(pool.pendings[Category1].accTxs))
	assert.Equal(t, 2, pool.allTxsForLook[Category1].LocalCount())
	assert.Equal(t, 2, pool.allTxsForLook[Category1].RemoteCount())

	assert.Equal(t, 2, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 2, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	hashcatmap := make(map[string]basic.TransactionCategory)
	hashcatmap[Key1] = basic.TransactionCategory_Topia_Universal
	hashcatmap[Key2] = basic.TransactionCategory_Topia_Universal
	hashcatmap[KeyR1] = basic.TransactionCategory_Topia_Universal
	hashcatmap[KeyR2] = basic.TransactionCategory_Topia_Universal
	var hashcatmap1, hashcatmap2 map[string]basic.TransactionCategory
	var hash string
	for _, tx := range txLocals[1:10] {
		hash, _ = tx.HashHex()
		hashcatmap1[hash] = basic.TransactionCategory_Topia_Universal
		pool.AddTx(tx, false)
	}
	for _, tx := range txLocals[20:40] {
		hash, _ = tx.HashHex()
		hashcatmap2[hash] = basic.TransactionCategory_Topia_Universal
		pool.AddTx(tx, false)
	}
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

	go func() {
		pool.chanRmTxs <- hashcatmap
	}()
	go func() {
		pool.chanRmTxs <- hashcatmap1
	}()
	go func() {
		pool.chanRmTxs <- hashcatmap2
	}()

	go func() {
		time.Sleep(5 * time.Second)
		assert.Equal(t, 0, len(pool.queues[Category1].accTxs))
		assert.Equal(t, 0, len(pool.pendings[Category1].accTxs))
		assert.Equal(t, 0, pool.allTxsForLook[Category1].LocalCount())
		assert.Equal(t, 0, pool.allTxsForLook[Category1].RemoteCount())

		assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
		assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	}()
	pool.wg.Wait()

}

func Test_transactionPool_loop_saveAllIfShutDown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	servant.EXPECT().CurrentBlock().Return(NewBlock).AnyTimes()

	log := TpiaLog
	pool := SetNewTransactionPool(Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant

	newnetwork := NewMockNetwork(ctrl)
	newnetwork.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	pool.network = newnetwork
	defer pool.wg.Wait()
	assert.Equal(t, 0, len(pool.queues[Category1].accTxs))
	assert.Equal(t, 0, len(pool.pendings[Category1].accTxs))
	assert.Equal(t, 0, pool.allTxsForLook[Category1].LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook[Category1].RemoteCount())

	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	assert.Equal(t, 2, len(pool.queues[Category1].accTxs))
	assert.Equal(t, 0, len(pool.pendings[Category1].accTxs))
	assert.Equal(t, 2, pool.allTxsForLook[Category1].LocalCount())
	assert.Equal(t, 2, pool.allTxsForLook[Category1].RemoteCount())

	assert.Equal(t, 2, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 2, len(pool.sortedLists.Pricedlist[Category1].all.remotes))

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
	pool := SetNewTransactionPool(Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant

	newnetwork := NewMockNetwork(ctrl)
	newnetwork.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	pool.network = newnetwork
	defer pool.wg.Wait()
	assert.Equal(t, 0, len(pool.queues[Category1].accTxs))
	assert.Equal(t, 0, len(pool.pendings[Category1].accTxs))
	assert.Equal(t, 0, pool.allTxsForLook[Category1].LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook[Category1].RemoteCount())

	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	assert.Equal(t, 2, len(pool.queues[Category1].accTxs))
	assert.Equal(t, 0, len(pool.pendings[Category1].accTxs))
	assert.Equal(t, 2, pool.allTxsForLook[Category1].LocalCount())
	assert.Equal(t, 2, pool.allTxsForLook[Category1].RemoteCount())

	assert.Equal(t, 2, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 2, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
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
	newheadevent := &ChainHeadEvent{NewBlock}
	pool.chanChainHead <- *newheadevent
}

func Test_transactionPool_loop_reportTicks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	servant.EXPECT().CurrentBlock().Return(NewBlock).AnyTimes()

	log := TpiaLog
	pool := SetNewTransactionPool(Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant

	newnetwork := NewMockNetwork(ctrl)
	newnetwork.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	pool.network = newnetwork
	defer pool.wg.Wait()
	assert.Equal(t, 0, len(pool.queues[Category1].accTxs))
	assert.Equal(t, 0, len(pool.pendings[Category1].accTxs))
	assert.Equal(t, 0, pool.allTxsForLook[Category1].LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook[Category1].RemoteCount())

	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	assert.Equal(t, 2, len(pool.queues[Category1].accTxs))
	assert.Equal(t, 0, len(pool.pendings[Category1].accTxs))
	assert.Equal(t, 2, pool.allTxsForLook[Category1].LocalCount())
	assert.Equal(t, 2, pool.allTxsForLook[Category1].RemoteCount())

	assert.Equal(t, 2, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 2, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
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
	pool := SetNewTransactionPool(Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant

	newnetwork := NewMockNetwork(ctrl)
	newnetwork.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	pool.network = newnetwork
	defer pool.wg.Wait()
	assert.Equal(t, 0, len(pool.queues[Category1].accTxs))
	assert.Equal(t, 0, len(pool.pendings[Category1].accTxs))
	assert.Equal(t, 0, pool.allTxsForLook[Category1].LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook[Category1].RemoteCount())

	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	assert.Equal(t, 2, len(pool.queues[Category1].accTxs))
	assert.Equal(t, 0, len(pool.pendings[Category1].accTxs))
	assert.Equal(t, 2, pool.allTxsForLook[Category1].LocalCount())
	assert.Equal(t, 2, pool.allTxsForLook[Category1].RemoteCount())

	assert.Equal(t, 2, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 2, len(pool.sortedLists.Pricedlist[Category1].all.remotes))

	//pool.wg.Add(1)
	//go pool.chanRemoveTxHashs()
	//pool.wg.Add(1)
	//go pool.saveAllIfShutDown()
	//pool.wg.Add(1)
	//go pool.resetIfNewHead()
	//pool.wg.Add(1)
	//go pool.reportTicks()

	waitChannel := make(chan struct{})

	//***********change lifTime to trigger**************
	pool.wg.Add(1)
	go pool.removeTxForUptoLifeTime()
	//pool.wg.Add(1)
	//go pool.regularSaveLocalTxs()
	//pool.wg.Add(1)
	//go pool.regularRepublic()
	<-waitChannel
	time.Sleep(10 * time.Second)
	assert.Equal(t, 0, len(pool.queues[Category1].accTxs))
	assert.Equal(t, 0, len(pool.pendings[Category1].accTxs))
	assert.Equal(t, 0, pool.allTxsForLook[Category1].LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook[Category1].RemoteCount())

	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
}

func Test_transactionPool_regularSaveLocalTxs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	servant.EXPECT().CurrentBlock().Return(NewBlock).AnyTimes()

	log := TpiaLog
	pool := SetNewTransactionPool(Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant

	newnetwork := NewMockNetwork(ctrl)
	newnetwork.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	pool.network = newnetwork
	defer pool.wg.Wait()
	assert.Equal(t, 0, len(pool.queues[Category1].accTxs))
	assert.Equal(t, 0, len(pool.pendings[Category1].accTxs))
	assert.Equal(t, 0, pool.allTxsForLook[Category1].LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook[Category1].RemoteCount())

	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	assert.Equal(t, 2, len(pool.queues[Category1].accTxs))
	assert.Equal(t, 0, len(pool.pendings[Category1].accTxs))
	assert.Equal(t, 2, pool.allTxsForLook[Category1].LocalCount())
	assert.Equal(t, 2, pool.allTxsForLook[Category1].RemoteCount())

	assert.Equal(t, 2, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 2, len(pool.sortedLists.Pricedlist[Category1].all.remotes))

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
	pool := SetNewTransactionPool(Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant

	newnetwork := NewMockNetwork(ctrl)
	newnetwork.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	pool.network = newnetwork
	defer pool.wg.Wait()
	assert.Equal(t, 0, len(pool.queues[Category1].accTxs))
	assert.Equal(t, 0, len(pool.pendings[Category1].accTxs))
	assert.Equal(t, 0, pool.allTxsForLook[Category1].LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook[Category1].RemoteCount())

	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	assert.Equal(t, 2, len(pool.queues[Category1].accTxs))
	assert.Equal(t, 0, len(pool.pendings[Category1].accTxs))
	assert.Equal(t, 2, pool.allTxsForLook[Category1].LocalCount())
	assert.Equal(t, 2, pool.allTxsForLook[Category1].RemoteCount())

	assert.Equal(t, 2, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 2, len(pool.sortedLists.Pricedlist[Category1].all.remotes))

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

func Test_transactionPool_loop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	servant.EXPECT().CurrentBlock().Return(NewBlock).AnyTimes()
	log := TpiaLog
	pool := SetNewTransactionPool(Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	newnetwork := NewMockNetwork(ctrl)
	newnetwork.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	pool.network = newnetwork
	defer pool.wg.Wait()

	keyCatLocals = make(map[string]basic.TransactionCategory, 0)
	keyCatRemotes = make(map[string]basic.TransactionCategory, 0)
	txLocals = make([]*basic.Transaction, 0)
	txRemotes = make([]*basic.Transaction, 0)
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
		keylocal, _ = txlocal.HashHex()
		keyCatLocals[keylocal] = basic.TransactionCategory_Topia_Universal
		txLocals = append(txLocals, txlocal)
		fromlocal = tpcrtypes.Address(txlocal.Head.FromAddr)

		txremote = setTxRemote(nonce, gasprice, gaslimit)
		txremote.Head.TimeStamp = starttime + uint64(i)
		if i > 1 {
			txremote.Head.FromAddr = append(txremote.Head.FromAddr, byte(i))
		}
		keyremote, _ = txremote.HashHex()
		keyCatLocals[keyremote] = basic.TransactionCategory_Topia_Universal
		txRemotes = append(txRemotes, txremote)
		fromremote = tpcrtypes.Address(txremote.Head.FromAddr)

		//	fmt.Printf("i == %d:start,\n", i)
		_ = pool.AddTx(txlocal, true)
		assert.Equal(t, 1, pool.queues[Category1].accTxs[fromlocal].txs.Len())
		//	fmt.Printf("i == %d:1,addLocaltx%v:\n", i, err)
		_ = pool.AddTx(txremote, false)
		assert.Equal(t, 1, pool.queues[Category1].accTxs[fromremote].txs.Len())
	}

	assert.Equal(t, 200, len(pool.queues[Category1].accTxs))
	assert.Equal(t, 0, len(pool.pendings[Category1].accTxs))
	assert.Equal(t, 100, pool.allTxsForLook[Category1].LocalCount())
	assert.Equal(t, 100, pool.allTxsForLook[Category1].RemoteCount())

	assert.Equal(t, 100, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 100, len(pool.sortedLists.Pricedlist[Category1].all.remotes))

	pool.loop()

	go func() {
		pool.chanRmTxs <- keyCatLocals
	}()
	go func() {
		pool.chanRmTxs <- keyCatRemotes
	}()

	waitChannel := make(chan struct{})

	newheadevent := &ChainHeadEvent{NewBlock}
	pool.chanChainHead <- *newheadevent
	pool.chanSysShutDown <- errors.New("shut down")

	<-waitChannel
	time.Sleep(20 * time.Second)

	assert.Equal(t, 200, len(pool.queues[Category1].accTxs))
	assert.Equal(t, 0, len(pool.pendings[Category1].accTxs))
	assert.Equal(t, 100, pool.allTxsForLook[Category1].LocalCount())
	assert.Equal(t, 100, pool.allTxsForLook[Category1].RemoteCount())

	assert.Equal(t, 100, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 100, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
}
