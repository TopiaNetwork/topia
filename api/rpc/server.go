package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"

	tlog "github.com/TopiaNetwork/topia/log"
	logcomm "github.com/TopiaNetwork/topia/log/common"
)

// Server struct
type Server struct {
	addr      string
	methodMap map[string]*methodType
	*Options
	logger tlog.Logger
}

// NewServer creates a new server
func NewServer(addr string, options ...Option) *Server {
	mylog, err := tlog.CreateMainLogger(logcomm.DebugLevel, tlog.JSONFormat, tlog.StdErrOutput, "")
	if err != nil {
		panic(err)
	}
	server := &Server{
		addr:      addr,
		methodMap: map[string]*methodType{},
		Options:   defaultOptions(),
		logger:    mylog,
	}

	for _, fn := range options {
		fn(server.Options)
	}

	return server
}

func (s *Server) Register(obj interface{}, name string, authority byte, cacheTime int, timeout time.Duration) error {
	objType := reflect.TypeOf(obj)
	for objType.Kind() == reflect.Ptr {
		objType = objType.Elem()
	}
	switch objType.Kind() {
	case reflect.Func:
		return s.registerMethod(obj, name, authority, cacheTime, timeout)
	case reflect.Struct:
		return s.registerStruct(obj, name, authority, cacheTime, timeout)
	default:
		return ErrInput
	}
}

func (s *Server) registerMethod(method interface{}, name string, authority byte, cacheTime int, timeout time.Duration) error {
	mType := reflect.TypeOf(method)
	if mType.Kind() != reflect.Func {
		return errors.New("input method is not func")
	}

	if strings.Trim(name, " ") == "" && mType.Name() == "" {
		return ErrMethodNameRegister
	}
	if strings.Trim(name, " ") == "" {
		name = mType.Name()
	}

	var errPos int
	if mType.NumOut() == 0 { // method have no return
		errPos = -1
	} else {
		var lastRetIndex = mType.NumOut() - 1
		if isErrorType(mType.Out(lastRetIndex)) { // judge if the last return value of func is error-type
			errPos = lastRetIndex
		} else {
			errPos = -1
		}
	}

	s.methodMap[name] = &methodType{
		isObjMethod: false,
		mValue:      reflect.ValueOf(method),
		mType:       mType,
		errPos:      errPos,
		authLevel:   authority,
		cacheTime:   cacheTime,
		timeout:     timeout,
	}
	return nil
}

func (s *Server) registerStruct(obj interface{}, name string, authority byte, cacheTime int, timeout time.Duration) error {
	typ := reflect.TypeOf(obj)
	objValue := reflect.ValueOf(obj)
	DoneRegister := 0
	for idx := 0; idx < typ.NumMethod(); idx++ {
		method := typ.Method(idx)
		mName := method.Name
		if !method.IsExported() {
			continue
		}
		if strings.Trim(name, " ") == "" {
			name = typ.Elem().Name()
		}

		var errPos int
		if method.Type.NumOut() == 0 { // method have no return
			errPos = -1
		} else {
			var lastRetIndex = method.Type.NumOut() - 1
			if isErrorType(method.Type.Out(lastRetIndex)) { // judge if the last return value of func is error-type
				errPos = lastRetIndex
			} else {
				errPos = -1
			}
		}

		mName = name + "_" + mName
		s.methodMap[mName] = &methodType{
			isObjMethod: true,
			receiver:    objValue,
			mValue:      method.Func,
			mType:       method.Type,
			errPos:      errPos,
			authLevel:   authority,
			cacheTime:   cacheTime,
			timeout:     timeout,
		}
		DoneRegister++
	}
	if DoneRegister == 0 {
		return ErrNoAvailable
	}

	return nil
}

func (s *Server) Verify(reqToken string, methodName string) (bool, error) {
	method, ok := s.methodMap[methodName]
	if !ok {
		err := ErrMethodName
		return false, err
	}
	requestLevel := s.auth.Level(reqToken)
	return requestLevel >= method.authLevel, nil
}

