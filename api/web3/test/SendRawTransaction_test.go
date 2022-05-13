package test

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/mocks"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3"
	"github.com/TopiaNetwork/topia/api/web3/types"
	"github.com/golang/mock/gomock"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendRawTransaction(t *testing.T) {
	//1发送请求
	//1.1构造请求
	body := `{
		"jsonrpc":"2.0",
		"method":"eth_sendRawTransaction",
		"params":["0x02f873030b8459682f0085025c90ff50825208946fcd7b39e75619a68ab86a68b92d01134ef34ea388016345785d8a000080c001a00468068551701a4eb935052c207598a6c0d4810242a838cbf05848f15087be5ea03fdd0e86b4c8642d19bedb5aa12e036a6e38a286ab4bb9b250e045580784682f"],
		"id":1
	}`

	req := httptest.NewRequest("POST", "http://localhost:8080/", strings.NewReader(body))
	res := httptest.NewRecorder()

	//1.2调用handler
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	apiServant := servant.NewAPIServant()
	txInterfaceMock := mocks.NewMockTxInterface(ctrl)
	w3s := web3.InitWeb3Server(apiServant, txInterfaceMock)

	w3s.Web3Handler(res, req)
	result, _ := io.ReadAll(res.Result().Body)
	j := types.JsonrpcMessage{}
	json.Unmarshal(result, &j)
}
