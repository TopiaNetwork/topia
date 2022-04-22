package test

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/TopiaNetwork/topia/api/web3"
	"github.com/TopiaNetwork/topia/api/web3/types"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendRawTransaction(t *testing.T) {
	//1发送请求
	//1.1构造请求
	//from:0x3F1B3C065aeE8cA34c47fa84aAC3024E95a2E6D9
	body := `{
		"jsonrpc": "2.0",
		"method": "eth_sendRawTransaction",
		"params": ["0x02f872032b848c347c90853ebd697803825208946fcd7b39e75619a68ab86a68b92d01134ef34ea387038d7ea4c6800080c001a0aa4d754d7976cbdeb232a6193709b52bd0478d67e575f8da682188c7050b5eeda01c023aad5b77970d8ebde03276f2de8367ed6117c921f5d0c83b93e7c9c403ff"],
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
	var from []byte
	json.Unmarshal(from, j.Result)
	fmt.Println(hex.EncodeToString(from))
	if strings.EqualFold(hex.EncodeToString(from), "3F1B3C065aeE8cA34c47fa84aAC3024E95a2E6D9") {
		t.Errorf("failed")
	}
}
