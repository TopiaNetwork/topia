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

func TestGetCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	servantMock := mocks.NewMockAPIServant(ctrl)
	servantMock.
		EXPECT().
		GetContractCode(gomock.Any(), gomock.Any()).
		DoAndReturn(func(addr tpcrtypes.Address, height uint64) ([]byte, error) {
			return []byte("0x608060405260043610601f5760003560e01c80632e1a7d4d14602a576025565b36602557005b600080fd5b348015603557600080fd5b50605f60048036036020811015604a57600080fd5b81019080803590602001909291905050506061565b005b67016345785d8a0000811115607557600080fd5b3373ffffffffffffffffffffffffffffffffffffffff166108fc829081150290604051600060405180830381858888f1935050505015801560ba573d6000803e3d6000fd5b505056fea26469706673582212206172611ad1a5f939aa5c8fb05fccb7a60b2099c3473ad8194417684f5e56257364736f6c63430006040033"), nil
		}).
		Times(1)
	//1发送请求
	//1.1构造请求
	body := `{
		"jsonrpc":"2.0",
		"method":"eth_getCode",
		"params":[
			"0xa9a1984662195ecfcbc85e208b17d78010838eb6",
			"0xb7c95f"
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

	contractCode := new(string)
	json.Unmarshal(j.Result, &contractCode)
	if *contractCode == "0x" {
		t.Errorf("failed")
	}
}
