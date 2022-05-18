package web3

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TopiaNetwork/topia/api/servant"
	"github.com/TopiaNetwork/topia/api/service"
	"github.com/TopiaNetwork/topia/api/web3/handlers"
	types "github.com/TopiaNetwork/topia/api/web3/types"
	tplog "github.com/TopiaNetwork/topia/log"
	tplogcmm "github.com/TopiaNetwork/topia/log/common"
	"net/http"
	"reflect"
	"time"
)

func HandlerWapper(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

func StartServer(config Web3ServerConfiguration, apiServant servant.APIServant, txInterface types.TxInterface) {
	w3Server := InitWeb3Server(config, apiServant, txInterface)

	addr := config.Host + ":" + config.Port
	srv := &http.Server{
		Addr:         addr,
		Handler:      nil,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	timeoutWeb3Handler := http.TimeoutHandler(HandlerWapper(w3Server.ServeHttp), 60*time.Second, "")
	http.Handle("/", timeoutWeb3Handler)
	//srv.ListenAndServeTLS("cert.pem", "key.pem")
	srv.ListenAndServe()
}

func (w *Web3Server) ServeHttp(res http.ResponseWriter, req *http.Request) {
	reqBodyMsg, err := readJson(req)
	req.Body.Close()

	if len(req.Header) > w.MaxHeader && w.MaxHeader != 0 || len(reqBodyMsg.String()) > w.MaxBody && w.MaxBody != 0 {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	log, _ := tplog.CreateMainLogger(tplogcmm.DebugLevel, tplog.JSONFormat, tplog.StdErrOutput, "")
	log.Info(reqBodyMsg.Method)
	var result *types.JsonrpcMessage
	if err != nil {
		result = types.ErrorMessage(&types.InvalidRequestError{err.Error()})
	} else if req.Method != "POST" {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	} else {
		method := reqBodyMsg.Method
		if _, exist := w.methodApiMap[method]; exist {
			var Handler handlers.HandlerService
			Handler = &service.Handler{}
			w.methodApiMap[method].Param.([]reflect.Value)[0].Set(reflect.ValueOf(Handler))
			err = JsonParamsToStruct(reqBodyMsg.Params, w.methodApiMap[method].Param.([]reflect.Value)[1])
			if err != nil {
				result = types.ErrorMessage(&types.InvalidParamsError{err.Error()})
			} else {
				handler := w.methodApiMap[method]
				w.methodApiMap[method].Param.([]reflect.Value)[2].Set(reflect.ValueOf(w.apiServant))
				w.methodApiMap[method].Param.([]reflect.Value)[3].Set(reflect.ValueOf(w.txInterface))
				handler.Call(method)
				result = w.methodApiMap[method].Return.(*types.JsonrpcMessage)
			}
		} else {
			result = types.ErrorMessage(&types.MethodNotFoundError{})
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
	txInterface  types.TxInterface
	apiServant   servant.APIServant
	methodApiMap map[string]*handlers.RequestHandler
	MaxHeader    int
	MaxBody      int
}
type Web3ServerConfiguration struct {
	MaxHeader int
	MaxBody   int
	Port      string
	Host      string
}

func (w *Web3Server) RegisterMethodHandler() {
	w.methodApiMap = make(map[string]*handlers.RequestHandler)
	apis := handlers.GetApis()
	for _, value := range apis {
		w.methodApiMap[value.Method] = value.Handler
	}
}

func InitWeb3Server(config Web3ServerConfiguration, apiServant servant.APIServant, txInterface types.TxInterface) *Web3Server {
	w3Server := new(Web3Server)
	w3Server.MaxHeader = config.MaxHeader
	w3Server.MaxBody = config.MaxBody
	w3Server.apiServant = apiServant
	w3Server.txInterface = txInterface
	w3Server.RegisterMethodHandler()
	return w3Server
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

func getValueArray(args interface{}) []reflect.Value {
	argsValue := reflect.ValueOf(args).Elem()
	argsType := reflect.TypeOf(args).Elem()
	argss := make([]reflect.Value, 0, argsType.NumField())
	for i := 0; i < argsValue.NumField(); i++ {
		argss = append(argss, argsValue.Field(i))
	}
	return argss
}
