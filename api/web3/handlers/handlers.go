package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/TopiaNetwork/topia/api/servant"
	types2 "github.com/TopiaNetwork/topia/api/web3/eth/types"
	"github.com/TopiaNetwork/topia/api/web3/eth/types/eth_account"
	hexutil "github.com/TopiaNetwork/topia/api/web3/eth/types/hexutil"
	"github.com/TopiaNetwork/topia/codec"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/currency"
	"github.com/TopiaNetwork/topia/transaction/universal"
	"math/big"
	"reflect"
	"strconv"
)

type HandlerService interface {
	CallHandler(parmas interface{}) interface{}
	ChainIdHandler(parmas interface{}) interface{}
	EstimateGasHandler(parmas interface{}) interface{}
	FeeHistoryHandler(parmas interface{}) interface{}
	GasPriceHandler(parmas interface{}) interface{}
	GetAccountsHandler(parmas interface{}) interface{}
	GetBalanceHandler(parmas interface{}) interface{}
	GetBlockByHashHandler(parmas interface{}) interface{}
	GetBlockByNumberHandler(parmas interface{}) interface{}
	GetBlockNumberHandler(parmas interface{}) interface{}
	GetCodeHandler(parmas interface{}) interface{}
	GetTransactionByHashHandler(parmas interface{}) interface{}
	GetTransactionCountHandler(parmas interface{}) interface{}
	GetTransactionReceiptHandler(parmas interface{}) interface{}
	NetListeningHandler(parmas interface{}) interface{}
	NetVersionHandler(parmas interface{}) interface{}
	SendRawTransactionHandler(parmas interface{}) interface{}
}

type Handler struct {
	Servant servant.APIServant
}

