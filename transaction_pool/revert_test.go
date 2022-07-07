package transactionpool

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	txpoolmock "github.com/TopiaNetwork/topia/transaction_pool/mock"
)

func Test_transactionPool_addTxsForBlocksRevert(t *testing.T) {
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

	txs := txLocals[:20]
	pool.addTxs(txs, true)
	pool.addTxs(txRemotes[:20], false)
	assert.Equal(t, 20, len(pool.GetRemoteTxs()))

	//NewBlock txs:txLocals[10:20]

	var blocks []*tpchaintypes.Block
	blocks = append(blocks, OldBlock)
	blocks = append(blocks, MidBlock)
	blocks = append(blocks, NewBlock)
	pool.chanBlocksRevert <- blocks
	time.Sleep(5 * time.Second)

	assert.Equal(t, 60, len(pool.GetRemoteTxs()))

}
