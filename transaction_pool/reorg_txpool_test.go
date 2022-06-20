package transactionpool

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	txpoolmock "github.com/TopiaNetwork/topia/transaction_pool/mock"

)

func Test_transactionPool_Reset(t *testing.T) {
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
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))
	txs := txLocals[20:50]
	pool.addTxs(txs, true)
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 30, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 30, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))

	if err := pool.reset(OldBlockHead, NewBlockHead); err != nil {
		t.Error("want", nil, "got", err)
	}
	var addrsNeedTurn []tpcrtypes.Address
	addrsNeedTurn = pool.queues.getAllAddress()
	pool.turnAddrTxsToPending(addrsNeedTurn)
	pool.demoteUnexecutables(Category1) //demote transactions
	pool.truncatePendingByCategory(Category1)
	pool.truncateQueueByCategory(Category1)
	assert.Equal(t, 30, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 30, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.getRemoteCountByCategory(Category1))

}
