package test

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/mocks"
	"github.com/TopiaNetwork/topia/api/web3"
	"github.com/TopiaNetwork/topia/api/web3/types"
	hexutil "github.com/TopiaNetwork/topia/api/web3/types/hexutil"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	"github.com/golang/mock/gomock"
	"io"
	"math/big"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEstimateGas(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	servantMock := mocks.NewMockAPIServant(controller)
	txInterfaceMock := mocks.NewMockTxInterface(controller)
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
		Host: "",
		Port: "8080",
	}
	w3s := web3.InitWeb3Server(config, servantMock, txInterfaceMock)
	w3s.ServeHttp(res, req)

	result, _ := io.ReadAll(res.Result().Body)

	j := types.JsonrpcMessage{}
	json.Unmarshal(result, &j)

	estimateGas := new(hexutil.Big)
	err := json.Unmarshal(j.Result, &estimateGas)
	if err != nil {
		t.Errorf(err.Error())
	}
	if estimateGas.ToInt().Cmp(big.NewInt(30000)) != 0 {
		t.Errorf("failed!")
	}
}
