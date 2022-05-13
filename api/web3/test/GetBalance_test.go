package test

import (
	"encoding/json"
	"errors"
	"github.com/TopiaNetwork/topia/api/mocks"
	"github.com/TopiaNetwork/topia/api/web3"
	"github.com/TopiaNetwork/topia/api/web3/types"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/currency"
	"github.com/golang/mock/gomock"
	"io"
	"math/big"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	servantMock := mocks.NewMockAPIServant(ctrl)
	servantMock.
		EXPECT().
		GetBalance(gomock.Any(), gomock.Any()).
		DoAndReturn(func(symbol currency.TokenSymbol, addr tpcrtypes.Address) (*big.Int, error) {
			//return big.NewInt(10), nil
			return nil, errors.New("getBalance error!")
		}).
		Times(1)
	//1发送请求
	//1.1构造请求
	body := `{
		"jsonrpc": "2.0",
		"method": "eth_getBalance",
		"params": ["0x9998E1896370fa57e42Cc98C835cEf7079dfaD7d", "latest"],
		"id": 2
	}`
	req := httptest.NewRequest("POST", "http://localhost:8080/home", strings.NewReader(body))
	res := httptest.NewRecorder()
	//1.2调用handler
	w3s := web3.InitWeb3Server(servantMock, nil)
	w3s.Web3Handler(res, req)

	result, _ := io.ReadAll(res.Result().Body)

	j := types.JsonrpcMessage{}
	json.Unmarshal(result, &j)
	balance := big.Int{}
	json.Unmarshal(j.Result, &balance)
	if balance.Cmp(big.NewInt(10)) != 0 {
		t.Errorf("failed")
	}
}
