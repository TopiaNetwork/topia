package transactionpool

import (
	"encoding/hex"
	"github.com/TopiaNetwork/topia/codec"
	"github.com/TopiaNetwork/topia/common/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_transactionPool_Reset(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	servant.EXPECT().GetBlock(gomock.Eq(types.BlockHash(hex.EncodeToString(OldBlockHead.TxHashRoot))), OldBlockHead.Height).
		Return(OldBlock).AnyTimes()
	servant.EXPECT().GetBlock(gomock.Eq(types.BlockHash(hex.EncodeToString(NewBlockHead.TxHashRoot))), NewBlockHead.Height).
		Return(NewBlock).AnyTimes()
	servant.EXPECT().StateAt(gomock.Any()).Return(State, nil).AnyTimes()
	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	assert.Equal(t, 0, len(pool.queue.accTxs))
	assert.Equal(t, 0, len(pool.pending.accTxs))
	assert.Equal(t, 0, pool.allTxsForLook.LocalCount())
	assert.Equal(t, 0, pool.allTxsForLook.RemoteCount())
	assert.Equal(t, 0, len(pool.sortedByPriced.all.locals))
	assert.Equal(t, 0, len(pool.sortedByPriced.all.remotes))
	if err := pool.Reset(OldBlockHead, NewBlockHead); err != nil {
		t.Error("want", nil, "got", err)
	}

}
