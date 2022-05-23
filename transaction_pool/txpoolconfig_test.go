package transactionpool

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/TopiaNetwork/topia/codec"
)

func Test_transactionPool_SetTxPoolConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	log := TpiaLog
	stateService := NewMockStateQueryService(ctrl)
	stateService.EXPECT().GetLatestBlock().AnyTimes().Return(OldBlock, nil)
	stateService.EXPECT().GetNonce(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	blockService := NewMockBlockService(ctrl)
	network := NewMockNetwork(ctrl)
	network.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1), stateService, blockService, network)

	newconf := pool.config
	newconf.TxPoolMaxSize = 123456789
	pool.SetTxPoolConfig(newconf)
	want := newconf.TxPoolMaxSize
	got := pool.config.TxPoolMaxSize
	if !assert.Equal(t, want, got) {
		t.Error("want", want, "got", got)
	}
}
