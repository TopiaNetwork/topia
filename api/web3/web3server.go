package web3

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3/handlers"
	types "github.com/TopiaNetwork/topia/api/web3/types"
	"net/http"
	"reflect"
	"time"
)

func cors(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Headers", "*")
		w.Header().Add("Access-Control-Allow-Credentials", "true")
		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("content-type", "application/json;charset=UTF-8")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		f(w, r)
	}
}

func StartServer(apiServant servant.APIServant, txInterface types.TxInterface) {
	w3Server := InitWeb3Server(apiServant, txInterface)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      nil,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	timeoutWeb3Handler := http.TimeoutHandler(cors(w3Server.Web3Handler), 60*time.Second, "timeout")
	http.Handle("/", timeoutWeb3Handler)
	//srv.ListenAndServeTLS("cert.pem", "key.pem")
	srv.ListenAndServe()
}

func (w *Web3Server) Web3Handler(res http.ResponseWriter, req *http.Request) {
	resultCh := make(chan *types.JsonrpcMessage, 1)
	ch := make(chan interface{}, 1)
	reqBodyMsg, err := readJson(req)

	go func() {
		var result *types.JsonrpcMessage
		if err != nil {
			result = types.ErrorMessage(&types.InvalidRequestError{err.Error()})
		} else {
			method := reqBodyMsg.Method
			if _, exist := w.methodApiMap[method]; exist {
				err = JsonParamsToStruct(reqBodyMsg.Params, w.methodApiMap[method].Params)
				if err != nil {
					result = types.ErrorMessage(&types.InvalidParamsError{err.Error()})
				} else {
					handler := w.methodApiMap[method]
					result = handler.Call(method).(*types.JsonrpcMessage)
				}
			} else {
				result = types.ErrorMessage(&types.MethodNotFoundError{})
			}
		}
		ch <- struct{}{}
		resultCh <- result
	}()

	select {
	case <-ch:
		result := <-resultCh
		re := constructResult(result, *reqBodyMsg)
		res.Write(re)
	case <-req.Context().Done():
		res.WriteHeader(http.StatusRequestTimeout)
	}
}

type Web3Server struct {
	txInterface  types.TxInterface
	apiServant   servant.APIServant
	methodApiMap map[string]*RequestHandler
}

func (w *Web3Server) RegisterMethodHandler() {
	w.methodApiMap = make(map[string]*RequestHandler)

	getBalanceHandler := &RequestHandler{
		Params:     &types.GetBalanceRequestType{},
		Handler:    handlers.GetBalanceHandler,
		ApiServant: w.apiServant,
	}
	w.methodApiMap["eth_getBalance"] = getBalanceHandler

	getTransactionByHashHandler := &RequestHandler{
		Params:     &types.GetTransactionByHashRequestType{},
		Handler:    handlers.GetTransactionByHashHandler,
		ApiServant: w.apiServant,
	}
	w.methodApiMap["eth_getTransactionByHash"] = getTransactionByHashHandler

	getTransactionCountHandler := &RequestHandler{
		Params:     &types.GetTransactionCountRequestType{},
		Handler:    handlers.GetTransactionCountHandler,
		ApiServant: w.apiServant,
	}
	w.methodApiMap["eth_getTransactionCount"] = getTransactionCountHandler

	getCodeHandler := &RequestHandler{
		Params:     &types.GetCodeRequestType{},
		Handler:    handlers.GetCodeHandler,
		ApiServant: w.apiServant,
	}
	w.methodApiMap["eth_getCode"] = getCodeHandler

	getBlockByHashHandler := &RequestHandler{
		Params:     &types.GetBlockByHashRequestType{},
		Handler:    handlers.GetBlockByHashHandler,
		ApiServant: w.apiServant,
	}
	w.methodApiMap["eth_getBlockByHash"] = getBlockByHashHandler

	getBlockByNumberHandler := &RequestHandler{
		Params:     &types.GetBlockByNumberRequestType{},
		Handler:    handlers.GetBlockByNumberHandler,
		ApiServant: w.apiServant,
	}
	w.methodApiMap["eth_getBlockByNumber"] = getBlockByNumberHandler

	getBlockNumberHandler := &RequestHandler{
		Params:     nil,
		Handler:    handlers.GetBlockNumberHandler,
		ApiServant: w.apiServant,
	}
	w.methodApiMap["eth_blockNumber"] = getBlockNumberHandler

	callHandler := &RequestHandler{
		Params:     &types.CallRequestType{},
		Handler:    handlers.CallHandler,
		ApiServant: w.apiServant,
	}
	w.methodApiMap["eth_call"] = callHandler

	estimateGasHandler := &RequestHandler{
		Params:     &types.EstimateGasRequestType{},
		Handler:    handlers.EstimateGasHandler,
		ApiServant: w.apiServant,
	}
	w.methodApiMap["eth_estimateGas"] = estimateGasHandler

	getTransactionReceipthandler := &RequestHandler{
		Params:     &types.GetTransactionReceiptRequestType{},
		Handler:    handlers.GetTransactionReceiptHandler,
		ApiServant: w.apiServant,
	}
	w.methodApiMap["eth_getTransactionReceipt"] = getTransactionReceipthandler

	sendRawTransactionHandler := &RequestHandler{
		Params:      &types.SendRawTransactionRequestType{},
		Handler:     handlers.SendRawTransactionHandler,
		ApiServant:  w.apiServant,
		TxInterface: w.txInterface,
	}
	w.methodApiMap["eth_sendRawTransaction"] = sendRawTransactionHandler

	gasPriceHandler := &RequestHandler{
		Params:     &types.GasPriceRequestType{},
		Handler:    handlers.GasPriceHandler,
		ApiServant: w.apiServant,
	}
	w.methodApiMap["eth_gasPrice"] = gasPriceHandler

	chainIdHandler := &RequestHandler{
		Params:     nil,
		Handler:    handlers.ChainIdHandler,
		ApiServant: w.apiServant,
	}
	w.methodApiMap["eth_chainId"] = chainIdHandler

	feeHistoryHandler := &RequestHandler{
		Params:     &types.FeeHistoryRequestType{},
		Handler:    handlers.FeeHistoryHandler,
		ApiServant: w.apiServant,
	}
	w.methodApiMap["eth_feeHistory"] = feeHistoryHandler

	netVersionHandler := &RequestHandler{
		Params:     nil,
		Handler:    handlers.NetVersionHandler,
		ApiServant: w.apiServant,
	}
	w.methodApiMap["net_version"] = netVersionHandler

	getAccountsHandler := &RequestHandler{
		Params:     nil,
		Handler:    handlers.GetAccountsHandler,
		ApiServant: w.apiServant,
	}
	w.methodApiMap["eth_accounts"] = getAccountsHandler

	netListeningHandler := &RequestHandler{
		Params:     nil,
		Handler:    handlers.NetListeningHandler,
		ApiServant: w.apiServant,
	}
	w.methodApiMap["net_listening"] = netListeningHandler
}

