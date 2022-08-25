package rpc

import (
	"encoding/json"
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
		s.registerMethod(obj, name, authority, cacheTime, timeout)
	case reflect.Struct:
		s.registerStruct(obj, name, authority, cacheTime, timeout)
	default:
		return ErrInput
	}
	return nil
}

func (s *Server) registerMethod(method interface{}, name string, authority byte, cacheTime int, timeout time.Duration) error {
	mType := reflect.TypeOf(method)
	if strings.Trim(name, " ") == "" && mType.Name() == "" {
		return ErrMethodNameRegister
	}
	if strings.Trim(name, " ") == "" {
		name = mType.Name()
	}

	s.methodMap[name] = &methodType{
		mValue:    reflect.ValueOf(method),
		mType:     reflect.TypeOf(method),
		authLevel: authority,
		cacheTime: cacheTime,
		timeout:   timeout,
	}
	return nil
}

func (s *Server) registerStruct(obj interface{}, name string, authority byte, cacheTime int, timeout time.Duration) error {
	typ := reflect.TypeOf(obj)
	len := 0
	for idx := 0; idx < typ.NumMethod(); idx++ {
		method := typ.Method(idx)
		mName := method.Name
		if !IsPublic(mName) {
			continue
		}
		if strings.Trim(name, " ") == "" {
			name = typ.Elem().Name()
		}
		mName = name + "_" + mName
		s.methodMap[mName] = &methodType{
			mValue:    method.Func,
			mType:     method.Type,
			authLevel: authority,
			cacheTime: cacheTime,
			timeout:   timeout,
		}
		len++
	}
	if len == 0 {
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
	return requestLevel&method.authLevel > 0, nil
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
				s.logger.Error(err.Error())
				w.Write([]byte(err.Error()))
				return
			}
			resp, err := s.handleMessage(message)
			if err != nil {
				s.logger.Error(err.Error())
				w.Write([]byte(err.Error()))
				return
			}
			w.Write(resp)
		})
		if method.timeout > 0 {
			s.logger.Info("set timeout")
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

	s.logger.Error(http.ListenAndServe(s.addr, mux).Error())
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
						s.logger.Error(err.Error())
					}
					resp, err := s.handleMessage(message)
					if err != nil {
						s.logger.Error(err.Error())
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
		s.logger.Error(err.Error())
		return
	}
	if !ok {
		err = ErrAuth
		s.logger.Error(err.Error())
		return
	}
	// convert payload to []reflect.Value
	in, err := s.getInArgs(message.MethodName, message.Payload)
	if err != nil {
		s.logger.Error(err.Error())
		return
	}
	// check cache
	key := message.MethodName + "_" + fmt.Sprintf("%v", in)
	cache, ok := s.checkCache(key)
	if ok {
		return EncodeMessage(message.RequestId, message.MethodName, message.AuthCode, cache)
	}
	// call method
	method := s.methodMap[message.MethodName]
	resp, err = method.Call(in)
	if err != nil {
		s.logger.Error(err.Error())
	} else {
		s.cache(key, resp, message.MethodName)
	}
	resp, err = EncodeMessage(message.RequestId, message.MethodName, message.AuthCode, resp)
	return
}

func setupCORS(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

func (s *Server) getInArgs(methodName string, payload []byte) ([]reflect.Value, error) {

	var inArgs []interface{}
	err := json.Unmarshal(payload, &inArgs)
	if err != nil {
		s.logger.Error(err.Error())
		return nil, err
	}

	s.logger.Infof("%v", inArgs)
	if reflect.TypeOf(inArgs).Kind() != reflect.Slice && reflect.TypeOf(inArgs).Kind() != reflect.Array {
		err = ErrInput
		s.logger.Error(err.Error())
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
		s.logger.Error(err.Error())
		return nil, false
	}
	return val, true
}

func (s *Server) cache(key string, val []byte, methodName string) bool {
	s.logger.Info("cache")
	err := s.cc.Set([]byte(key), val, s.methodMap[methodName].cacheTime)
	if err != nil {
		s.logger.Error(err.Error())
		return false
	}
	return true
}
