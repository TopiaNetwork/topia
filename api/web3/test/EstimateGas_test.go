package test

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/mocks"
	"github.com/TopiaNetwork/topia/api/web3"
	"github.com/TopiaNetwork/topia/api/web3/types"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	"github.com/golang/mock/gomock"
	"io"
	"math/big"
	"net/http/httptest"
	"strings"
	"testing"
)

//TODO:需要其他handler的mock
//getTransactionByHasg
//getBlockByHash
func TestEstimateGas(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	//我应该构造的是servant的mock，即为servant里面的那些方法进行打桩
	servantMock := mocks.NewMockAPIServant(controller)
	//能不能只对目标代码进行更新？可以，使用DoAndReturn
	servantMock.
		EXPECT().
		EstimateGas(gomock.Any()).
		DoAndReturn(func(tx *txbasic.Transaction) (*big.Int, error) {
			return big.NewInt(30000), nil
		}).
		Times(1)

	//1发送请求
	//1.1构造请求
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
	//1.2调用handler
	w3s := web3.InitWeb3Server(servantMock, nil)
	w3s.Web3Handler(res, req)

	result, _ := io.ReadAll(res.Result().Body)

	j := types.JsonrpcMessage{}
	json.Unmarshal(result, &j)

	estimateGas := new(big.Int)
	json.Unmarshal(j.Result, &estimateGas)

	if estimateGas.Cmp(big.NewInt(10000)) != 0 {
		t.Errorf("failed!")
	}
}
