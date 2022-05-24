package test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TopiaNetwork/topia/api/web3"
	types2 "github.com/TopiaNetwork/topia/api/web3/eth/types"
	"github.com/TopiaNetwork/topia/api/web3/eth/types/eth_account"
	"github.com/TopiaNetwork/topia/api/web3/eth/types/eth_transaction"
	handlers "github.com/TopiaNetwork/topia/api/web3/handlers"
	mocks2 "github.com/TopiaNetwork/topia/api/web3/mocks"
	tpchaintypes "github.com/TopiaNetwork/topia/chain/types"
	"github.com/TopiaNetwork/topia/codec"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/currency"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txuni "github.com/TopiaNetwork/topia/transaction/universal"
	"github.com/golang/mock/gomock"
	"math/big"
	"strings"
	"testing"
)

var Accounts = initAccount()

const ChainId = "9"

func TestWeb3Server(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	height := uint64(100)

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

	servantMock.
		EXPECT().
		ExecuteTxSim(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, tx *txbasic.Transaction) (*txbasic.TransactionResult, error) {
			resu, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000000")

			resultUniversal := &txuni.TransactionResultUniversal{
				Version:   1,
				TxHash:    GetHexByte("0x66121743b3eebd28626da9297f4e5a9ecc72aa97340513491824aa6ca4f21974"),
				GasUsed:   100_000,
				ErrString: nil,
				Status:    txuni.TransactionResultUniversal_OK,
				Data:      resu,
			}

			marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
			txUniRSBytes, err := marshaler.Marshal(resultUniversal)
			if err != nil {
				fmt.Println(err)
			}
			aaa := &txbasic.TransactionResult{
				Head: &txbasic.TransactionResultHead{
					Category: []byte(txbasic.TransactionCategory_Eth),
					ChainID:  []byte(ChainId),
					Version:  txbasic.Transaction_Eth_V1,
				},
				Data: &txbasic.TransactionResultData{
					Specification: txUniRSBytes,
				},
			}
			return aaa, nil
		}).
		AnyTimes()
	servantMock.
		EXPECT().
		GetBlockByTxHash(gomock.Any()).
		DoAndReturn(func(txHashHex string) (*tpchaintypes.Block, error) {
			gasFees, _ := json.Marshal("0x68fc0")
			return &tpchaintypes.Block{
				Head: &tpchaintypes.BlockHead{
					ChainID:         []byte(ChainId),
					Version:         1,
					Height:          100,
					Epoch:           1,
					Round:           1,
					ParentBlockHash: GetHexByte("0x6ab6aff3346d3dc27b1fa87ece4fdb83dff42207d692179128ebd56b31229acc"),
					Launcher:        GetHexByte("0x9c71fbe2d28080b8afa88cea8a1e319de2c09d44"),
					Proposer:        GetHexByte("0x9c71fbe2d28080b8afa88cea8a1e319de2c09d44"),
					TxCount:         4,
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
						GetHexByte("0x41597ee75a4de42f30930a1da4db24d932cc4ac688452ac7b0fe547df80b7716"),
						GetHexByte("0xe7435b7c408dd6a4589bf6fda96bf4808523443571ed9bc175ea72930a88808b"),
						GetHexByte("0x2e58685b7be596138707fd45c42854be9603822476be50be39164e71b0a49e6e"),
						GetHexByte("0xa99c75904cb96b7e557f3dde1c5e90f43f670d616b2fe57ef9e0a3b983f9a908"),
					},
				},
			}, nil
		}).
		AnyTimes()
	servantMock.
		EXPECT().
		EstimateGas(gomock.Any()).
		DoAndReturn(func(tx *txbasic.Transaction) (*big.Int, error) {
			return big.NewInt(30000), nil
		}).
		AnyTimes()
	servantMock.
		EXPECT().
		GetLatestBlock().
		DoAndReturn(func() (*tpchaintypes.Block, error) {
			height = height + 10
			gasFees, _ := json.Marshal("0x68fc0")
			return &tpchaintypes.Block{
				Head: &tpchaintypes.BlockHead{
					ChainID:         []byte(ChainId),
					Version:         1,
					Height:          height,
					Epoch:           1,
					Round:           1,
					ParentBlockHash: GetHexByte("0x6ab6aff3346d3dc27b1fa87ece4fdb83dff42207d692179128ebd56b31229acc"),
					Launcher:        GetHexByte("0x9c71fbe2d28080b8afa88cea8a1e319de2c09d44"),
					Proposer:        GetHexByte("0x9c71fbe2d28080b8afa88cea8a1e319de2c09d44"),
					TxCount:         4,
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
		AnyTimes()
	servantMock.
		EXPECT().
		GetBlockByHeight(gomock.Any()).
		DoAndReturn(func(height uint64) (*tpchaintypes.Block, error) {
			gasFees, _ := json.Marshal("0x68fc0")
			return &tpchaintypes.Block{
				Head: &tpchaintypes.BlockHead{
					ChainID:         []byte(ChainId),
					Version:         1,
					Height:          100,
					Epoch:           1,
					Round:           1,
					ParentBlockHash: GetHexByte("0x6ab6aff3346d3dc27b1fa87ece4fdb83dff42207d692179128ebd56b31229acc"),
					Launcher:        GetHexByte("0x9c71fbe2d28080b8afa88cea8a1e319de2c09d44"),
					Proposer:        GetHexByte("0x9c71fbe2d28080b8afa88cea8a1e319de2c09d44"),
					TxCount:         4,
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
						GetHexByte("0x41597ee75a4de42f30930a1da4db24d932cc4ac688452ac7b0fe547df80b7716"),
						GetHexByte("0xe7435b7c408dd6a4589bf6fda96bf4808523443571ed9bc175ea72930a88808b"),
						GetHexByte("0x2e58685b7be596138707fd45c42854be9603822476be50be39164e71b0a49e6e"),
						GetHexByte("0xa99c75904cb96b7e557f3dde1c5e90f43f670d616b2fe57ef9e0a3b983f9a908"),
					},
				},
			}, nil
		}).
		AnyTimes()
	servantMock.
		EXPECT().
		GetTransactionByHash(gomock.Any()).
		DoAndReturn(func(txHashHex string) (*txbasic.Transaction, error) {
			to, _ := hex.DecodeString("6fcd7b39e75619a68ab86a68b92d01134ef34ea3")
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
			tx, err := handlers.ConvertEthTransactionToTopiaTransaction(ethLegacyTx)
			if err != nil {
				fmt.Println(err)
			}
			return tx, nil
		}).
		AnyTimes()
	servantMock.
		EXPECT().
		GetBalance(gomock.Any(), gomock.Any()).
		DoAndReturn(func(symbol currency.TokenSymbol, addr tpcrtypes.Address) (*big.Int, error) {
			address, _ := addr.HexString()
			addre := strings.ToLower(address)
			if addrAccount, ok := Accounts[addre]; ok {
				return addrAccount.GetBalance(), nil
			} else {
				return big.NewInt(0), nil
			}
		}).
		AnyTimes()
	servantMock.
		EXPECT().
		GetBlockByHash(gomock.Any()).
		DoAndReturn(func(txHashHex string) (*tpchaintypes.Block, error) {
			gasFees, _ := json.Marshal("0x68fc0")
			return &tpchaintypes.Block{
				Head: &tpchaintypes.BlockHead{
					ChainID:         []byte(ChainId),
					Version:         1,
					Height:          100,
					Epoch:           1,
					Round:           1,
					ParentBlockHash: GetHexByte("0x6ab6aff3346d3dc27b1fa87ece4fdb83dff42207d692179128ebd56b31229acc"),
					Launcher:        GetHexByte("0x9c71fbe2d28080b8afa88cea8a1e319de2c09d44"),
					Proposer:        GetHexByte("0x9c71fbe2d28080b8afa88cea8a1e319de2c09d44"),
					TxCount:         4,
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
						GetHexByte("0x41597ee75a4de42f30930a1da4db24d932cc4ac688452ac7b0fe547df80b7716"),
						GetHexByte("0xe7435b7c408dd6a4589bf6fda96bf4808523443571ed9bc175ea72930a88808b"),
						GetHexByte("0x2e58685b7be596138707fd45c42854be9603822476be50be39164e71b0a49e6e"),
						GetHexByte("0xa99c75904cb96b7e557f3dde1c5e90f43f670d616b2fe57ef9e0a3b983f9a908"),
					},
				},
			}, nil
		}).
		AnyTimes()
	servantMock.
		EXPECT().
		GetContractCode(gomock.Any(), gomock.Any()).
		DoAndReturn(func(addr tpcrtypes.Address, height uint64) ([]byte, error) {
			address, _ := addr.HexString()
			address = strings.ToLower(address)
			addrAccount := Accounts[address]
			if addrAccount != nil {
				//return addrAccount.GetCode(), nil
				return nil, nil
			}
			return []byte("608060405260043610601f5760003560e01c80632e1a7d4d14602a576025565b36602557005b600080fd5b348015603557600080fd5b50605f60048036036020811015604a57600080fd5b81019080803590602001909291905050506061565b005b67016345785d8a0000811115607557600080fd5b3373ffffffffffffffffffffffffffffffffffffffff166108fc829081150290604051600060405180830381858888f1935050505015801560ba573d6000803e3d6000fd5b505056fea26469706673582212206172611ad1a5f939aa5c8fb05fccb7a60b2099c3473ad8194417684f5e56257364736f6c63430006040033"), nil
		}).
		AnyTimes()
	servantMock.
		EXPECT().
		GetTransactionCount(gomock.Any(), gomock.Any()).
		DoAndReturn(func(addr tpcrtypes.Address, height uint64) (uint64, error) {
			return 10, nil
		}).
		AnyTimes()
	servantMock.
		EXPECT().
		GetTransactionResultByHash(gomock.Any()).
		DoAndReturn(func(txHashHex string) (*txbasic.TransactionResult, error) {
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
		AnyTimes()
	servantMock.
		EXPECT().
		ChainID().
		DoAndReturn(func() tpchaintypes.ChainID {
			return tpchaintypes.ChainID(ChainId)
		}).
		AnyTimes()

	config := web3.Web3ServerConfiguration{
		HttpHost:  "localhost",
		HttpPort:  "8080",
		HttpsHost: "localhost",
		HttpsPost: "8443",
	}
	web3.StartServer(config, servantMock)
}

func initAccount() map[string]*types2.Account {
	balance, _ := new(big.Int).SetString("1000000000000000000000", 10)

	return map[string]*types2.Account{
		strings.ToLower("0x3F1B3C065aeE8cA34c47fa84aAC3024E95a2E6D9"): &types2.Account{
			Addr:    []byte("0x3F1B3C065aeE8cA34c47fa84aAC3024E95a2E6D9"),
			Balance: new(big.Int).Set(balance),
			Nonce:   0,
			Code:    nil,
		},
		strings.ToLower("0x6FCd7b39E75619A68Ab86a68B92d01134Ef34ea3"): &types2.Account{
			Addr:    []byte("0x6FCd7b39E75619A68Ab86a68B92d01134Ef34ea3"),
			Balance: new(big.Int).Set(balance),
			Nonce:   0,
			Code:    nil,
		},
		strings.ToLower("0x825021c776Ea10fF688147063D7Ad96f05E232DA"): &types2.Account{
			Addr:    []byte("0x825021c776Ea10fF688147063D7Ad96f05E232DA"),
			Balance: new(big.Int).Set(balance),
			Nonce:   0,
			Code:    nil,
		},
		strings.ToLower("0x36Df1457573BC4947c4b2bf8C66e7f290e6C50e6"): &types2.Account{
			Addr:    []byte("0x36Df1457573BC4947c4b2bf8C66e7f290e6C50e6"),
			Balance: new(big.Int).Set(balance),
			Nonce:   0,
			Code:    nil,
		},
		strings.ToLower("0x141F79b9a561C775D76B8FC0a62ec85089d5f40D"): &types2.Account{
			Addr:    []byte("0x141F79b9a561C775D76B8FC0a62ec85089d5f40D"),
			Balance: new(big.Int).Set(balance),
			Nonce:   0,
			Code:    nil,
		},
		strings.ToLower("0x621d4e17CAd065De6B25c73B0c97ef81cFF5B1Ee"): &types2.Account{
			Addr:    []byte("0x621d4e17CAd065De6B25c73B0c97ef81cFF5B1Ee"),
			Balance: new(big.Int).Set(balance),
			Nonce:   0,
			Code:    nil,
		},
		strings.ToLower("0x047B52185ad4E9E695D3B077De8C155CE7b514D3"): &types2.Account{
			Addr:    []byte("0x047B52185ad4E9E695D3B077De8C155CE7b514D3"),
			Balance: new(big.Int).Set(balance),
			Nonce:   0,
			Code:    nil,
		},
		strings.ToLower("0xC157Fdf20cDD376eFB537E0a40F64c87c4DEEDF6"): &types2.Account{
			Addr:    []byte("0xC157Fdf20cDD376eFB537E0a40F64c87c4DEEDF6"),
			Balance: new(big.Int).Set(balance),
			Nonce:   0,
			Code:    nil,
		},
	}
}