func (h *Handler) CallHandler(parmas interface{}) interface{} {
	args := parmas.(*CallRequestType)

	ctx := context.Background()
	tx := ConstructTransaction(*args)
	callResult, err := h.Servant.ExecuteTxSim(ctx, tx)
	if err != nil {
		return types2.ErrorMessage(err)
	}
	var transactionResultUniversal universal.TransactionResultUniversal
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	_ = marshaler.Unmarshal(callResult.GetData().GetSpecification(), &transactionResultUniversal)
	result := transactionResultUniversal.GetData()

	enc, err := json.Marshal(hexutil.Bytes(result))
	if err != nil {
		return types2.ErrorMessage(err)
	}
	return &types2.JsonrpcMessage{Result: enc}
}
func (h *Handler) ChainIdHandler(parmas interface{}) interface{} {
	chainId, _ := new(big.Int).SetString(string(h.Servant.ChainID()), 10)
	chId := (*hexutil.Big)(chainId)
	enc, err := json.Marshal(chId)
	if err != nil {
		return types2.ErrorMessage(err)
	}
	return &types2.JsonrpcMessage{Result: enc}
}
func (h *Handler) EstimateGasHandler(parmas interface{}) interface{} {
	args := parmas.(*EstimateGasRequestType)

	tx := ConstructGasTransaction(*args)
	estimateGas, err := h.Servant.EstimateGas(tx)
	if err != nil {
		return types2.ErrorMessage(err)
	}
	gas := hexutil.Uint64(new(big.Int).SetBytes(estimateGas.Bytes()).Uint64())

	enc, err := json.Marshal(gas)
	if err != nil {
		return types2.ErrorMessage(err)
	}
	return &types2.JsonrpcMessage{Result: enc}
}
func (h *Handler) FeeHistoryHandler(parmas interface{}) interface{} {
	args := parmas.(*FeeHistoryRequestType)

	feeHistory := FeeHistory(args, h.Servant)

	enc, err := json.Marshal(feeHistory)
	if err != nil {
		return types2.ErrorMessage(err)
	}
	return &types2.JsonrpcMessage{Result: enc}
}
func (h *Handler) GasPriceHandler(parmas interface{}) interface{} {
	gasPrice, err := EstimateTipFee(h.Servant)
	if err != nil {
		return types2.ErrorMessage(err)
	}

	enc, err := json.Marshal((*hexutil.Big)(gasPrice))
	if err != nil {
		return types2.ErrorMessage(err)
	}
	return &types2.JsonrpcMessage{Result: enc}
}
func (h *Handler) GetAccountsHandler(parmas interface{}) interface{} {
	var accounts []string
	accounts = append(accounts, "0x3F1B3C065aeE8cA34c47fa84aAC3024E95a2E6D9")

	enc, err := json.Marshal(accounts)
	if err != nil {
		//TODO:
		return types2.ErrorMessage(err)
	}
	return &types2.JsonrpcMessage{Result: enc}
}
func (h *Handler) GetBalanceHandler(parmas interface{}) interface{} {
	args := parmas.(*GetBalanceRequestType)

	ad, err := tpcrtypes.TopiaAddressFromEth(args.Address)
	if err != nil {
		return types2.ErrorMessage(err)
	}

	balance, err := h.Servant.GetBalance(currency.TokenSymbol_Native, ad)
	if err != nil {
		return types2.ErrorMessage(err)
	}
	var enc []byte
	if balance.BitLen() != 0 {
		enc, err = json.Marshal((*hexutil.Big)(balance))
	} else {
		enc, err = json.Marshal((*hexutil.Big)(big.NewInt(0)))
	}
	if err != nil {
		return types2.ErrorMessage(err)
	}
	return &types2.JsonrpcMessage{Result: enc}
}
func (h *Handler) GetBlockByHashHandler(parmas interface{}) interface{} {
	args := parmas.(*GetBlockByHashRequestType)

	block, err := h.Servant.GetBlockByHash(args.BlockHash)
	if err != nil {
		return types2.ErrorMessage(err)
	}
	ethBlock := ConvertTopiaBlockToEthBlock(block)
	enc, err := json.Marshal(ethBlock)
	if err != nil {
		return types2.ErrorMessage(err)
	}
	return &types2.JsonrpcMessage{Result: enc}
}
func (h *Handler) GetBlockByNumberHandler(parmas interface{}) interface{} {
	args := parmas.(*GetBlockByNumberRequestType)

	var height uint64
	number := args.Height
	if value, ok := number.(string); ok {
		height, _ = strconv.ParseUint(value[2:], 16, 64)
	} else if value, ok := number.(float64); ok {
		height = uint64(value)
	}

	block, err := h.Servant.GetBlockByHeight(height)
	if err != nil {
		return types2.ErrorMessage(err)
	}

	ethBlock := ConvertTopiaBlockToEthBlock(block)
	enc, err := json.Marshal(ethBlock)
	if err != nil {
		return types2.ErrorMessage(err)
	}
	return &types2.JsonrpcMessage{Result: enc}
}
func (h *Handler) GetBlockNumberHandler(parmas interface{}) interface{} {
	block, err := h.Servant.GetLatestBlock()
	if err != nil {
		return types2.ErrorMessage(err)
	}
	height := block.GetHead().GetHeight()

	enc, err := json.Marshal(hexutil.Uint64(height))
	if err != nil {
		return types2.ErrorMessage(err)
	}
	return &types2.JsonrpcMessage{Result: enc}
}
func (h *Handler) GetCodeHandler(parmas interface{}) interface{} {
	args := parmas.(*GetCodeRequestType)

	var height uint64
	number := args.Height
	if value, ok := number.(string); ok {
		height, _ = strconv.ParseUint(value[2:], 16, 64)
	} else if value, ok := number.(float64); ok {
		height = uint64(value)
	}

	tpaAddress, err := tpcrtypes.TopiaAddressFromEth(args.Address)
	if err != nil {
		return types2.ErrorMessage(err)
	}

	code, err := h.Servant.GetContractCode(tpaAddress, height)
	if err != nil {
		return types2.ErrorMessage(err)
	}

	enc, err := json.Marshal(hexutil.Bytes(code))
	if err != nil {
		return types2.ErrorMessage(err)
	}
	return &types2.JsonrpcMessage{Result: enc}
}
func (h *Handler) GetTransactionByHashHandler(parmas interface{}) interface{} {
	args := parmas.(*GetTransactionByHashRequestType)

	transaction, err := h.Servant.GetTransactionByHash(args.TxHash)
	if err != nil {
		return types2.ErrorMessage(err)
	}
	ethTransaction, err := TopiaTransactionToEthTransaction(*transaction, h.Servant)
	if err != nil {
		return types2.ErrorMessage(err)
	}
	enc, err := json.Marshal(ethTransaction)
	if err != nil {
		return types2.ErrorMessage(err)
	}
	return &types2.JsonrpcMessage{Result: enc}
}
func (h *Handler) GetTransactionCountHandler(parmas interface{}) interface{} {
	args := parmas.(*GetTransactionCountRequestType)

	height, _ := strconv.ParseUint(args.Height, 10, 64)
	count, err := h.Servant.GetTransactionCount(tpcrtypes.NewFromString(args.Address.String()), height)
	if err != nil {
		return types2.ErrorMessage(err)
	}
	txCount := hexutil.Uint(count)

	enc, err := json.Marshal(txCount)
	if err != nil {
		return types2.ErrorMessage(err)
	}
	return &types2.JsonrpcMessage{Result: enc}
}
func (h *Handler) GetTransactionReceiptHandler(parmas interface{}) interface{} {
	args := parmas.(*GetTransactionReceiptRequestType)

	result, err := h.Servant.GetTransactionResultByHash(args.TxHash.Hex())
	if err != nil {
		return types2.ErrorMessage(err)
	}
	ethReceipt := ConvertTopiaResultToEthReceipt(*result, h.Servant)

	enc, err := json.Marshal(ethReceipt)
	if err != nil {
		return types2.ErrorMessage(err)
	}
	return &types2.JsonrpcMessage{Result: enc}
}
func (h *Handler) NetListeningHandler(parmas interface{}) interface{} {
	enc, err := json.Marshal(true)
	if err != nil {
		return types2.ErrorMessage(err)
	}
	return &types2.JsonrpcMessage{Result: enc}
}
func (h *Handler) NetVersionHandler(parmas interface{}) interface{} {
	enc, err := json.Marshal("9")

	if err != nil {
		return types2.ErrorMessage(err)
	}
	return &types2.JsonrpcMessage{Result: enc}
}
func (h *Handler) SendRawTransactionHandler(parmas interface{}) interface{} {
	args := parmas.(*SendRawTransactionRequestType)

	tx, _ := ConstructTopiaTransaction(*args) // TODO: account type is not uniform, location of the conversion type is not uniform, and it is a mess
	ctx := context.Background()
	result, err := h.Servant.ForwardTxSync(ctx, tx)
	if err != nil {
		return types2.ErrorMessage(err)
	}
	var resultUni universal.TransactionResultUniversal
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	marshaler.Unmarshal(result.GetData().GetSpecification(), &resultUni)

	var hashByte []byte
	if resultUni.GetStatus() == universal.TransactionResultUniversal_OK {
		hashByte, _ = tx.HashBytes()
	} else {
		return types2.ErrorMessage(errors.New("sendTransaction failed!"))
	}

	var hash eth_account.Hash
	hash.SetBytes(hashByte)

	enc, err := json.Marshal(hash)
	if err != nil {
		return types2.ErrorMessage(err)
	}
	return &types2.JsonrpcMessage{Result: enc}
}

