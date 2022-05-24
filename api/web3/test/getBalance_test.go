package test

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/web3"
	"github.com/TopiaNetwork/topia/api/web3/eth/types"
	hexutil "github.com/TopiaNetwork/topia/api/web3/eth/types/hexutil"
	mocks2 "github.com/TopiaNetwork/topia/api/web3/mocks"
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

	servantMock := mocks2.NewMockAPIServant(ctrl)
	servantMock.
		EXPECT().
		GetBalance(gomock.Any(), gomock.Any()).
		DoAndReturn(func(symbol currency.TokenSymbol, addr tpcrtypes.Address) (*big.Int, error) {
			return big.NewInt(10), nil
		}).
		Times(1)
	body := `{
		"jsonrpc": "2.0",
		"method": "eth_getBalance",
		"params": ["0x9998E1896370fa57e42Cc98C835cEf7079dfaD7d", "latest"],
		"id": 2
	}`
	req := httptest.NewRequest("POST", "http://localhost:8080/home", strings.NewReader(body))
	res := httptest.NewRecorder()

	config := web3.Web3ServerConfiguration{
		HttpHost:  "",
		HttpPort:  "8080",
		HttpsHost: "",
		HttpsPost: "8443",
	}
	w3s := web3.InitWeb3Server(config, servantMock)
	w3s.ServeHttp(res, req)

	result, _ := io.ReadAll(res.Result().Body)

	j := types.JsonrpcMessage{}
	json.Unmarshal(result, &j)
	balance := new(hexutil.Big)
	json.Unmarshal(j.Result, &balance)
	if balance.ToInt().Cmp(big.NewInt(10)) != 0 {
		t.Errorf("failed")
	}
}
