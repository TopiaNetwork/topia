package test

import (
	"encoding/hex"
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/web3"
	types2 "github.com/TopiaNetwork/topia/api/web3/eth/types"
	"github.com/TopiaNetwork/topia/api/web3/eth/types/eth_account"
	hexutil "github.com/TopiaNetwork/topia/api/web3/eth/types/hexutil"
	"github.com/TopiaNetwork/topia/api/web3/handlers"
	mocks2 "github.com/TopiaNetwork/topia/api/web3/mocks"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/golang/mock/gomock"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGetBlockByHash(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	servantMock := mocks2.NewMockAPIServant(ctrl)
	servantMock.
		EXPECT().
		GetBlockByHash(gomock.Any()).
		DoAndReturn(func(txHashHex string) (*tpchaintypes.Block, error) {
			gasFees, _ := json.Marshal("0x68fc0")
			hdChunk := &tpchaintypes.BlockHeadChunk{
				Version:      tpchaintypes.BLOCK_VER,
				DomainID:     []byte("topiaexe"),
				Launcher:     []byte("0x9c71fbe2d28080b8afa88cea8a1e319de2c09d44"),
				TxCount:      6,
				TxRoot:       []byte("0x4ea6e8ed3f28744b8cb239b64150f024e3eb8f0ff4491acb14dc2e821a04d463"),
				TxResultRoot: []byte("0x4db6969931ba48e0e4073b7699fc32a7c1c6f738339b22f6a2f02279a814bb19"),
			}
			hdChunkBytes, _ := hdChunk.Marshal()

			dataChunk := &tpchaintypes.BlockDataChunk{
				Version:  tpchaintypes.BLOCK_VER,
				RefIndex: 0,
				Txs: [][]byte{
					[]byte("0x9a5b716d6ff3d51f9196c579a49f724f0176ee7fc8fa618fd9aee3e10c002e18"),
					[]byte("0xe7435b7c408dd6a4589bf6fda96bf4808523443571ed9bc175ea72930a88808b"),
					[]byte("0x2e58685b7be596138707fd45c42854be9603822476be50be39164e71b0a49e6e"),
					[]byte("0xa99c75904cb96b7e557f3dde1c5e90f43f670d616b2fe57ef9e0a3b983f9a908"),
				},
			}
			dataChunkBytes, _ := dataChunk.Marshal()
			return &tpchaintypes.Block{
				Head: &tpchaintypes.BlockHead{
					ChainID:         []byte(ChainId),
					Version:         1,
					Height:          10,
					Epoch:           1,
					Round:           1,
					ParentBlockHash: GetHexByte("0x6ab6aff3346d3dc27b1fa87ece4fdb83dff42207d692179128ebd56b31229acc"),
					Proposer:        GetHexByte("0x9c71fbe2d28080b8afa88cea8a1e319de2c09d44"),
					StateRoot:       GetHexByte("0xb512f0bd7b64481857e0bbd1d7d2c90578d6f7ad2d20e82f415980a99b0e38ee"),
					ChunkCount:      1,
					HeadChunks:      [][]byte{hdChunkBytes},
					GasFees:         gasFees,
					TimeStamp:       0x62656db0,
					Hash:            GetHexByte("0x983cd9063e6760ab4c7b1db96f3cbaa78588b7005516d3b4fdaad23fdde99499"),
				},
				Data: &tpchaintypes.BlockData{
					Version:    1,
					DataChunks: [][]byte{dataChunkBytes},
				},
			}, nil
		}).
		Times(1)

	body := `{
		"jsonrpc":"2.0",
		"method":"eth_getBlockByHash",
		"params":[
			"0xfce2a84106c71bac8e14ee3aa967a37e93658c09a36a956ecaf415546c89c9a2",
			false
		],
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

	j := types2.JsonrpcMessage{}
	err := json.Unmarshal(result, &j)
	if err != nil {
		return
	}

	ethBlock := handlers.GetBlockResponseType{}
	err = json.Unmarshal(j.Result, &ethBlock)
	if err != nil {
		return
	}

	if ethBlock.ParentHash.Hex() != "0x6ab6aff3346d3dc27b1fa87ece4fdb83dff42207d692179128ebd56b31229acc" {
		t.Errorf("failed")
	}
}

func TestGetBlockByNumber(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	servantMock := mocks2.NewMockAPIServant(ctrl)
	servantMock.
		EXPECT().
		GetBlockByHeight(gomock.Any()).
		DoAndReturn(func(height uint64) (*tpchaintypes.Block, error) {
			gasFees, _ := json.Marshal("0x68fc0")
			hdChunk := &tpchaintypes.BlockHeadChunk{
				Version:      tpchaintypes.BLOCK_VER,
				DomainID:     []byte("topiaexe"),
				Launcher:     GetHexByte("0x9c71fbe2d28080b8afa88cea8a1e319de2c09d44"),
				TxCount:      6,
				TxRoot:       GetHexByte("0x4ea6e8ed3f28744b8cb239b64150f024e3eb8f0ff4491acb14dc2e821a04d463"),
				TxResultRoot: GetHexByte("0x4db6969931ba48e0e4073b7699fc32a7c1c6f738339b22f6a2f02279a814bb19"),
			}
			hdChunkBytes, _ := hdChunk.Marshal()

			dataChunk := &tpchaintypes.BlockDataChunk{
				Version:  tpchaintypes.BLOCK_VER,
				RefIndex: 0,
				Txs: [][]byte{
					GetHexByte("0x9a5b716d6ff3d51f9196c579a49f724f0176ee7fc8fa618fd9aee3e10c002e18"),
					GetHexByte("0xe7435b7c408dd6a4589bf6fda96bf4808523443571ed9bc175ea72930a88808b"),
					GetHexByte("0x2e58685b7be596138707fd45c42854be9603822476be50be39164e71b0a49e6e"),
					GetHexByte("0xa99c75904cb96b7e557f3dde1c5e90f43f670d616b2fe57ef9e0a3b983f9a908"),
				},
			}
			dataChunkBytes, _ := dataChunk.Marshal()
			return &tpchaintypes.Block{
				Head: &tpchaintypes.BlockHead{
					ChainID:         []byte(ChainId),
					Version:         1,
					Height:          10,
					Epoch:           1,
					Round:           1,
					ParentBlockHash: GetHexByte("0x6ab6aff3346d3dc27b1fa87ece4fdb83dff42207d692179128ebd56b31229acc"),
					Proposer:        GetHexByte("0x9c71fbe2d28080b8afa88cea8a1e319de2c09d44"),
					StateRoot:       GetHexByte("0xb512f0bd7b64481857e0bbd1d7d2c90578d6f7ad2d20e82f415980a99b0e38ee"),
					GasFees:         gasFees,
					ChunkCount:      1,
					HeadChunks:      [][]byte{hdChunkBytes},
					TimeStamp:       0x62656db0,
					Hash:            GetHexByte("0x983cd9063e6760ab4c7b1db96f3cbaa78588b7005516d3b4fdaad23fdde99499"),
				},
				Data: &tpchaintypes.BlockData{
					Version:    1,
					DataChunks: [][]byte{dataChunkBytes},
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

	j := types2.JsonrpcMessage{}
	err := json.Unmarshal(result, &j)
	if err != nil {
		return
	}

	ethBlock := handlers.GetBlockResponseType{}
	err := json.Unmarshal(j.Result, &ethBlock)
	if err != nil {
		return
	}

	if ethBlock.ParentHash.Hex() != "0x6ab6aff3346d3dc27b1fa87ece4fdb83dff42207d692179128ebd56b31229acc" {
		t.Errorf("failed")
	}
}

func TestGetBlockNumber(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	servantMock := mocks2.NewMockAPIServant(ctrl)
	servantMock.
		EXPECT().
		GetLatestBlock().
		DoAndReturn(func() (*tpchaintypes.Block, error) {
			hdChunk := &tpchaintypes.BlockHeadChunk{
				Version:      tpchaintypes.BLOCK_VER,
				DomainID:     []byte("topiaexe"),
				Launcher:     []byte("0x9c71fbe2d28080b8afa88cea8a1e319de2c09d44"),
				TxCount:      4,
				TxRoot:       []byte("0x4ea6e8ed3f28744b8cb239b64150f024e3eb8f0ff4491acb14dc2e821a04d463"),
				TxResultRoot: []byte("0x4db6969931ba48e0e4073b7699fc32a7c1c6f738339b22f6a2f02279a814bb19"),
			}
			hdChunkBytes, _ := hdChunk.Marshal()

			dataChunk := &tpchaintypes.BlockDataChunk{
				Version:  tpchaintypes.BLOCK_VER,
				RefIndex: 0,
				Txs: [][]byte{
					GetHexByte("0x9a5b716d6ff3d51f9196c579a49f724f0176ee7fc8fa618fd9aee3e10c002e18"),
					GetHexByte("0xe7435b7c408dd6a4589bf6fda96bf4808523443571ed9bc175ea72930a88808b"),
					GetHexByte("0x2e58685b7be596138707fd45c42854be9603822476be50be39164e71b0a49e6e"),
					GetHexByte("0xa99c75904cb96b7e557f3dde1c5e90f43f670d616b2fe57ef9e0a3b983f9a908"),
				},
			}
			dataChunkBytes, _ := dataChunk.Marshal()
			return &tpchaintypes.Block{
				Head: &tpchaintypes.BlockHead{
					ChainID:         []byte(ChainId),
					Version:         1,
					Height:          10,
					Epoch:           1,
					Round:           1,
					ChunkCount:      1,
					HeadChunks:      [][]byte{hdChunkBytes},
					ParentBlockHash: []byte("0x6ab6aff3346d3dc27b1fa87ece4fdb83dff42207d692179128ebd56b31229acc"),
					Proposer:        []byte("0x9c71fbe2d28080b8afa88cea8a1e319de2c09d44"),
					StateRoot:       []byte("0xb512f0bd7b64481857e0bbd1d7d2c90578d6f7ad2d20e82f415980a99b0e38ee"),
					GasFees:         []byte("0x68fc0"),
					TimeStamp:       0x62656db0,
					Hash:            []byte("0x983cd9063e6760ab4c7b1db96f3cbaa78588b7005516d3b4fdaad23fdde99499"),
				},
				Data: &tpchaintypes.BlockData{
					Version:    1,
					DataChunks: [][]byte{dataChunkBytes},
				},
			}, nil
		}).
		Times(1)
	body := `{
		"jsonrpc":"2.0",
		"method":"eth_blockNumber",
		"params":[],
		"id":83
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

	j := types2.JsonrpcMessage{}
	err := json.Unmarshal(result, &j)
	if err != nil {
		return
	}

	ethBlock := new(hexutil.Uint64)
	err = json.Unmarshal(j.Result, ethBlock)
	if err != nil {
		return
	}

	if ethBlock.String() != "0xa" {
		t.Errorf("failed")
	}
}

func GetHexByte(hash string) []byte {
	if strings.HasPrefix(hash, "0x") {
		hash = hash[2:]
	}

	hexStr, _ := hex.DecodeString(hash)
	if len(hash) == 64 {
		var hash eth_account.Hash
		hash.SetBytes(hexStr)
		return hash.Bytes()
	} else {
		var addr eth_account.Address
		addr.SetBytes(hexStr)
		return addr.Bytes()
	}
}
