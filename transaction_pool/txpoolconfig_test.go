package transactionpool

import (
	"encoding/json"
	_interface "github.com/TopiaNetwork/topia/transaction_pool/interface"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/TopiaNetwork/topia/codec"
)

func Test_transactionPool_SaveConfig(t *testing.T) {
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

	//pool.config.PathRemote[Category1] = "newremote.json"

	if err := pool.SaveConfig(); err != nil {
		t.Error("want", nil, "got", err)
	}

	data, err := ioutil.ReadFile(pool.config.PathConfig)
	if err != nil {
		t.Error("want", nil, "got", err)
	}
	var conf _interface.TransactionPoolConfig
	config := &conf
	err = json.Unmarshal(data, &config)
	want := pool.config
	got := *config
	if !reflect.DeepEqual(want, got) {
		t.Error("want", want, "got", got)
	}
}

func Test_transactionPool_LoadConfig(t *testing.T) {
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

	if err := pool.SaveConfig(); err != nil {
		t.Error("want", nil, "got", err)
	}

	conf, _ := pool.LoadConfig()
	want := pool.config
	got := *conf
	if !assert.Equal(t, want, got) {
		t.Error("want", want, "got", got)
	}
}

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

	//pool.config.PathRemote[Category1] = "newremote.json"
	if err := pool.SaveConfig(); err != nil {
		t.Error("want", nil, "got", err)
	}

	conf, _ := pool.LoadConfig()
	pool.SetTxPoolConfig(*conf)
	want := *conf
	got := pool.config
	if !assert.Equal(t, want, got) {
		t.Error("want", *conf, "got", pool.config)
	}
}