func GetApis() []Api {
	return []Api{
		{
			Method: "eth_getBalance",
			Handler: &RequestHandler{
				Param: getValueArray(&RequestType{
					RequestType: &GetBalanceRequestType{},
				}),
				Func: reflect.ValueOf(HandlerService.GetBalanceHandler),
			},
		},
		{
			Method: "eth_getTransactionByHash",
			Handler: &RequestHandler{
				Param: getValueArray(&RequestType{
					RequestType: &GetTransactionByHashRequestType{},
				}),
				Func: reflect.ValueOf(HandlerService.GetTransactionByHashHandler),
			},
		},
		{
			Method: "eth_getTransactionCount",
			Handler: &RequestHandler{
				Param: getValueArray(&RequestType{
					RequestType: &GetTransactionCountRequestType{},
				}),
				Func: reflect.ValueOf(HandlerService.GetTransactionCountHandler),
			},
		},
		{
			Method: "eth_getCode",
			Handler: &RequestHandler{
				Param: getValueArray(&RequestType{
					RequestType: &GetCodeRequestType{},
				}),
				Func: reflect.ValueOf(HandlerService.GetCodeHandler),
			},
		},
		{
			Method: "eth_getBlockByHash",
			Handler: &RequestHandler{
				Param: getValueArray(&RequestType{
					RequestType: &GetBlockByHashRequestType{},
				}),
				Func: reflect.ValueOf(HandlerService.GetBlockByHashHandler),
			},
		},
		{
			Method: "eth_getBlockByNumber",
			Handler: &RequestHandler{
				Param: getValueArray(&RequestType{
					RequestType: &GetBlockByNumberRequestType{},
				}),
				Func: reflect.ValueOf(HandlerService.GetBlockByNumberHandler),
			},
		},
		{
			Method: "eth_blockNumber",
			Handler: &RequestHandler{
				Param: getValueArray(&RequestType{
					RequestType: &EmptyType{},
				}),
				Func: reflect.ValueOf(HandlerService.GetBlockNumberHandler),
			},
		},
		{
			Method: "eth_call",
			Handler: &RequestHandler{
				Param: getValueArray(&RequestType{
					RequestType: &CallRequestType{},
				}),
				Func: reflect.ValueOf(HandlerService.CallHandler),
			},
		},
		{
			Method: "eth_estimateGas",
			Handler: &RequestHandler{
				Param: getValueArray(&RequestType{
					RequestType: &EstimateGasRequestType{},
				}),
				Func: reflect.ValueOf(HandlerService.EstimateGasHandler),
			},
		},
		{
			Method: "eth_getTransactionReceipt",
			Handler: &RequestHandler{
				Param: getValueArray(&RequestType{
					RequestType: &GetTransactionReceiptRequestType{},
				}),
				Func: reflect.ValueOf(HandlerService.GetTransactionReceiptHandler),
			},
		},
		{
			Method: "eth_sendRawTransaction",
			Handler: &RequestHandler{
				Param: getValueArray(&RequestType{
					RequestType: &SendRawTransactionRequestType{},
				}),
				Func: reflect.ValueOf(HandlerService.SendRawTransactionHandler),
			},
		},
		{
			Method: "eth_gasPrice",
			Handler: &RequestHandler{
				Param: getValueArray(&RequestType{
					RequestType: &GasPriceRequestType{},
				}),
				Func: reflect.ValueOf(HandlerService.GasPriceHandler),
			},
		},
		{
			Method: "eth_chainId",
			Handler: &RequestHandler{
				Param: getValueArray(&RequestType{
					RequestType: &EmptyType{},
				}),
				Func: reflect.ValueOf(HandlerService.ChainIdHandler),
			},
		},
		{
			Method: "eth_feeHistory",
			Handler: &RequestHandler{
				Param: getValueArray(&RequestType{
					RequestType: &FeeHistoryRequestType{},
				}),
				Func: reflect.ValueOf(HandlerService.FeeHistoryHandler),
			},
		},
		{
			Method: "net_version",
			Handler: &RequestHandler{
				Param: getValueArray(&RequestType{
					RequestType: &EmptyType{},
				}),
				Func: reflect.ValueOf(HandlerService.NetVersionHandler),
			},
		},
		{
			Method: "eth_accounts",
			Handler: &RequestHandler{
				Param: getValueArray(&RequestType{
					RequestType: &EmptyType{},
				}),
				Func: reflect.ValueOf(HandlerService.GetAccountsHandler),
			},
		},
		{
			Method: "net_listening",
			Handler: &RequestHandler{
				Param: getValueArray(&RequestType{
					RequestType: &EmptyType{},
				}),
				Func: reflect.ValueOf(HandlerService.NetListeningHandler),
			},
		},
	}
}
