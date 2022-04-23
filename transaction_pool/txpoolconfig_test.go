package transactionpool

import (
	"encoding/json"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/TopiaNetwork/topia/codec"
)

func Test_transactionPool_SaveConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servant := NewMockTransactionPoolServant(ctrl)
	log := TpiaLog
	pool := SetNewTransactionPool(NodeID, Ctx, TestTxPoolConfig, 1, log, codec.CodecType(1))
	pool.query = servant

	//pool.config.PathRemote[Category1] = "newremote.json"

	if err := pool.SaveConfig(); err != nil {
		t.Error("want", nil, "got", err)
	}

	data, err := ioutil.ReadFile(pool.config.PathConfig)
	if err != nil {
		t.Error("want", nil, "got", err)
	}
	var conf TransactionPoolConfig
	config := &conf
	err = json.Unmarshal(data, &config)
	want := pool.config
	got := *config
	if !reflect.DeepEqual(want, got) {
		t.Error("want", want, "got", got)
	}
}