// Run server
func (s *Server) Start() {
	mux := http.NewServeMux()
	// if s's websocket is on, register websocket handler
	if s.upgrader != nil {
		mux.HandleFunc("/websocket", func(w http.ResponseWriter, r *http.Request) {
			remoteAddr := r.RemoteAddr
			conn, err := s.upgrader.Upgrade(w, r, nil)

			s.logger.Info(remoteAddr + " is connected")

			if err != nil {
				s.logger.Error(err.Error())
				w.Write([]byte(err.Error()))
				return
			}
			val, ok := s.websocketServers.Load(remoteAddr)
			if ok {
				ws := val.(*WebsocketServer)
				ws.mu.Lock()
				ws.conn = conn
				ws.isConnected = true
				ws.mu.Unlock()
				s.logger.Info("reconnect")
			} else {
				conn.SetPingHandler(func(string) error {
					log.Print("receive ping")
					return nil
				})
				ws := &WebsocketServer{
					remoteAddr:     remoteAddr,
					conn:           conn,
					writeWait:      10 * time.Second,
					pongWait:       10 * time.Second,
					pingPeriod:     2 * time.Second,
					maxMessageSize: 512,
					isConnected:    true,
				}
				s.websocketServers.Store(remoteAddr, ws)

				go s.run()
				go ws.readPump()
				go ws.writePump()
				// go s.handleReceivedMessage(ws)
				// go s.handleSendMessage(ws)
				// go s.handleMessage()
			}
		})

	}
	// register http handler for every method
	for k := range s.methodMap {
		method := s.methodMap[k]
		var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			setupCORS(&w)
			// decode message
			message, err := IODecodeMessage(r.Body)
			if err != nil {
				s.logger.Error("Illegal request err: " + err.Error())
				return
			}
			resp, err := s.handleMessage(message)
			if err != nil {
				s.logger.Error("handleMessage err: " + err.Error())
				w.Write([]byte(err.Error()))
				return
			}
			w.Write(resp)
		})
		if method.timeout > 0 {
			handler = http.TimeoutHandler(handler, method.timeout, "")
		}
		mux.Handle("/"+k+"/", handler)
	}

	// handle server not found
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		// The "/" pattern matches everything, so we need to check
		// that we're at the root here.
		if req.URL.Path != "/" {
			http.NotFound(w, req)
			return
		}
	})

	s.logger.Error("http.ListenAndServe err: " + http.ListenAndServe(s.addr, mux).Error())
}

func (s *Server) run() {
	for {
		s.Options.websocketServers.Range(func(key, value interface{}) bool {
			ws := value.(*WebsocketServer)
			select {
			case data := <-ws.receive:
				go func() {
					message, err := DecodeMessage(data)
					if err != nil {
						s.logger.Error("DecodeMessage err: " + err.Error())
					}
					resp, err := s.handleMessage(message)
					if err != nil {
						s.logger.Error("handleMessage err: " + err.Error())
					}
					s.logger.Info("response:" + message.RequestId)
					ws.send <- resp
				}()
			default:
			}
			return true
		})
	}
}

// func (s *Server) handleSendMessage(ws *WebsocketServer) {
// 	for {
// 		message := <-ws.send
// 		err := ws.WriteMessage(1, message)
// 		if err != nil {
// 			continue
// 		}
// 	}
// }

// func (s *Server) handleReceivedMessage(ws *WebsocketServer) {
// 	timeSleep := time.Duration(0)
// 	timeIncrease := 500 * time.Microsecond
// 	for {
// 		_, message, err := ws.ReadMessage()
// 		if err != nil {
// 			timeSleep := timeSleep + timeIncrease
// 			time.Sleep(timeSleep)
// 			continue
// 		}
// 		timeSleep = time.Duration(0)
// 		go func() {
// 			message, err := DecodeMessage(message)
// 			if err != nil {
// 				s.logger.Error(err.Error())
// 			}
// 			resp, err := s.handleMessage(message)
// 			if err != nil {
// 				s.logger.Error(err.Error())
// 			}
// 			s.logger.Info("response:" + message.RequestId)
// 			ws.send <- resp
// 		}()
// 	}
// }

