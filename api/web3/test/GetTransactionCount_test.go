package test

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/mocks"
	"github.com/TopiaNetwork/topia/api/web3"
	"github.com/TopiaNetwork/topia/api/web3/types"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/golang/mock/gomock"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetTransactionCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	servantMock := mocks.NewMockAPIServant(ctrl)
	servantMock.
		EXPECT().
		GetTransactionCount(gomock.Any(), gomock.Any()).
		DoAndReturn(func(addr tpcrtypes.Address, height uint64) (uint64, error) {
			return 10, nil
		}).
		Times(1)
	//1发送请求
	//1.1构造请求
	body := `{
		"jsonrpc":"2.0",
		"method":"eth_getTransactionCount",
		"params":[
			"0x3F1B3C065aeE8cA34c47fa84aAC3024E95a2E6D9",
			"latest"
		],
		"id":1
	}`
	req := httptest.NewRequest("POST", "http://localhost:8080/home", strings.NewReader(body))
	res := httptest.NewRecorder()
	//1.2调用handler
	w3s := web3.InitWeb3Server(servantMock, nil)
	w3s.Web3Handler(res, req)

	result, _ := io.ReadAll(res.Result().Body)

	j := types.JsonrpcMessage{}
	json.Unmarshal(result, &j)

	count := new(uint64)
	json.Unmarshal(j.Result, count)
	if *count != 10 {
		t.Errorf("failed")
	}
}
