package test

import (
	"encoding/hex"
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/mocks"
	"github.com/TopiaNetwork/topia/api/web3"
	"github.com/TopiaNetwork/topia/api/web3/convert"
	"github.com/TopiaNetwork/topia/api/web3/types"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	"github.com/golang/mock/gomock"
	"io"
	"math/big"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGasPrice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	servantMock := mocks.NewMockAPIServant(ctrl)
	servantMock.
		EXPECT().
		GetLatestBlock().
		DoAndReturn(func() (*tpchaintypes.Block, error) {
			return &tpchaintypes.Block{
				Head: &tpchaintypes.BlockHead{
					ChainID:         []byte{9},
					Version:         1,
					Height:          10,
					Epoch:           1,
					Round:           1,
					ParentBlockHash: []byte("0x6ab6aff3346d3dc27b1fa87ece4fdb83dff42207d692179128ebd56b31229acc"),
					Launcher:        []byte("0x9c71fbe2d28080b8afa88cea8a1e319de2c09d44"),
					Proposer:        []byte("0x9c71fbe2d28080b8afa88cea8a1e319de2c09d44"),
					TxCount:         4,
					TxRoot:          []byte("0x4ea6e8ed3f28744b8cb239b64150f024e3eb8f0ff4491acb14dc2e821a04d463"),
					TxResultRoot:    []byte("0x4db6969931ba48e0e4073b7699fc32a7c1c6f738339b22f6a2f02279a814bb19"),
					StateRoot:       []byte("0xb512f0bd7b64481857e0bbd1d7d2c90578d6f7ad2d20e82f415980a99b0e38ee"),
					GasFees:         []byte("0x68fc0"),
					TimeStamp:       0x62656db0,
					Hash:            []byte("0x983cd9063e6760ab4c7b1db96f3cbaa78588b7005516d3b4fdaad23fdde99499"),
				},
				Data: &tpchaintypes.BlockData{
					Version: 1,
					Txs: [][]byte{
						[]byte("0x9a5b716d6ff3d51f9196c579a49f724f0176ee7fc8fa618fd9aee3e10c002e18"),
						[]byte("0xe7435b7c408dd6a4589bf6fda96bf4808523443571ed9bc175ea72930a88808b"),
						[]byte("0x2e58685b7be596138707fd45c42854be9603822476be50be39164e71b0a49e6e"),
						[]byte("0xa99c75904cb96b7e557f3dde1c5e90f43f670d616b2fe57ef9e0a3b983f9a908"),
					},
				},
			}, nil
		}).AnyTimes()
	servantMock.
		EXPECT().
		GetBlockByHeight(gomock.Any()).
		DoAndReturn(func(height uint64) (*tpchaintypes.Block, error) {
			return &tpchaintypes.Block{
				Head: &tpchaintypes.BlockHead{
					ChainID:         []byte{9},
					Version:         1,
					Height:          10,
					Epoch:           1,
					Round:           1,
					ParentBlockHash: []byte("0x6ab6aff3346d3dc27b1fa87ece4fdb83dff42207d692179128ebd56b31229acc"),
					Launcher:        []byte("0x9c71fbe2d28080b8afa88cea8a1e319de2c09d44"),
					Proposer:        []byte("0x9c71fbe2d28080b8afa88cea8a1e319de2c09d44"),
					TxCount:         6,
					TxRoot:          []byte("0x4ea6e8ed3f28744b8cb239b64150f024e3eb8f0ff4491acb14dc2e821a04d463"),
					TxResultRoot:    []byte("0x4db6969931ba48e0e4073b7699fc32a7c1c6f738339b22f6a2f02279a814bb19"),
					StateRoot:       []byte("0xb512f0bd7b64481857e0bbd1d7d2c90578d6f7ad2d20e82f415980a99b0e38ee"),
					GasFees:         []byte("0x68fc0"),
					TimeStamp:       0x62656db0,
					Hash:            []byte("0x983cd9063e6760ab4c7b1db96f3cbaa78588b7005516d3b4fdaad23fdde99499"),
				},
				Data: &tpchaintypes.BlockData{
					Version: 1,
					Txs: [][]byte{
						[]byte("0x9a5b716d6ff3d51f9196c579a49f724f0176ee7fc8fa618fd9aee3e10c002e18"),
						[]byte("0xe7435b7c408dd6a4589bf6fda96bf4808523443571ed9bc175ea72930a88808b"),
						[]byte("0x2e58685b7be596138707fd45c42854be9603822476be50be39164e71b0a49e6e"),
						[]byte("0xa99c75904cb96b7e557f3dde1c5e90f43f670d616b2fe57ef9e0a3b983f9a908"),
					},
				},
			}, nil
		}).AnyTimes()
	servantMock.
		EXPECT().
		GetTransactionByHash(gomock.Any()).
		DoAndReturn(func(txHashHex string) (*txbasic.Transaction, error) {
			//需要构建一个txbasic.Transaction实例才行
			//构建一个eth的，然后调用方法转换一下
			to := []byte("0x6fcd7b39e75619a68ab86a68b92d01134ef34ea3")
			toAddr := types.Address{}
			toAddr.SetBytes(to)

			rByte, _ := hex.DecodeString("fe4cd7376c6df62df3f8c60c6b82d90582c0b6922658d038665548eda98fc9f5")
			sByte, _ := hex.DecodeString("08725bdf6cb82a27e4798cbcb70fa220e1d7459353ef1a2c716f8101f4c9c601")

			ethLegacyTx := types.NewTx(&types.DynamicFeeTx{
				ChainID:   big.NewInt(9),
				Nonce:     0x0,
				To:        &toAddr,
				Value:     big.NewInt(0x38d7ea4c68000),
				Gas:       0x5208,
				GasFeeCap: big.NewInt(0x3ebd697803),
				GasTipCap: big.NewInt(0x8c347c90),
				V:         big.NewInt(0x1), //v不对
				R:         new(big.Int).SetBytes(rByte),
				S:         new(big.Int).SetBytes(sByte),
			})
			tx, _ := convert.ConvertEthTransactionToTopiaTransaction(ethLegacyTx)
			return tx, nil
		}).AnyTimes()

	//1发送请求
	//1.1构造请求
	body := `{
		"jsonrpc":"2.0",
		"method":"eth_gasPrice",
		"params":[],
		"id":73
	}`
	req := httptest.NewRequest("POST", "http://localhost:8080/home", strings.NewReader(body))
	res := httptest.NewRecorder()
	//1.2调用handler
	w3s := web3.InitWeb3Server(servantMock, nil)
	w3s.Web3Handler(res, req)

	result, _ := io.ReadAll(res.Result().Body)

	j := types.JsonrpcMessage{}
	json.Unmarshal(result, &j)

	gasPrice := new(big.Int)
	json.Unmarshal(j.Result, gasPrice)

	if gasPrice.Cmp(big.NewInt(269465778179)) != 0 {
		t.Errorf("failed")
	}
}
