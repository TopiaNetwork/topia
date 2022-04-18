package transactionpool

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
)

func Test_transactionPool_Reset(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	servant.EXPECT().GetBlock(gomock.Eq(types.BlockHash(OldBlockHead.Hash)), OldBlockHead.Height).
		Return(OldBlock).AnyTimes()
	servant.EXPECT().GetBlock(gomock.Eq(types.BlockHash(NewBlockHead.Hash)), NewBlockHead.Height).
		Return(NewBlock).AnyTimes()
	servant.EXPECT().StateAt(gomock.Any()).Return(State, nil).AnyTimes()
	log := TpiaLog
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queues.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, len(pool.pendings.getAddrTxListOfCategory(Category1)))
	assert.Equal(t, 0, pool.allTxsForLook.getLocalCountByCategory(Category1))
	assert.Equal(t, 0, pool.allTxsForLook.all[Category1].RemoteCount())
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.locals))
	assert.Equal(t, 0, len(pool.sortedLists.Pricedlist[Category1].all.remotes))
	if err := pool.Reset(OldBlockHead, NewBlockHead); err != nil {
		t.Error("want", nil, "got", err)
	}

}
