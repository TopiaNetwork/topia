package service

import (
	"context"
	"encoding/json"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3/eth"
	"github.com/TopiaNetwork/topia/api/web3/handlers"
	"github.com/TopiaNetwork/topia/api/web3/types"
	hexutil "github.com/TopiaNetwork/topia/api/web3/types/hexutil"
	"github.com/TopiaNetwork/topia/codec"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/currency"
	"github.com/TopiaNetwork/topia/transaction/universal"
	"math/big"
	"strconv"
)

type Handler struct {
}

func (h *Handler) CallHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{} {
	args := parmas.(*handlers.CallRequestType)
	servant := apiServant.(servant.APIServant)

	ctx := context.Background()
	tx := handlers.ConstructTransaction(*args)
	callResult, err := servant.ExecuteTxSim(ctx, tx)
	if err != nil {
		return types.ErrorMessage(err)
	}
	var transactionResultUniversal universal.TransactionResultUniversal
	marshaler := codec.CreateMarshaler(codec.CodecType_PROTO)
	_ = marshaler.Unmarshal(callResult.GetData().GetSpecification(), &transactionResultUniversal)
	//result := transactionResultUniversal.GetData()
	var re hexutil.Bytes
	re = callResult.GetData().GetSpecification()[2:]
	enc, err := json.Marshal(re)
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
func (h *Handler) ChainIdHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{} {
	servant := apiServant.(servant.APIServant)
	chainId, _ := new(big.Int).SetString(string(servant.ChainID()), 10)
	chId := (*hexutil.Big)(chainId)
	enc, err := json.Marshal(chId)
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
func (h *Handler) EstimateGasHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{} {
	args := parmas.(*handlers.EstimateGasRequestType)
	servant := apiServant.(servant.APIServant)

	tx := handlers.ConstructGasTransaction(*args)
	estimateGas, err := servant.EstimateGas(tx)
	if err != nil {
		return types.ErrorMessage(err)
	}
	gas := hexutil.Uint64(new(big.Int).SetBytes(estimateGas.Bytes()).Uint64())

	enc, err := json.Marshal(gas)
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
func (h *Handler) FeeHistoryHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{} {
	args := parmas.(*handlers.FeeHistoryRequestType)
	servant := apiServant.(servant.APIServant)

	feeHistory := eth.FeeHistory(args, servant)

	enc, err := json.Marshal(feeHistory)
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
func (h *Handler) GasPriceHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{} {
	gasPrice, err := eth.EstimateTipFee(apiServant.(servant.APIServant))
	if err != nil {
		return types.ErrorMessage(err)
	}

	enc, err := json.Marshal((*hexutil.Big)(gasPrice))
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
func (h *Handler) GetAccountsHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{} {
	var accounts []string
	accounts = append(accounts, "0x3F1B3C065aeE8cA34c47fa84aAC3024E95a2E6D9")

	enc, err := json.Marshal(accounts)
	if err != nil {
		//TODO:
		return nil //msg.ErrorResponse(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
func (h *Handler) GetBalanceHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{} {
	args := parmas.(*handlers.GetBalanceRequestType)
	servant := apiServant.(servant.APIServant)

	balance, err := servant.GetBalance(currency.TokenSymbol_Native, tpcrtypes.NewFromString(args.Address))
	if err != nil {
		return types.ErrorMessage(err)
	}
	var enc []byte
	if balance.BitLen() != 0 {
		enc, err = json.Marshal((*hexutil.Big)(balance))
	} else {
		enc, err = json.Marshal((*hexutil.Big)(big.NewInt(0)))
	}
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
func (h *Handler) GetBlockByHashHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{} {
	args := parmas.(*handlers.GetBlockByHashRequestType)
	servant := apiServant.(servant.APIServant)

	block, err := servant.GetBlockByHash(args.BlockHash)
	if err != nil {
		return types.ErrorMessage(err)
	}
	ethBlock := handlers.ConvertTopiaBlockToEthBlock(block)
	enc, err := json.Marshal(ethBlock)
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
func (h *Handler) GetBlockByNumberHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{} {
	args := parmas.(*handlers.GetBlockByNumberRequestType)
	servant := apiServant.(servant.APIServant)

	var height uint64
	number := args.Height
	if value, ok := number.(string); ok {
		height, _ = strconv.ParseUint(value[2:], 16, 64)
	} else if value, ok := number.(float64); ok {
		height = uint64(value)
	}

	block, err := servant.GetBlockByHeight(height)
	if err != nil {
		return types.ErrorMessage(err)
	}

	ethBlock := handlers.ConvertTopiaBlockToEthBlock(block)
	enc, err := json.Marshal(ethBlock)
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
func (h *Handler) GetBlockNumberHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{} {
	servant := apiServant.(servant.APIServant)

	block, err := servant.GetLatestBlock()
	if err != nil {
		return types.ErrorMessage(err)
	}
	height := block.GetHead().GetHeight()

	enc, err := json.Marshal(hexutil.Uint64(height))
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
func (h *Handler) GetCodeHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{} {
	args := parmas.(*handlers.GetCodeRequestType)
	servant := apiServant.(servant.APIServant)

	var height uint64
	number := args.Height
	if value, ok := number.(string); ok {
		height, _ = strconv.ParseUint(value[2:], 16, 64)
	} else if value, ok := number.(float64); ok {
		height = uint64(value)
	}

	code, err := servant.GetContractCode(tpcrtypes.NewFromString(args.Address), height)
	if err != nil {
		return types.ErrorMessage(err)
	}

	enc, err := json.Marshal(hexutil.Bytes(code))
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
func (h *Handler) GetTransactionByHashHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{} {
	args := parmas.(*handlers.GetTransactionByHashRequestType)
	servant := apiServant.(servant.APIServant)

	transaction, err := servant.GetTransactionByHash(args.TxHash)
	if err != nil {
		return types.ErrorMessage(err)
	}
	ethTransaction, err := handlers.TopiaTransactionToEthTransaction(*transaction, servant)
	if err != nil {
		return types.ErrorMessage(err)
	}
	enc, err := json.Marshal(ethTransaction)
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
func (h *Handler) GetTransactionCountHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{} {
	args := parmas.(*handlers.GetTransactionCountRequestType)
	servant := apiServant.(servant.APIServant)

	height, _ := strconv.ParseUint(args.Height, 10, 64)
	count, err := servant.GetTransactionCount(tpcrtypes.NewFromString(args.Address.String()), height)
	if err != nil {
		return types.ErrorMessage(err)
	}
	txCount := hexutil.Uint(count)

	enc, err := json.Marshal(txCount)
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
func (h *Handler) GetTransactionReceiptHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{} {
	args := parmas.(*handlers.GetTransactionReceiptRequestType)
	servant := apiServant.(servant.APIServant)
	result, err := servant.GetTransactionResultByHash(args.TxHash.Hex())
	if err != nil {
		return types.ErrorMessage(err)
	}
	ethReceipt := handlers.ConvertTopiaResultToEthReceipt(*result, servant)

	enc, err := json.Marshal(ethReceipt)
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
func (h *Handler) NetListeningHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{} {
	enc, err := json.Marshal(true)
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
func (h *Handler) NetVersionHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{} {
	//enc, err := json.Marshal(big.NewInt(9))
	enc, err := json.Marshal("9")

	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
func (h *Handler) SendRawTransactionHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{} {
	args := parmas.(*handlers.SendRawTransactionRequestType)
	txIn := txInterface.(types.TxInterface)
	tx, _ := handlers.ConstructTopiaTransaction(*args)
	ctx := context.Background()
	err := txIn.SendTransaction(ctx, tx)
	if err != nil {
		return types.ErrorMessage(err)
	}
	hashByte, _ := tx.HashBytes()
	var hash types.Hash
	hash.SetBytes(hashByte)

	enc, err := json.Marshal(hash)
	if err != nil {
		return types.ErrorMessage(err)
	}
	return &types.JsonrpcMessage{Result: enc}
}
