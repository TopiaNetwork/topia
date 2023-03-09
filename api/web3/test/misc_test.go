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
	"time"
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
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		HttpHost:     "",
		HttpPort:     "8080",
		HttpsHost:    "",
		HttpsPost:    "8443",
	}
	w3s := web3.InitWeb3Server(config, servantMock)
	w3s.ServeHttp(res, req)

	result, _ := io.ReadAll(res.Result().Body)

	j := types.JsonrpcMessage{}
	err := json.Unmarshal(result, &j)
	if err != nil {
		return
	}
	balance := new(hexutil.Big)
	err = json.Unmarshal(j.Result, &balance)
	if err != nil {
		return
	}
	if balance.ToInt().Cmp(big.NewInt(10)) != 0 {
		t.Errorf("failed")
	}
}

func TestGetCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	servantMock := mocks2.NewMockAPIServant(ctrl)
	servantMock.
		EXPECT().
		GetContractCode(gomock.Any(), gomock.Any()).
		DoAndReturn(func(addr tpcrtypes.Address, height uint64) ([]byte, error) {
			return []byte("0x608060405260043610601f5760003560e01c80632e1a7d4d14602a576025565b36602557005b600080fd5b348015603557600080fd5b50605f60048036036020811015604a57600080fd5b81019080803590602001909291905050506061565b005b67016345785d8a0000811115607557600080fd5b3373ffffffffffffffffffffffffffffffffffffffff166108fc829081150290604051600060405180830381858888f1935050505015801560ba573d6000803e3d6000fd5b505056fea26469706673582212206172611ad1a5f939aa5c8fb05fccb7a60b2099c3473ad8194417684f5e56257364736f6c63430006040033"), nil
		}).
		Times(1)

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
	config := web3.Web3ServerConfiguration{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		HttpHost:     "",
		HttpPort:     "8080",
		HttpsHost:    "",
		HttpsPost:    "8443",
	}
	w3s := web3.InitWeb3Server(config, servantMock)
	w3s.ServeHttp(res, req)

	result, _ := io.ReadAll(res.Result().Body)

	j := types.JsonrpcMessage{}
	err := json.Unmarshal(result, &j)
	if err != nil {
		return
	}
	contractCode := new(string)
	err = json.Unmarshal(j.Result, &contractCode)
	if err != nil {
		return
	}
	if *contractCode != "0x30783630383036303430353236303034333631303630316635373630303033353630653031633830363332653161376434643134363032613537363032353536356233363630323535373030356236303030383066643562333438303135363033353537363030303830666435623530363035663630303438303336303336303230383131303135363034613537363030303830666435623831303139303830383033353930363032303031393039323931393035303530353036303631353635623030356236373031363334353738356438613030303038313131313536303735353736303030383066643562333337336666666666666666666666666666666666666666666666666666666666666666666666666666666631363631303866633832393038313135303239303630343035313630303036303430353138303833303338313835383838386631393335303530353035303135383031353630626135373364363030303830336533643630303066643562353035303536666561323634363937303636373335383232313232303631373236313161643161356639333961613563386662303566636362376136306232303939633334373361643831393434313736383466356535363235373336343733366636633633343330303036303430303333" {
		t.Errorf("failed")
	}
}
