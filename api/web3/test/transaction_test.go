package test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/TopiaNetwork/topia/api/web3"
	types2 "github.com/TopiaNetwork/topia/api/web3/eth/types"
	"github.com/TopiaNetwork/topia/api/web3/eth/types/eth_account"
	"github.com/TopiaNetwork/topia/api/web3/eth/types/eth_transaction"
	hexutil "github.com/TopiaNetwork/topia/api/web3/eth/types/hexutil"
	"github.com/TopiaNetwork/topia/api/web3/handlers"
	mocks2 "github.com/TopiaNetwork/topia/api/web3/mocks"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txuni "github.com/TopiaNetwork/topia/transaction/universal"
	"github.com/golang/mock/gomock"
	"io"
	"math/big"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGetTransactionByHash(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	servantMock := mocks2.NewMockAPIServant(ctrl)
	servantMock.
		EXPECT().
		GetTransactionByHash(gomock.Any()).
		DoAndReturn(func(txHashHex string) (*txbasic.Transaction, error) {
			to := []byte("0x6fcd7b39e75619a68ab86a68b92d01134ef34ea3")
			toAddr := eth_account.Address{}
			toAddr.SetBytes(to)

			rByte, _ := hex.DecodeString("fe4cd7376c6df62df3f8c60c6b82d90582c0b6922658d038665548eda98fc9f5")
			sByte, _ := hex.DecodeString("08725bdf6cb82a27e4798cbcb70fa220e1d7459353ef1a2c716f8101f4c9c601")

			ethLegacyTx := eth_transaction.NewTx(&eth_transaction.DynamicFeeTx{
				ChainID:   big.NewInt(3),
				Nonce:     0x0,
				To:        &toAddr,
				Value:     big.NewInt(0x38d7ea4c68000),
				Gas:       0x5208,
				GasFeeCap: big.NewInt(0x3ebd697803),
				GasTipCap: big.NewInt(0x8c347c90),
				V:         big.NewInt(0x1),
				R:         new(big.Int).SetBytes(rByte),
				S:         new(big.Int).SetBytes(sByte),
			})
			tx, _ := handlers.ConvertEthTransactionToTopiaTransaction(ethLegacyTx)
			return tx, nil
		}).
		Times(1)

	servantMock.
		EXPECT().
		GetBlockByTxHash(gomock.Any()).
		DoAndReturn(func(txHashHex string) (*tpchaintypes.Block, error) {
			return &tpchaintypes.Block{
				Head: &tpchaintypes.BlockHead{
					ChainID:         []byte(ChainId),
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
		}).
		Times(1)

	body := `{
		"jsonrpc":"2.0",
		"method":"eth_getTransactionByHash",
		"params":[
			"0xeecab57a7ce01d5adf81b6266de37bd74d3bbd7d729b0e1ca35163d304549ee6"
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
	json.Unmarshal(result, &j)
	ethTransaction := handlers.EthRpcTransaction{}
	json.Unmarshal(j.Result, &ethTransaction)

	if ethTransaction.Nonce != 0 {
		t.Errorf("failed")
	}
}
func TestGetTransactionCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	servantMock := mocks2.NewMockAPIServant(ctrl)
	servantMock.
		EXPECT().
		GetTransactionCount(gomock.Any(), gomock.Any()).
		DoAndReturn(func(addr tpcrtypes.Address, height uint64) (uint64, error) {
			return 10, nil
		}).
		Times(1)

	body := `{
		"jsonrpc":"2.0",
		"method":"eth_getTransactionCount",
		"params":[
			"0x3F1B3C065aeE8cA34c47fa84aAC3024E95a2E6D9",
			"latest"
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

	j := types2.JsonrpcMessage{}
	json.Unmarshal(result, &j)

	count := new(hexutil.Uint64)
	json.Unmarshal(j.Result, count)
	if count.String() != "0xa" {
		t.Errorf("failed")
	}
}
func TestGetTransactionReceipt(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	servantMock := mocks2.NewMockAPIServant(ctrl)
	servantMock.
		EXPECT().
		GetTransactionResultByHash(gomock.Any()).
		DoAndReturn(func(txHashHex string) (*txbasic.TransactionResult, error) {
			resultUniversal := txuni.TransactionResultUniversal{
				Version:   uint32(1),
				TxHash:    []byte("0x66121743b3eebd28626da9297f4e5a9ecc72aa97340513491824aa6ca4f21974"),
				GasUsed:   100_000,
				ErrString: nil,
				Status:    txuni.TransactionResultUniversal_OK,
			}

			marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
			txUniRSBytes, err := marshaler.Marshal(&resultUniversal)
			if err != nil {

			}

			return &txbasic.TransactionResult{
				Head: &txbasic.TransactionResultHead{
					Category: []byte(txbasic.TransactionCategory_Eth),
					ChainID:  []byte(ChainId),
					Version:  txbasic.Transaction_Eth_V1,
				},
				Data: &txbasic.TransactionResultData{
					Specification: txUniRSBytes,
				},
			}, nil
		}).
		Times(1)

	servantMock.
		EXPECT().
		GetTransactionByHash(gomock.Any()).
		DoAndReturn(func(txHashHex string) (*txbasic.Transaction, error) {
			to := []byte("0x6fcd7b39e75619a68ab86a68b92d01134ef34ea3")
			toAddr := eth_account.Address{}
			toAddr.SetBytes(to)

			rByte, _ := hex.DecodeString("fe4cd7376c6df62df3f8c60c6b82d90582c0b6922658d038665548eda98fc9f5")
			sByte, _ := hex.DecodeString("08725bdf6cb82a27e4798cbcb70fa220e1d7459353ef1a2c716f8101f4c9c601")

			ethLegacyTx := eth_transaction.NewTx(&eth_transaction.DynamicFeeTx{
				ChainID:   big.NewInt(9),
				Nonce:     0x0,
				To:        &toAddr,
				Value:     big.NewInt(0x38d7ea4c68000),
				Gas:       0x5208,
				GasFeeCap: big.NewInt(0x3ebd697803),
				GasTipCap: big.NewInt(0x8c347c90),
				V:         big.NewInt(0x1),
				R:         new(big.Int).SetBytes(rByte),
				S:         new(big.Int).SetBytes(sByte),
			})
			tx, _ := handlers.ConvertEthTransactionToTopiaTransaction(ethLegacyTx)
			return tx, nil
		}).
		Times(1)

	servantMock.
		EXPECT().
		GetBlockByTxHash(gomock.Any()).
		DoAndReturn(func(txHashHex string) (*tpchaintypes.Block, error) {
			return &tpchaintypes.Block{
				Head: &tpchaintypes.BlockHead{
					ChainID:         []byte(ChainId),
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
		}).
		Times(1)

	body := `{
		"jsonrpc":"2.0",
		"method":"eth_getTransactionReceipt",
		"params":[
			"0xd10620fe7a7c9b49bedcb33b120758b7ee9b3c6c94eb2ff0aa52ed006e0ddf29"
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
	json.Unmarshal(result, &j)

	ethReceipt := handlers.GetTransactionReceiptResponseType{}
	json.Unmarshal(j.Result, &ethReceipt)

	if ethReceipt.Status != "0x1" {
		t.Errorf("failed")
	}
}
func TestSendRawTransaction(t *testing.T) {
	body := `{
		"jsonrpc":"2.0",
		"method":"eth_sendRawTransaction",
		"params":["0x02f873030b8459682f0085025c90ff50825208946fcd7b39e75619a68ab86a68b92d01134ef34ea388016345785d8a000080c001a00468068551701a4eb935052c207598a6c0d4810242a838cbf05848f15087be5ea03fdd0e86b4c8642d19bedb5aa12e036a6e38a286ab4bb9b250e045580784682f"],
		"id":1
	}`

	req := httptest.NewRequest("POST", "http://localhost:8080/", strings.NewReader(body))
	res := httptest.NewRecorder()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servantMock := mocks2.NewMockAPIServant(ctrl)
	servantMock.
		EXPECT().
		ForwardTxSync(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, tran *txbasic.Transaction) (*txbasic.TransactionResult, error) {
			resultUniversal := txuni.TransactionResultUniversal{
				Version:   1,
				TxHash:    GetHexByte("0x41597ee75a4de42f30930a1da4db24d932cc4ac688452ac7b0fe547df80b7716"),
				GasUsed:   100_000,
				ErrString: nil,
				Status:    txuni.TransactionResultUniversal_OK,
			}

			marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
			txUniRSBytes, err := marshaler.Marshal(&resultUniversal)
			if err != nil {

			}

			result := &txbasic.TransactionResult{
				Head: &txbasic.TransactionResultHead{
					Category: []byte(txbasic.TransactionCategory_Eth),
					ChainID:  []byte(ChainId),
					Version:  txbasic.Transaction_Eth_V1,
				},
				Data: &txbasic.TransactionResultData{
					Specification: txUniRSBytes,
				},
			}

			var txUniversal txuni.TransactionUniversal
			_ = json.Unmarshal(tran.GetData().GetSpecification(), &txUniversal)
			txUniversalHead := txUniversal.GetHead()
			txType := txUniversalHead.GetType()
			switch txType {
			case uint32(txuni.TransactionUniversalType_Transfer):
				var txTransfer txuni.TransactionUniversalTransfer
				_ = json.Unmarshal(txUniversal.GetData().GetSpecification(), &txTransfer)
				from := handlers.AddPrefix(strings.ToLower(hex.EncodeToString(tran.GetHead().GetFromAddr())))
				to, _ := txTransfer.TargetAddr.HexString()
				to = handlers.AddPrefix(strings.ToLower(to))
				value := new(big.Int).Add(txTransfer.Targets[0].Value, new(big.Int).SetUint64(txUniversalHead.GetGasPrice()*txUniversalHead.GetGasLimit()))

				fromAccount := Accounts[from]
				toAccount := Accounts[to]
				if value.Cmp(fromAccount.GetBalance()) < 0 {
					fromAccount.SubBalance(value)
					fromAccount.AddNonce()
					toAccount.AddBalance(value)
					return result, nil
				} else {
					return result, errors.New("not enough balance")
				}
			case uint32(txuni.TransactionUniversalType_ContractInvoke):
				from := handlers.AddPrefix(strings.ToLower(hex.EncodeToString(tran.GetHead().GetFromAddr())))
				value := new(big.Int).SetUint64(txUniversalHead.GetGasPrice() * txUniversalHead.GetGasLimit())

				fromAccount := Accounts[from]
				if value.Cmp(fromAccount.GetBalance()) < 0 && fromAccount.GetCode() != nil {
					fromAccount.SubBalance(value)
					fromAccount.AddNonce()
					return result, nil
				} else {
					return result, errors.New("not enough balance or code not exist")
				}
			case uint32(txuni.TransactionUniversalType_ContractDeploy):
				from := handlers.AddPrefix(strings.ToLower(hex.EncodeToString(tran.GetHead().GetFromAddr())))
				code := tran.GetData().GetSpecification()
				value := new(big.Int).SetUint64(txUniversalHead.GetGasPrice() * txUniversalHead.GetGasLimit())

				fromAccount := Accounts[from]
				if value.Cmp(fromAccount.GetBalance()) < 0 {
					fromAccount.SubBalance(value)
					fromAccount.SetCode(code)
					fromAccount.AddNonce()
					return result, nil
				} else {
					return result, errors.New("not enough balance")
				}
			default:
				return result, errors.New("unknown txType")
			}
		}).
		AnyTimes()
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
	json.Unmarshal(result, &j)

	hash := new(eth_account.Hash)
	err := json.Unmarshal(j.Result, hash)
	if err != nil {
		t.Errorf("failed")
	}
	if hash.String() != "0x297a532dda774fc86d3244a579c10ccc671a007297e0cb61abd25275c4b2021a" {
		t.Errorf("failed")
	}
}