func (s *Server) handleMessage(message *Message) (resp []byte, err error) {
	// verify
	ok, err := s.Verify(message.AuthCode, message.MethodName)
	if err != nil {
		s.logger.Error("Verify err: " + err.Error())
		return EncodeMessage(message.RequestId, message.MethodName, message.AuthCode,
			&ErrMsg{
				Errtype:   ErrMethodNotFound,
				ErrString: err.Error(),
			},
			nil)
	}
	if !ok {
		err = ErrAuth
		s.logger.Error(err.Error())
		return EncodeMessage(message.RequestId, message.MethodName, message.AuthCode,
			&ErrMsg{
				Errtype:   ErrAuthFailed,
				ErrString: err.Error(),
			},
			nil)
	}
	// convert payload to []reflect.Value
	in, err := s.getInArgs(message.Payload)
	if err != nil {
		s.logger.Error("getInArgs err: " + err.Error())
		return EncodeMessage(message.RequestId, message.MethodName, message.AuthCode,
			&ErrMsg{
				Errtype:   ErrIllegalArgument,
				ErrString: err.Error(),
			},
			nil)
	}
	// check cache
	key := message.MethodName + "_" + fmt.Sprintf("%v", in)
	cache, ok := s.checkCache(key)
	if ok {
		return EncodeMessage(message.RequestId, message.MethodName, message.AuthCode, &ErrMsg{}, cache)
	}
	var callErr = &ErrMsg{} // hold ErrMsg for method returned, default empty
	// call method
	method := s.methodMap[message.MethodName]
	resp, err = method.Call(in, callErr)
	if err != nil {
		s.logger.Error("Call method err: " + err.Error())
		return EncodeMessage(message.RequestId, message.MethodName, message.AuthCode,
			&ErrMsg{
				Errtype:   ErrIllegalArgument,
				ErrString: err.Error(),
			}, nil)
	}

	if callErr.Errtype == NoErr {
		s.setCache(key, resp, message.MethodName)
	}
	return EncodeMessage(message.RequestId, message.MethodName, message.AuthCode, callErr, resp)
}

func setupCORS(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

func (s *Server) getInArgs(payload []byte) ([]reflect.Value, error) {

	var inArgs []interface{}

	if len(payload) == 0 { // no input args
		return nil, nil
	}

	err := json.Unmarshal(payload, &inArgs)
	if err != nil {
		s.logger.Error("Unmarshal err: " + err.Error())
		return nil, err
	}

	s.logger.Infof("%v", inArgs)
	if reflect.TypeOf(inArgs).Kind() != reflect.Slice && reflect.TypeOf(inArgs).Kind() != reflect.Array {
		err = ErrInput
		s.logger.Error("ErrInput :" + err.Error())
		return nil, err
	}
	inArgsValue := reflect.ValueOf(inArgs)
	in := make([]reflect.Value, inArgsValue.Len())
	for i := range in {
		in[i] = inArgsValue.Index(i).Elem()
	}
	return in, nil
}

func (s *Server) checkCache(key string) ([]byte, bool) {
	val, err := s.cc.Get([]byte(key))
	if err != nil {
		s.logger.Info("try to get from cache err: " + err.Error())
		return nil, false
	}
	return val, true
}

func (s *Server) setCache(key string, val []byte, methodName string) bool {
	err := s.cc.Set([]byte(key), val, s.methodMap[methodName].cacheTime)
	if err != nil {
		s.logger.Error("set cache err: " + err.Error())
		return false
	}
	return true
}
