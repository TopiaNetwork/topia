package web3

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TopiaNetwork/topia/api/servant"
	types2 "github.com/TopiaNetwork/topia/api/web3/eth/types"
	handlers "github.com/TopiaNetwork/topia/api/web3/handlers"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"net/http"
	"reflect"
	"runtime/debug"
	"time"
)

func HandlerWapper(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if pr := recover(); pr != nil {
				fmt.Println("panic recover: %v", pr)
				debug.PrintStack()
			}
		}()

		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
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

func StartServer(config Web3ServerConfiguration, apiServant servant.APIServant) {
	w3Server := InitWeb3Server(config, apiServant)
	timeoutWeb3Handler := http.TimeoutHandler(HandlerWapper(w3Server.ServeHttp), 60*time.Second, "")
	http.Handle("/", timeoutWeb3Handler)

	httpAddr := config.HttpHost + ":" + config.HttpPort
	srv := &http.Server{
		Addr:         httpAddr,
		Handler:      nil,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	HttpsAddr := config.HttpsHost + ":" + config.HttpsPost
	srvs := &http.Server{
		Addr:         HttpsAddr,
		Handler:      nil,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go srvs.ListenAndServeTLS("../cert.pem", "../key.pem")
	srv.ListenAndServe()
}

func (w *Web3Server) ServeHttp(res http.ResponseWriter, req *http.Request) {
	reqBodyMsg, err := readJson(req)
	if req != nil {
		defer req.Body.Close()
	}

	if len(req.Header) > w.MaxHeader && w.MaxHeader != 0 || len(reqBodyMsg.String()) > w.MaxBody && w.MaxBody != 0 {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	log, _ := tplog.CreateMainLogger(tplogcmm.DebugLevel, tplog.JSONFormat, tplog.StdErrOutput, "")
	log.Info(reqBodyMsg.Method)
	var result *types2.JsonrpcMessage
	if err != nil {
		result = types2.ErrorMessage(&types2.InvalidRequestError{err.Error()})
	} else if req.Method != "POST" {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	} else {
		method := reqBodyMsg.Method
		if _, exist := w.methodApiMap[method]; exist {
			var Handler handlers.HandlerService
			Handler = &handlers.Handler{
				w.apiServant,
			}
			w.methodApiMap[method].Param.([]reflect.Value)[0].Set(reflect.ValueOf(Handler))
			err = JsonParamsToStruct(reqBodyMsg.Params, w.methodApiMap[method].Param.([]reflect.Value)[1])
			if err != nil {
				result = types2.ErrorMessage(&types2.InvalidParamsError{err.Error()})
			} else {
				w.methodApiMap[method].Call(method)
				result = w.methodApiMap[method].Return.(*types2.JsonrpcMessage)
			}
		} else {
			result = types2.ErrorMessage(&types2.MethodNotFoundError{})
		}
	}
	if result.Result == nil && result.Error == nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	re := constructResult(result, *reqBodyMsg)
	res.Write(re)
}

type Web3Server struct {
	apiServant   servant.APIServant
	methodApiMap map[string]*handlers.RequestHandler
	MaxHeader    int
	MaxBody      int
}

type Web3ServerConfiguration struct {
	MaxHeader int
	MaxBody   int
	HttpPort  string
	HttpHost  string
	HttpsPost string
	HttpsHost string
}

func (w *Web3Server) RegisterMethodHandler() {
	w.methodApiMap = make(map[string]*handlers.RequestHandler)
	apis := handlers.GetApis()
	for _, value := range apis {
		w.methodApiMap[value.Method] = value.Handler
	}
}

func InitWeb3Server(config Web3ServerConfiguration, apiServant servant.APIServant) *Web3Server {
	w3Server := new(Web3Server)
	w3Server.MaxHeader = config.MaxHeader
	w3Server.MaxBody = config.MaxBody
	w3Server.apiServant = apiServant
	w3Server.RegisterMethodHandler()
	return w3Server
}

func readJson(req *http.Request) (*types2.JsonrpcMessage, error) {
	dec := json.NewDecoder(req.Body)
	reqBodyMsg := new(types2.JsonrpcMessage)
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

		args := data.(reflect.Value).Interface()
		dataValue := reflect.ValueOf(args).Elem()
		dataType := reflect.TypeOf(args).Elem()

		if value, ok := r.([]interface{}); ok {
			if len(r.([]interface{})) > dataType.NumField() {
				return fmt.Errorf("too many arguments, want at most %d", dataType.NumField())
			}
			for index, v := range value {
				dataItem := reflect.New(dataType.Field(index).Type)
				ms, _ := json.Marshal(v)
				err = json.Unmarshal(ms, dataItem.Interface())
				if err != nil {
					return fmt.Errorf("invalid argument %d: %v", index, err)
				}
				if dataItem.IsNil() && dataItem.Field(index).Kind() != reflect.Ptr {
					return fmt.Errorf("missing value for required argument %d", index)
				}
				dataValue.Field(index).Set(dataItem.Elem())
			}
		} else {
			return errors.New("non-array args")
		}
	}
	return nil
}

func constructResult(answer interface{}, msg types2.JsonrpcMessage) []byte {
	re := answer.(*types2.JsonrpcMessage)
	re.Version = msg.Version
	re.ID = msg.ID
	result, err := json.Marshal(re)
	if err != nil {
		return nil
	}
	return result
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
