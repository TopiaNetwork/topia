package test

import (
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/mocks"
	"github.com/TopiaNetwork/topia/api/web3"
	"github.com/TopiaNetwork/topia/api/web3/handlers"
	"github.com/TopiaNetwork/topia/api/web3/types"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/golang/mock/gomock"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetBlockByNumber(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	servantMock := mocks.NewMockAPIServant(ctrl)
	txInterfaceMock := mocks.NewMockTxInterface(ctrl)
	servantMock.
		EXPECT().
		GetBlockByHeight(gomock.Any()).
		DoAndReturn(func(height uint64) (*tpchaintypes.Block, error) {
			gasFees, _ := json.Marshal("0x68fc0")
			return &tpchaintypes.Block{
				Head: &tpchaintypes.BlockHead{
					ChainID:         []byte(ChainId),
					Version:         1,
					Height:          10,
					Epoch:           1,
					Round:           1,
					ParentBlockHash: GetHexByte("0x6ab6aff3346d3dc27b1fa87ece4fdb83dff42207d692179128ebd56b31229acc"),
					Launcher:        GetHexByte("0x9c71fbe2d28080b8afa88cea8a1e319de2c09d44"),
					Proposer:        GetHexByte("0x9c71fbe2d28080b8afa88cea8a1e319de2c09d44"),
					TxCount:         6,
					TxRoot:          GetHexByte("0x4ea6e8ed3f28744b8cb239b64150f024e3eb8f0ff4491acb14dc2e821a04d463"),
					TxResultRoot:    GetHexByte("0x4db6969931ba48e0e4073b7699fc32a7c1c6f738339b22f6a2f02279a814bb19"),
					StateRoot:       GetHexByte("0xb512f0bd7b64481857e0bbd1d7d2c90578d6f7ad2d20e82f415980a99b0e38ee"),
					GasFees:         gasFees,
					TimeStamp:       0x62656db0,
					Hash:            GetHexByte("0x983cd9063e6760ab4c7b1db96f3cbaa78588b7005516d3b4fdaad23fdde99499"),
				},
				Data: &tpchaintypes.BlockData{
					Version: 1,
					Txs: [][]byte{
						GetHexByte("0x9a5b716d6ff3d51f9196c579a49f724f0176ee7fc8fa618fd9aee3e10c002e18"),
						GetHexByte("0xe7435b7c408dd6a4589bf6fda96bf4808523443571ed9bc175ea72930a88808b"),
						GetHexByte("0x2e58685b7be596138707fd45c42854be9603822476be50be39164e71b0a49e6e"),
						GetHexByte("0xa99c75904cb96b7e557f3dde1c5e90f43f670d616b2fe57ef9e0a3b983f9a908"),
					},
				},
			}, nil
		}).
		Times(1)

	body := `{
		"jsonrpc":"2.0",
		"method":"eth_getBlockByNumber",
		"params":[
			"0xba647b",
			false
		],
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

	ethBlock := handlers.GetBlockResponseType{}
	json.Unmarshal(j.Result, &ethBlock)

	if ethBlock.ParentHash.Hex() != "0x6ab6aff3346d3dc27b1fa87ece4fdb83dff42207d692179128ebd56b31229acc" {
		t.Errorf("failed")
	}
}
