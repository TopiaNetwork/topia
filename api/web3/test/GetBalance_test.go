package test

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/web3"
	"github.com/TopiaNetwork/topia/api/web3/types"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetBalance(t *testing.T) {
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
	w3s := web3.InitWeb3Server()
	w3s.Web3Handler(res, req)

	result, _ := io.ReadAll(res.Result().Body)

	j := types.JsonrpcMessage{}
	json.Unmarshal(result, &j)
	if j.Version != "2.0" {
		t.Errorf("failed")
	}
}