func InitWeb3Server(apiServant servant.APIServant, txInterface types.TxInterface) *Web3Server {
	w3Server := new(Web3Server)
	w3Server.apiServant = apiServant
	w3Server.txInterface = txInterface
	w3Server.RegisterMethodHandler()
	return w3Server
}

type ReqHandler func(interface{}, interface{}) interface{}

type RequestHandler struct {
	Params      interface{}
	Handler     ReqHandler
	ApiServant  servant.APIServant
	TxInterface types.TxInterface
}

func (r RequestHandler) Call(method string) interface{} {
	if method == "eth_sendRawTransaction" {
		result := r.Handler(r.Params, r.TxInterface)
		return result
	} else {
		result := r.Handler(r.Params, r.ApiServant)
		return result
	}
}

func readJson(req *http.Request) (*types.JsonrpcMessage, error) {
	dec := json.NewDecoder(req.Body)
	reqBodyMsg := new(types.JsonrpcMessage)
	err := dec.Decode(reqBodyMsg)
	return reqBodyMsg, err
}

func JsonParamsToStruct(msg json.RawMessage, data interface{}) error {
	err := json.Unmarshal(msg, data)
	if err != nil {
		var r interface{}
		err = json.Unmarshal(msg, &r)

		if len(r.([]interface{})) == 0 {
			return nil
		}

		dataValue := reflect.ValueOf(data).Elem()
		dataType := reflect.TypeOf(data).Elem()

		if value, ok := r.([]interface{}); ok {
			if len(r.([]interface{})) > dataType.NumField() {
				return fmt.Errorf("too many arguments, want at most %d", dataType.NumField())
			}
			for index, v := range value {
				dataItem := reflect.New(dataType.Field(index).Type)

				ms, _ := json.Marshal(v)
				err = json.Unmarshal(ms, dataItem.Interface())

				if err != nil {
					//参数解析出错
					return fmt.Errorf("invalid argument %d: %v", index, err)
				}
				if dataItem.IsNil() && dataItem.Field(index).Kind() != reflect.Ptr {
					return fmt.Errorf("missing value for required argument %d", index)
				}

				dataValue.Field(index).Set(dataItem.Elem())
			}
		} else {
			//json数组断言失败
			return errors.New("non-array args")
		}

	}
	return nil
}

func constructResult(answer interface{}, msg types.JsonrpcMessage) []byte {
	re := answer.(*types.JsonrpcMessage)
	re.Version = msg.Version
	re.ID = msg.ID
	result, err := json.Marshal(re)
	if err != nil {
		return nil
	}
	return result
}
