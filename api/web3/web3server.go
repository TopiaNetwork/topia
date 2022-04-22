package web3

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/web3/handlers"
	"github.com/TopiaNetwork/topia/api/web3/types"
	"net/http"
	"reflect"
	"time"
)

//跨域访问
func cors(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")                                // 允许访问所有域，可以换成具体url，注意仅具体url才能带cookie信息
		w.Header().Add("Access-Control-Allow-Headers", "*")                               //header的类型
		w.Header().Add("Access-Control-Allow-Credentials", "true")                        //设置为true，允许ajax异步请求带cookie信息
		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE") //允许请求方法
		w.Header().Set("content-type", "application/json;charset=UTF-8")                  //返回数据格式是json
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		f(w, r)
	}
}

func StartServer() {
	w3Server := InitWeb3Server()

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      nil,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	timeoutWeb3Handler := http.TimeoutHandler(cors(w3Server.Web3Handler), 60*time.Second, "timeout")
	http.Handle("/", timeoutWeb3Handler)
	srv.ListenAndServe()
}

/*
	1.跨域访问	√
	2.超时处理	√
	3.错误码		√
*/
func (w *Web3Server) Web3Handler(res http.ResponseWriter, req *http.Request) {
	//解码body,获取method
	reqBodyMsg, _ := readJson(req)
	//匹配body.method
	method := reqBodyMsg.Method
	//解析参数
	err := JsonParamsToStruct(reqBodyMsg.Params, w.methodApiMap[method].Params)
	if err != nil {
		//返回值里面写入错误信息
		//错误码+错误信息
		errResp, _ := json.Marshal(types.ErrorResponse(err))
		res.Write(errResp)
	}
	//调用对应handler
	ch := make(chan struct{})
	resultCh := make(chan interface{})
	go func() {
		result := w.methodApiMap[method].Call()
		ch <- struct{}{} //超时处理
		resultCh <- result
	}()
	//返回结果
	select {
	case <-ch:
		result := <-resultCh
		re := constructResult(result, *reqBodyMsg)
		res.Write(re)
	case <-req.Context().Done():
		res.WriteHeader(http.StatusRequestTimeout)
		//TODO:写入错误原因的返回值
	}
}

type Web3Server struct {
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
		Params:     &types.SendRawTransactionRequestType{},
		Handler:    handlers.SendRawTransactionHandler,
		ApiServant: w.apiServant,
	}
	w.methodApiMap["eth_sendRawTransaction"] = sendRawTransactionHandler

	gasPriceHandler := &RequestHandler{
		Params:     &types.GasPriceRequestType{},
		Handler:    handlers.GasPriceHandler,
		ApiServant: w.apiServant,
	}
	w.methodApiMap["eth_gasPrice"] = gasPriceHandler
}

func InitWeb3Server() *Web3Server {
	w3Server := new(Web3Server)
	w3Server.apiServant = servant.NewAPIServant()
	w3Server.RegisterMethodHandler()
	return w3Server
}

type ReqHandler func(interface{}, interface{}) interface{}

type RequestHandler struct {
	Params     interface{} //注册时候， 通过new实际的参数结构赋给该字段
	Handler    ReqHandler
	ApiServant servant.APIServant
}

func (r RequestHandler) Call() interface{} {
	//根据给定的参数和方法，调用对应的handler
	result := r.Handler(r.Params, r.ApiServant)
	return result
}

func readJson(req *http.Request) (*types.JsonrpcMessage, error) {
	dec := json.NewDecoder(req.Body)
	reqBodyMsg := new(types.JsonrpcMessage)
	err := dec.Decode(reqBodyMsg)
	if err != nil {
		return nil, err
	}
	return reqBodyMsg, err
}

func JsonParamsToStruct(msg json.RawMessage, data interface{}) error {
	//1.直接解析json到data
	err := json.Unmarshal(msg, data) //直接解析失败，尝试逐个解析
	if err != nil {
		//2.使用反射解析到data中
		var r interface{}
		err = json.Unmarshal(msg, &r) //这个解析不会出错

		//遍历data进行装配
		dataValue := reflect.ValueOf(data).Elem()
		dataType := reflect.TypeOf(data).Elem()
		//检查r是否是数组
		//默认json params数据和data的file是对齐的
		if value, ok := r.([]interface{}); ok {
			if len(r.([]interface{})) > dataType.NumField() {
				return fmt.Errorf("too many arguments, want at most %d", dataType.NumField())
			}
			//遍历数组的每一项，并尝试将其装配至data
			for index, v := range value {
				dataItem := reflect.New(dataType.Field(index).Type) //获取对应的类型实例

				ms, _ := json.Marshal(v)                       //获取原生的Json.RawMessage
				err = json.Unmarshal(ms, dataItem.Interface()) //反序列化，装配

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
		//解析失败
	}
	return result
}
