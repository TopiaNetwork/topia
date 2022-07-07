package transactionpool

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txpoolmock "github.com/TopiaNetwork/topia/transaction_pool/mock"
)

func Test_transactionPool_loop_chanRemoveTxHashes(t *testing.T) {
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

	var hashes1, hashes2 []txbasic.TxID
	var hash txbasic.TxID

	for _, tx := range txLocals[:10] {
		hash, _ = tx.TxID()
		hashes1 = append(hashes1, hash)
		pool.AddTx(tx, false)
	}

	for _, tx := range txLocals[20:40] {
		hash, _ = tx.TxID()
		hashes2 = append(hashes2, hash)
		pool.AddTx(tx, false)
	}
	assert.Equal(t, 30, len(pool.GetRemoteTxs()))
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		pool.chanRmTxs <- hashes1
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		pool.chanRmTxs <- hashes2
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(5 * time.Second)
		pool.ctx.Done()
	}()

	wg.Wait()
	assert.Equal(t, 0, len(pool.GetRemoteTxs()))
	pool.TruncateTxPool()

}

func Test_transactionPool_loop_saveAllIfShutDown(t *testing.T) {
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

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)

	pool.SysShutDown()
	time.Sleep(5 * time.Second)
	fmt.Println("done")

}

func Test_transactionPool_loopDropTxsIfBlockAdded(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	log := TpiaLog

	stateService := txpoolmock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := txpoolmock.NewMockBlockService(ctrl)
	blockService.EXPECT().GetBlockByHash(tpchaintypes.BlockHash(OldBlock.Head.Hash)).AnyTimes().Return(OldBlock, nil)
	blockService.EXPECT().GetBlockByHash(tpchaintypes.BlockHash(MidBlock.Head.Hash)).AnyTimes().Return(MidBlock, nil)
	blockService.EXPECT().GetBlockByHash(tpchaintypes.BlockHash(NewBlock.Head.Hash)).AnyTimes().Return(NewBlock, nil)
	blockService.EXPECT().GetBlockByNumber(tpchaintypes.BlockNum(10)).AnyTimes().Return(OldBlock, nil)
	blockService.EXPECT().GetBlockByNumber(tpchaintypes.BlockNum(11)).AnyTimes().Return(MidBlock, nil)
	blockService.EXPECT().GetBlockByNumber(tpchaintypes.BlockNum(12)).AnyTimes().Return(NewBlock, nil)
	network := txpoolmock.NewMockNetwork(ctrl)

	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	pool.TruncateTxPool()

	txs := txLocals[10:40]

	pool.addTxs(txs, true)
	assert.Equal(t, 30, len(pool.GetLocalTxs()))
	assert.Equal(t, 0, len(pool.GetRemoteTxs()))
	//OldBlock txs:txLocals[10:20]，remotes[10:20]
	pool.chanBlockAdded <- OldBlock
	time.Sleep(10 * time.Second)
	assert.Equal(t, 20, len(pool.GetLocalTxs()))
	assert.Equal(t, 0, len(pool.GetRemoteTxs()))

}

func Test_transactionPool_removeTxForUptoLifeTime(t *testing.T) {
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

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)

	//**********for test
	//*change default lifTime to 4 second,and change TxExpiredTime to 3 second **
	//**********for test
	pool.config.LifetimeForTx = 4 * time.Second
	assert.Equal(t, int64(4), pool.Count())
	time.Sleep(13 * time.Second)
	locals := pool.GetLocalTxs()
	if len(locals) > 0 {
		for _, tx := range locals {
			fmt.Println(tx.Head.FromAddr, tx.Head.Nonce)
		}
	}
	remotes := pool.GetRemoteTxs()
	if len(remotes) > 0 {
		for _, tx := range remotes {
			fmt.Println(tx.Head.FromAddr, tx.Head.Nonce)
		}
	}
	assert.Equal(t, int64(0), pool.Count())
	fmt.Println("test down")

}

