package test

import (
	"encoding/json"
	"fmt"
	"github.com/TopiaNetwork/topia/api/web3"
	"github.com/TopiaNetwork/topia/api/web3/eth/types"
	hexutil "github.com/TopiaNetwork/topia/api/web3/eth/types/hexutil"
	mocks2 "github.com/TopiaNetwork/topia/api/web3/mocks"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	"github.com/golang/mock/gomock"
	"io"
	"math/big"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEstimateGas(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	servantMock := mocks2.NewMockAPIServant(controller)
	servantMock.
		EXPECT().
		EstimateGas(gomock.Any()).
		DoAndReturn(func(tx *txbasic.Transaction) (*big.Int, error) {
			return big.NewInt(30000), nil
		}).
		Times(1)

	body := `{
		"jsonrpc":"2.0",
		"method":"eth_estimateGas",
		"params":[{
			"from": "0xb60e8dd61c5d32be8058bb8eb970870f07233155",
			"to": "0xd46e8dd67c5d32be8058bb8eb970870f07244567",
			"gas": "",
			"gasPrice": "",
			"value": "",
			"data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"},"latest"],
		"id":1
	}`
	req := httptest.NewRequest("POST", "http://localhost:8080/", strings.NewReader(body))
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

	estimateGas := new(hexutil.Big)
	err = json.Unmarshal(j.Result, &estimateGas)
	if err != nil {
		t.Errorf(err.Error())
	}
	if estimateGas.ToInt().Cmp(big.NewInt(30000)) != 0 {
		t.Errorf("failed!")
	}
}

func TestGasPrice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	servantMock := mocks2.NewMockAPIServant(ctrl)
	servantMock.
		EXPECT().
		GetLatestBlock().
		DoAndReturn(func() (*tpchaintypes.Block, error) {
			return &Block, nil
		}).AnyTimes()
	servantMock.
		EXPECT().
		GetBlockByHeight(gomock.Any()).
		DoAndReturn(func(height uint64) (*tpchaintypes.Block, error) {
			return &Block, nil
		}).AnyTimes()
	servantMock.
		EXPECT().
		GetTransactionByHash(gomock.Any()).
		DoAndReturn(func(txHashHex string) (*txbasic.Transaction, error) {
			return &Transaction, nil
		}).AnyTimes()

	body := `{
		"jsonrpc":"2.0",
		"method":"eth_gasPrice",
		"params":[],
		"id":73
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

	gasPrice := new(hexutil.Big)
	err = json.Unmarshal(j.Result, gasPrice)
	if err != nil {
		return
	}
	fmt.Println(gasPrice.ToInt())
	if gasPrice.ToInt().Cmp(big.NewInt(269465778179)) != 0 {
		t.Errorf("failed")
	}
}
