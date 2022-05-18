package handlers

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/TopiaNetwork/topia/api/web3/types"
	hexutil "github.com/TopiaNetwork/topia/api/web3/types/hexutil"
	secp "github.com/TopiaNetwork/topia/crypt/secp256"
	tpcrtypes "github.com/TopiaNetwork/topia/crypt/types"
	"github.com/TopiaNetwork/topia/currency"
	txbasic "github.com/TopiaNetwork/topia/transaction/basic"
	txuni "github.com/TopiaNetwork/topia/transaction/universal"
	"math/big"
	"reflect"
	"strconv"
)

type HandlerService interface {
	CallHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{}
	ChainIdHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{}
	EstimateGasHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{}
	FeeHistoryHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{}
	GasPriceHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{}
	GetAccountsHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{}
	GetBalanceHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{}
	GetBlockByHashHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{}
	GetBlockByNumberHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{}
	GetBlockNumberHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{}
	GetCodeHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{}
	GetTransactionByHashHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{}
	GetTransactionCountHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{}
	GetTransactionReceiptHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{}
	NetListeningHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{}
	NetVersionHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{}
	SendRawTransactionHandler(parmas interface{}, apiServant interface{}, txInterface interface{}) interface{}
}

type RequestHandler struct {
	Func   reflect.Value
	Param  interface{}
	Return interface{}
}

func (r *RequestHandler) Call(method string) {
	result := r.Func.Call(r.Param.([]reflect.Value))
	r.Return = result[0].Interface()
}

