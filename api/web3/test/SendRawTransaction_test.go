package test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/TopiaNetwork/topia/api/mocks"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3"
	"github.com/TopiaNetwork/topia/api/web3/handlers"
	"github.com/TopiaNetwork/topia/api/web3/types"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txuni "github.com/TopiaNetwork/topia/transaction/universal"
	"github.com/golang/mock/gomock"
	"io"
	"math/big"
	"net/http/httptest"
	"strings"
	"testing"
)

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
	apiServant := servant.NewAPIServant()
	txInterfaceMock := mocks.NewMockTxInterface(ctrl)
	txInterfaceMock.
		EXPECT().
		SendTransaction(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, tran *txbasic.Transaction) error {
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
					return nil
				} else {
					return errors.New("not enough balance")
				}
			case uint32(txuni.TransactionUniversalType_ContractInvoke):
				from := handlers.AddPrefix(strings.ToLower(hex.EncodeToString(tran.GetHead().GetFromAddr())))
				value := new(big.Int).SetUint64(txUniversalHead.GetGasPrice() * txUniversalHead.GetGasLimit())

				fromAccount := Accounts[from]
				if value.Cmp(fromAccount.GetBalance()) < 0 && fromAccount.GetCode() != nil {
					fromAccount.SubBalance(value)
					fromAccount.AddNonce()
					return nil
				} else {
					return errors.New("not enough balance or code not exist")
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
					return nil
				} else {
					return errors.New("not enough balance")
				}
			default:
				return errors.New("unknown txType")
			}
		}).
		AnyTimes()
	config := web3.Web3ServerConfiguration{
		Host: "",
		Port: "8080",
	}
	w3s := web3.InitWeb3Server(config, apiServant, txInterfaceMock)

	w3s.ServeHttp(res, req)
	result, _ := io.ReadAll(res.Result().Body)
	j := types.JsonrpcMessage{}
	json.Unmarshal(result, &j)

	hash := new(types.Hash)
	err := json.Unmarshal(j.Result, hash)
	if err != nil {
		t.Errorf("failed")
	}
	if hash.String() != "0x297a532dda774fc86d3244a579c10ccc671a007297e0cb61abd25275c4b2021a" {
		t.Errorf("failed")
	}
}
