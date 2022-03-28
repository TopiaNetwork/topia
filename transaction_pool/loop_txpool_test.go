package transactionpool

import (
	"github.com/TopiaNetwork/topia/codec"
	"github.com/golang/mock/gomock"
	"testing"
)

func Test_transactionPool_loop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant
	pool.wg.Add(1)
	go pool.loop()
}