type Api struct {
	Method  string
	Handler *RequestHandler
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
func getValueArray(args interface{}) []reflect.Value {
	argsValue := reflect.ValueOf(args).Elem()
	argsType := reflect.TypeOf(args).Elem()
	argss := make([]reflect.Value, 0, argsType.NumField())
	for i := 0; i < argsValue.NumField(); i++ {
		argss = append(argss, argsValue.Field(i))
	}
	return argss
}

type RequestType struct {
	Handler     HandlerService
	RequestType interface{}
	ApiServant  interface{}
	TxInterface interface{}
}
type EmptyType struct{}

type CallRequestType struct {
	TranArgs    TransactionArgs
	BlockNumber interface{}
}
type CallResponseType struct {
	Balance string `json:"balance"`
}
type TransactionArgs struct {
	From                 *types.Address  `json:"from"`
	To                   *types.Address  `json:"to"`
	Gas                  *hexutil.Uint64 `json:"gas"`
	GasPrice             *hexutil.Big    `json:"gasPrice"`
	MaxFeePerGas         *hexutil.Big    `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *hexutil.Big    `json:"maxPriorityFeePerGas"`
	Value                *hexutil.Big    `json:"value"`
	Nonce                *hexutil.Uint64 `json:"nonce"`

	// We accept "data" and "input" for backwards-compatibility reasons.
	// "input" is the newer name and should be preferred by clients.
	// Issue detail: https://github.com/ethereum/go-ethereum/issues/15628
	Data  *hexutil.Bytes `json:"data"`
	Input *hexutil.Bytes `json:"input"`

	// Introduced by AccessListTxType transaction.
	AccessList *types.AccessList `json:"accessList,omitempty"`
	ChainID    *hexutil.Big      `json:"chainId,omitempty"`
}

type EstimateGasRequestType struct {
	TranArgs TransactionArgs
	Height   string
}
type EstimateGasResponseType struct {
	Balance string `json:"balance"`
}

type FeeHistoryRequestType struct {
	BlockCount  int
	BlockHeight string
	Percentile  []float64
}
type FeeHistoryResponseType struct {
	OldestBlock  *hexutil.Big
	Reward       [][]*hexutil.Big
	BaseFee      []*hexutil.Big
	GasUsedRatio []float64
}

type GasPriceRequestType struct {
	Address string
	Height  string
}
type GasPriceResponseType struct {
	Balance string `json:"balance"`
}

type GetBalanceRequestType struct {
	Address string
	Height  interface{}
}
type GetBalanceResponseType struct {
	Balance string `json:"balance"`
}

type GetBlockByHashRequestType struct {
	BlockHash string
	Tx        bool
}
type GetBlockResponseType struct {
	Number           *hexutil.Big   `json:"number"`
	Hash             types.Hash     `json:"hash"`
	ParentHash       types.Hash     `json:"parentHash"`
	Nonce            []byte         `json:"nonce"`
	MixHash          types.Hash     `json:"mixHash"`
	Sha3Uncles       types.Hash     `json:"sha3Uncles"`
	LogsBloom        []byte         `json:"logsBloom"`
	StateRoot        types.Hash     `json:"stateRoot"`
	Miner            types.Address  `json:"miner"`
	Difficulty       *hexutil.Big   `json:"difficulty"`
	ExtraData        []byte         `json:"extraData"`
	Size             hexutil.Uint64 `json:"size"`
	GasLimit         hexutil.Uint64 `json:"gasLimit"`
	GasUsed          hexutil.Uint64 `json:"gasUsed"`
	Timestamp        hexutil.Uint64 `json:"timestamp"`
	TransactionsRoot types.Hash     `json:"transactionsRoot"`
	ReceiptsRoot     types.Hash     `json:"receiptsRoot"`
	BaseFeePerGas    *hexutil.Big   `json:"baseFeePerGas"`
	Transactions     []interface{}  `json:"transactions"`
	Uncles           []types.Hash   `json:"uncles"`
	TotalDifficulty  *hexutil.Big   `json:"totalDifficulty"`
}

type GetBlockByNumberRequestType struct {
	Height interface{}
	Tx     bool
}

type GetCodeRequestType struct {
	Address string
	Height  interface{}
}

type EthRpcTransaction struct {
	BlockHash        types.Hash      `json:"blockHash"`
	BlockNumber      *hexutil.Big    `json:"blockNumber"`
	From             types.Address   `json:"from"`
	Gas              hexutil.Uint64  `json:"gas"`
	GasPrice         *hexutil.Big    `json:"gasPrice"`
	GasFeeCap        *hexutil.Big    `json:"maxFeePerGas,omitempty"`
	GasTipCap        *hexutil.Big    `json:"maxPriorityFeePerGas,omitempty"`
	Hash             types.Hash      `json:"hash"`
	Input            hexutil.Bytes   `json:"input"`
	Nonce            hexutil.Uint64  `json:"nonce"`
	To               types.Address   `json:"to"`
	TransactionIndex *hexutil.Uint64 `json:"transactionIndex"`
	Value            *hexutil.Big    `json:"value"`
	Type             hexutil.Uint64  `json:"type"`
	Accesses         string          `json:"accessList,omitempty"`
	ChainID          *hexutil.Big    `json:"chainId,omitempty"`
	V                *hexutil.Big    `json:"v"`
	R                *hexutil.Big    `json:"r"`
	S                *hexutil.Big    `json:"s"`
}

type GetTransactionByHashRequestType struct {
	TxHash string
}
type GetTransactionByhashResponseType struct {
	BlockHash        []byte   `json:"blockHash"`
	BlockNumber      uint64   `json:"blockNumber"`
	From             []byte   `json:"from"`
	Gas              uint64   `json:"gas"`
	GasPrice         uint64   `json:"gasPrice"`
	GasFeeCap        big.Int  `json:"maxFeePerGas,omitempty"`
	GasTipCap        big.Int  `json:"maxPriorityFeePerGas,omitempty"`
	Hash             []byte   `json:"hash"`
	Input            []byte   `json:"input"`
	Nonce            uint64   `json:"nonce"`
	To               []byte   `json:"to"`
	TransactionIndex uint64   `json:"transactionIndex"`
	Value            *big.Int `json:"value"`
	Type             uint64   `json:"type"`
	Accesses         []byte   `json:"accessList,omitempty"`
	ChainID          *big.Int `json:"chainId,omitempty"`
	V                *big.Int `json:"v"`
	R                *big.Int `json:"r"`
	S                *big.Int `json:"s"`
}

type GetTransactionCountRequestType struct {
	Address types.Address
	Height  string
}

type GetTransactionReceiptRequestType struct {
	TxHash types.Hash
}
type GetTransactionReceiptResponseType struct {
	BlockHash         string      `json:"blockHash"`
	BlockNumber       string      `json:"blockNumber"`
	TransactionHash   string      `json:"transactionHash"`
	TransactionIndex  string      `json:"transactionIndex"`
	From              string      `json:"from"`
	To                interface{} `json:"to"`
	GasUsed           string      `json:"gasUsed"`
	CumulativeGasUsed string      `json:"cumulativeGasUsed"`
	ContractAddress   string      `json:"contractAddress"`
	Logs              string      `json:"logs"`
	LogsBloom         string      `json:"logsBloom"`
	Type              string      `json:"type"`
	EffectiveGasPrice string      `json:"effectiveGasPrice"`
	Status            string      `json:"status"`
}

type SendRawTransactionRequestType struct {
	RawTransaction hexutil.Bytes
}
type SendRawTransactionResponseType struct{}

func ConstructTopiaTransaction(rawTransaction SendRawTransactionRequestType) (*txbasic.Transaction, error) {
	tx := new(types.Transaction)
	err := tx.UnmarshalBinary(rawTransaction.RawTransaction)
	if err != nil {
		return nil, errors.New("unmarshal rawTransaction error")
	}

	V, R, S := tx.RawSignatureValues()
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	v := ComputeV(V, tx.Type(), tx.ChainId())
	if v.BitLen() == 0 {
		sig[64] = uint8(0)
	} else {
		sig[64] = v.Bytes()[0]
	}

	var c secp.CryptServiceSecp256
	sigHash := tx.Hash().Bytes()
	pubKey, err := c.RecoverPublicKey(sigHash, sig)
	if err != nil {
		return nil, errors.New("recover PubKey error")
	}

	var from types.Address
	copy(from[:], types.Keccak256(pubKey[1:])[12:])

	var txType int
	if tx.To() == nil && tx.Data() != nil {
		txType = createContract
	} else if tx.To() != nil && len(tx.Data()) == 0 {
		txType = transfer
	} else if tx.To() != nil && len(tx.Data()) != 0 {
		txType = invokeContract
	} else {
		txType = unknown
	}

	switch txType {
	case createContract:
		transactionUniversalHead := txuni.TransactionUniversalHead{
			Version:           txbasic.Transaction_Topia_Universal_V1,
			FeePayer:          from.Bytes(),
			GasPrice:          tx.GasPrice().Uint64(),
			GasLimit:          tx.Gas(),
			Type:              uint32(txuni.TransactionUniversalType_ContractDeploy),
			FeePayerSignature: []byte(rawTransaction.RawTransaction),
		}
		txData, _ := json.Marshal(tx.Data())
		txUni := txuni.TransactionUniversal{
			Head: &transactionUniversalHead,
			Data: &txuni.TransactionUniversalData{
				Specification: txData,
			},
		}

		transactionHead := txbasic.TransactionHead{
			Category:  []byte(txbasic.TransactionCategory_Eth),
			ChainID:   types.Null,
			Version:   uint32(txbasic.Transaction_Eth_V1),
			FromAddr:  from.Bytes(),
			Nonce:     tx.Nonce(),
			Signature: []byte(rawTransaction.RawTransaction),
		}
		txDataBytes, _ := json.Marshal(txUni)
		tx := txbasic.Transaction{
			Head: &transactionHead,
			Data: &txbasic.TransactionData{
				Specification: txDataBytes,
			},
		}
		return &tx, nil
	case transfer:
		transactionHead := txbasic.TransactionHead{
			Category: []byte(txbasic.TransactionCategory_Eth),
			ChainID:  types.Null,
			Version:  uint32(txbasic.Transaction_Eth_V1),
			FromAddr: from.Bytes(),
			Nonce:    tx.Nonce(),
		}
		transactionUniversalHead := txuni.TransactionUniversalHead{
			Version:  txbasic.Transaction_Topia_Universal_V1,
			FeePayer: from.Bytes(),
			GasPrice: tx.GasPrice().Uint64(),
			GasLimit: tx.Gas(),
			Type:     uint32(txuni.TransactionUniversalType_Transfer),
		}
		targets := []txuni.TargetItem{
			{
				currency.TokenSymbol_Native,
				tx.Value(),
			},
		}
		txfer := txuni.TransactionUniversalTransfer{
			TransactionHead:          transactionHead,
			TransactionUniversalHead: transactionUniversalHead,
			TargetAddr:               tpcrtypes.Address(hex.EncodeToString(tx.To().Bytes())),
			Targets:                  targets,
		}

		txferData, _ := json.Marshal(txfer)

		txUni := txuni.TransactionUniversal{
			Head: &txfer.TransactionUniversalHead,
			Data: &txuni.TransactionUniversalData{
				Specification: txferData,
			},
		}
		txDataBytes, _ := json.Marshal(txUni)
		tx := txbasic.Transaction{
			Head: &transactionHead,
			Data: &txbasic.TransactionData{
				Specification: txDataBytes,
			},
		}
		return &tx, nil
	case invokeContract:
		transactionUniversalHead := txuni.TransactionUniversalHead{
			Version:           txbasic.Transaction_Topia_Universal_V1,
			FeePayer:          from.Bytes(),
			GasPrice:          tx.GasPrice().Uint64(),
			GasLimit:          tx.Gas(),
			Type:              uint32(txuni.TransactionUniversalType_ContractInvoke),
			FeePayerSignature: []byte(rawTransaction.RawTransaction),
		}
		txData, _ := json.Marshal(tx.Data())
		txUni := txuni.TransactionUniversal{
			Head: &transactionUniversalHead,
			Data: &txuni.TransactionUniversalData{
				Specification: txData,
			},
		}

		transactionHead := txbasic.TransactionHead{
			Category: []byte(txbasic.TransactionCategory_Eth),
			ChainID:  types.Null,
			Version:  uint32(txbasic.Transaction_Eth_V1),
			FromAddr: from.Bytes(),
			Nonce:    tx.Nonce(),
		}
		txDataBytes, _ := json.Marshal(txUni)
		tx := txbasic.Transaction{
			Head: &transactionHead,
			Data: &txbasic.TransactionData{
				Specification: txDataBytes,
			},
		}
		return &tx, nil
	case unknown:
	default:
		return nil, nil
	}
	return nil, nil
}
func ComputeV(v *big.Int, txType byte, chainId *big.Int) *big.Int {
	switch txType {
	case types.LegacyTxType:
		if chainId.BitLen() == 0 {
			v = new(big.Int).Sub(v, big.NewInt(27))
		} else {
			v = new(big.Int).Sub(v, big.NewInt(35))
			v = new(big.Int).Sub(v, new(big.Int).Mul(big.NewInt(2), chainId))
		}
	case types.AccessListTxType:
	case types.DynamicFeeTxType:
	}
	return v
}
func ConstructTransaction(call CallRequestType) *txbasic.Transaction {
	var gasPrice, gasLimit uint64
	if call.TranArgs.Gas == nil {
		gasLimit = 0
	} else {
		gasLimit, _ = strconv.ParseUint(call.TranArgs.Gas.String(), 10, 64)
	}
	if call.TranArgs.GasPrice == nil {
		gasPrice = 0
	} else {
		gasPrice, _ = strconv.ParseUint(call.TranArgs.GasPrice.String(), 10, 64)
	}

	var from []byte
	if call.TranArgs.From != nil {
		from = call.TranArgs.From.Bytes()
	} else {
		from = []byte{}
	}
	transactionUniversalHead := txuni.TransactionUniversalHead{
		Version:  txbasic.Transaction_Topia_Universal_V1,
		FeePayer: from,
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Type:     uint32(txuni.TransactionUniversalType_ContractInvoke),
	}
	var txData []byte
	if call.TranArgs.Data != nil {
		txData, _ = json.Marshal(call.TranArgs.Data)
	} else {
		txData, _ = json.Marshal(call.TranArgs.Input)
	}
	txUni := txuni.TransactionUniversal{
		Head: &transactionUniversalHead,
		Data: &txuni.TransactionUniversalData{
			Specification: txData,
		},
	}

	transactionHead := txbasic.TransactionHead{
		Category: []byte(txbasic.TransactionCategory_Eth),
		ChainID:  types.Null,
		Version:  uint32(txbasic.Transaction_Eth_V1),
		FromAddr: from,
	}
	txDataBytes, _ := json.Marshal(txUni)
	return &txbasic.Transaction{
		Head: &transactionHead,
		Data: &txbasic.TransactionData{
			Specification: txDataBytes,
		},
	}
}
func ConstructGasTransaction(call EstimateGasRequestType) *txbasic.Transaction {
	var from []byte
	if call.TranArgs.From != nil {
		from = call.TranArgs.From.Bytes()
	} else {
		from = []byte{}
	}
	transactionHead := txbasic.TransactionHead{
		Category: []byte(txbasic.TransactionCategory_Eth),
		ChainID:  types.Null,
		Version:  uint32(txbasic.Transaction_Eth_V1),
		FromAddr: from,
	}
	var gasPrice, gasLimit uint64
	if call.TranArgs.Gas == nil {
		gasLimit = 0
	} else {
		gasLimit, _ = strconv.ParseUint(call.TranArgs.Gas.String(), 10, 64)
	}
	if call.TranArgs.GasPrice == nil {
		gasPrice = 0
	} else {
		gasPrice, _ = strconv.ParseUint(call.TranArgs.GasPrice.String(), 10, 64)
	}

	transactionUniversalHead := txuni.TransactionUniversalHead{
		Version:  txbasic.Transaction_Topia_Universal_V1,
		FeePayer: from,
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		Type:     uint32(txuni.TransactionUniversalType_Transfer),
	}
	var value *big.Int
	if call.TranArgs.Value != nil {
		value = call.TranArgs.Value.ToInt()
	} else {
		value = big.NewInt(0)
	}
	targets := []txuni.TargetItem{
		{
			currency.TokenSymbol_Native,
			value,
		},
	}
	var to string
	if call.TranArgs.To != nil {
		to = call.TranArgs.To.String()
	} else {
		to = ""
	}
	txfer := txuni.TransactionUniversalTransfer{
		TransactionHead:          transactionHead,
		TransactionUniversalHead: transactionUniversalHead,
		TargetAddr:               tpcrtypes.Address(to),
		Targets:                  targets,
	}

	txferData, _ := json.Marshal(txfer)

	txUni := txuni.TransactionUniversal{
		Head: &txfer.TransactionUniversalHead,
		Data: &txuni.TransactionUniversalData{
			Specification: txferData,
		},
	}
	txDataBytes, _ := json.Marshal(txUni)
	return &txbasic.Transaction{
		Data: &txbasic.TransactionData{
			Specification: txDataBytes,
		},
	}
}
