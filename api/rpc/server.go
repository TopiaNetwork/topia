package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/net/netutil"
	"log"
	"net"
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
	logger      tlog.Logger
	httpServer  *http.Server
	shutDownSig chan struct{} // signal to shut down the server
}

// NewServer creates a new server
func NewServer(addr string, options ...Option) *Server {
	mylog, err := tlog.CreateMainLogger(logcomm.DebugLevel, tlog.JSONFormat, tlog.StdErrOutput, "")
	if err != nil {
		panic(err)
	}
	server := &Server{
		addr:        addr,
		methodMap:   map[string]*methodType{},
		Options:     defaultOptions(),
		logger:      mylog,
		shutDownSig: make(chan struct{}, 1),
	}

	for _, fn := range options {
		fn(server.Options)
	}

	return server
}

func (s *Server) Register(obj interface{}, name string, authority byte, cacheAble bool, cacheTime int, timeout time.Duration) error {
	objType := reflect.TypeOf(obj)
	for objType.Kind() == reflect.Ptr {
		objType = objType.Elem()
	}
	switch objType.Kind() {
	case reflect.Func:
		return s.registerMethod(obj, name, authority, cacheAble, cacheTime, timeout)
	case reflect.Struct:
		return s.registerStruct(obj, name, authority, cacheAble, cacheTime, timeout)
	default:
		return ErrInput
	}
}

func (s *Server) registerMethod(method interface{}, name string, authority byte, cacheAble bool, cacheTime int, timeout time.Duration) error {
	mType := reflect.TypeOf(method)

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
		cacheAble:   cacheAble,
		cacheTime:   cacheTime,
		timeout:     timeout,
	}
	return nil
}

func (s *Server) registerStruct(obj interface{}, name string, authority byte, cacheAble bool, cacheTime int, timeout time.Duration) error {
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
			cacheAble:   cacheAble,
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

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		s.logger.Errorf("listen on %v err: %v", s.addr, err)
		return
	}
	listener = netutil.LimitListener(listener, MaxSimultaneousTcpConn)

	mux := http.NewServeMux()
	// if s's websocket is on, register websocket handler
	if s.upgrader != nil {
		go s.runWsServer()

		mux.HandleFunc("/websocket", func(w http.ResponseWriter, r *http.Request) {

			remoteAddr := r.RemoteAddr
			conn, err := s.upgrader.Upgrade(w, r, nil)

			s.logger.Info(remoteAddr + " is connected")

			if err != nil {
				s.logger.Error(err.Error())
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
					send:           make(chan []byte),
					receive:        make(chan []byte),
				}
				s.websocketServers.Store(remoteAddr, ws)

				go ws.readPump()
				go ws.writePump()
				// go s.handleReceivedMessage(ws)
				// go s.handleSendMessage(ws)
				// go s.handleMessage()

			}
		})

	}
	// register http handler for every method
	for mName, method := range s.methodMap {
		var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			setupCORS(&w)

			r.Body = http.MaxBytesReader(w, r.Body, int64(MaxHttpBodyReaderBytes))

			// decode message
			message, err := IODecodeMessage(r.Body)
			if err != nil {
				s.logger.Error("Illegal request err: " + err.Error())
				return
			}
			resp, err := s.handleMessage(message)
			if err != nil {
				s.logger.Error("handleMessage err: " + err.Error())
				return
			}
			w.Write(resp)
		})
		if method.timeout > 0 {
			handler = http.TimeoutHandler(handler, method.timeout, "")
		}
		mux.Handle("/"+mName+"/", handler)

	}

	// shutDownHandler
	var shutDownHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setupCORS(&w)

		// decode message
		message, err := IODecodeMessage(r.Body)
		if err != nil {
			s.logger.Error("Illegal request err: " + err.Error())
			return
		}

		if message.AuthCode != s.auth.tokenArr[Manager] {
			err = ErrAuth
			s.logger.Error(err.Error())
			resp, _ := EncodeMessage(message.RequestId, message.MethodName, message.AuthCode,
				&ErrMsg{
					Errtype:   ErrAuthFailed,
					ErrString: err.Error(),
				},
				nil)
			w.Write(resp)
			return
		}

		resp, _ := EncodeMessage(message.RequestId, message.MethodName, message.AuthCode,
			&ErrMsg{}, nil)
		w.Write(resp)

		s.shutDownSig <- struct{}{}

	})
	mux.Handle("/CloseServer/", shutDownHandler)

	// wait for the shutdown signal
	go func() {
		<-s.shutDownSig
		s.shutDown()
		return
	}()

	// handle server not found
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// The "/" pattern matches everything, so we need to check
		// that we're at the root here.
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
	})

	s.httpServer = &http.Server{
		Handler:           mux,
		MaxHeaderBytes:    MaxHttpHeaderBytes,
		WriteTimeout:      HttpWriteTimeout,
		ReadHeaderTimeout: HttpReadHeaderTimeout,
	}

	//err = httpServer.Serve(listener) // TODO
	//if err != nil {
	//	s.logger.Error("httpServer serve err: " + err.Error())
	//}

	err = s.httpServer.ServeTLS(
		listener,
		CertFilePath,
		KeyFilePath,
	)
	if err != nil {
		if err == http.ErrServerClosed {
			s.logger.Info(err.Error())
		} else {
			s.logger.Error("httpServer serve err: " + err.Error())
		}
	}

}

func (s *Server) shutDown() error {
	time.Sleep(1 * time.Second)
	if s.upgrader != nil {
		s.websocketServers.Range(func(remoteAddr, wsServer interface{}) bool {
			server := wsServer.(*WebsocketServer)
			err := server.conn.Close()
			if err != nil {
				s.logger.Error("websocket conn close err: " + err.Error())
			}

			return true
		})
	}
	return s.httpServer.Shutdown(context.Background())
}

// runWsServer receive msg from corresponding receiveChan of all servers and process them
func (s *Server) runWsServer() {
	for {
		s.Options.websocketServers.Range(func(key, value interface{}) bool {
			ws := value.(*WebsocketServer)

			select {
			case data, ok := <-ws.receive:
				if !ok {
					// TODO 对应的ws receive chan已经关闭, 删掉对应的ws server
					return true
				}
				go func() {
					message, err := DecodeMessage(data) // TODO 这里的message可能是call 也可能是订阅
					if err != nil {
						s.logger.Error("DecodeMessage err: " + err.Error())
						return
					}
					resp, err := s.handleMessage(message)
					if err != nil {
						s.logger.Error("handleMessage err: " + err.Error())
						return
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
	val, err := s.cache.Get([]byte(key))
	if err != nil {
		s.logger.Info("try to get from cache err: " + err.Error())
		return nil, false
	}
	return val, true
}

func (s *Server) setCache(key string, val []byte, methodName string) bool {
	if !s.methodMap[methodName].cacheAble {
		return false
	}

	err := s.cache.Set([]byte(key), val, s.methodMap[methodName].cacheTime)
	if err != nil {
		s.logger.Error("set cache err: " + err.Error())
		return false
	}
	return true
}