func Test_transactionPool_regularSaveLocalTxs(t *testing.T) {
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

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)
	//**********for test
	//*change default ReStoredDur to 4 second **
	//**********for test
	time.Sleep(10 * time.Second)
	pool.ClearLocalFile(pool.config.PathTxsStorage)
	//you can see this log:
	// loadTxsData file removed  module=TransactionPool
}

func Test_transactionPool_regularRepublic(t *testing.T) {
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

	pool.AddTx(Tx1, true)
	pool.AddTx(Tx2, true)
	pool.AddTx(TxR1, false)
	pool.AddTx(TxR2, false)

	wrapT1, ok := pool.allWrappedTxs.Get(Key1)
	assert.Equal(t, true, ok)
	assert.Equal(t, false, wrapT1.IsRepublished)
	wrapT2, ok := pool.allWrappedTxs.Get(KeyR2)
	assert.Equal(t, true, ok)
	assert.Equal(t, false, wrapT2.IsRepublished)

	//**********for test
	//*change RepublishTxInterval to 3000 * time.Millisecond,
	// change default TxTTLTimeOfRepublish to 4 *time.Second
	//**********for test
	time.Sleep(13 * time.Second)

	wrapT1, ok = pool.allWrappedTxs.Get(Key1)
	assert.Equal(t, true, ok)
	assert.Equal(t, true, wrapT1.IsRepublished)
	wrapT2, ok = pool.allWrappedTxs.Get(Key2)
	assert.Equal(t, true, ok)
	assert.Equal(t, true, wrapT2.IsRepublished)
	fmt.Println("test down")

}

func Test_transactionPool_loop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	log := TpiaLog

	stateService := txpoolmock.NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := txpoolmock.NewMockBlockService(ctrl)
	blockService.EXPECT().GetBlockByHash(tpchaintypes.BlockHash(OldBlock.Head.Hash)).AnyTimes().Return(OldBlock, nil)
	blockService.EXPECT().GetBlockByHash(tpchaintypes.BlockHash(MidBlock.Head.Hash)).AnyTimes().Return(MidBlock, nil)
	blockService.EXPECT().GetBlockByHash(tpchaintypes.BlockHash(NewBlock.Head.Hash)).AnyTimes().Return(NewBlock, nil)
	blockService.EXPECT().GetBlockByNumber(OldBlock.BlockNum()).AnyTimes().Return(OldBlock, nil)
	blockService.EXPECT().GetBlockByNumber(MidBlock.BlockNum()).AnyTimes().Return(MidBlock, nil)
	blockService.EXPECT().GetBlockByNumber(NewBlock.BlockNum()).AnyTimes().Return(NewBlock, nil)

	network := txpoolmock.NewMockNetwork(ctrl)

	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)
	pool.TruncateTxPool()

	keyLocals = make([]txbasic.TxID, 0)
	keyRemotes = make([]txbasic.TxID, 0)
	txLocals = make([]*txbasic.Transaction, 0)
	txRemotes = make([]*txbasic.Transaction, 0)
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

		txremote = setTxRemote(nonce, gasprice, gaslimit)
		txremote.Head.TimeStamp = starttime + uint64(i)
		if i > 1 {
			txremote.Head.FromAddr = append(txremote.Head.FromAddr, byte(i))
		}
		keyremote, _ = txremote.TxID()
		keyRemotes = append(keyRemotes, keyremote)
		txRemotes = append(txRemotes, txremote)

		pool.AddTx(txlocal, true)
		pool.AddTx(txremote, false)
	}
	assert.Equal(t, int64(200), pool.Count())
	go func() {
		pool.chanRmTxs <- keyLocals
	}()
	go func() {
		pool.chanRmTxs <- keyRemotes
	}()

	pool.chanBlockAdded <- NewBlock
	pool.SysShutDown()

	time.Sleep(20 * time.Second)
	assert.Equal(t, int64(0), pool.Count())

}
