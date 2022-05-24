package rpc

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strings"
)

// Server struct
type Server struct {
	addr      string
	methodMap map[string]*methodType
	options   *Options
}

// NewServer creates a new server
func NewServer(addr string, options ...Option) *Server {
	server := &Server{
		addr:      addr,
		methodMap: map[string]*methodType{},
		options:   defaultOptions(),
	}

	for _, fn := range options {
		fn(server.options)
	}

	return server
}

func (s *Server) Register(obj interface{}, name string, level int, cacheTime int) error {
	objType := reflect.TypeOf(obj)
	for objType.Kind() == reflect.Ptr {
		objType = objType.Elem()
	}
	switch objType.Kind() {
	case reflect.Func:
		s.registerMethod(obj, name, level, cacheTime)
	case reflect.Struct:
		s.registerStruct(obj, name, level, cacheTime)
	default:
		return ErrInput
	}
	return nil
}

func (s *Server) registerMethod(method interface{}, name string, level int, cacheTime int) error {
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
		authLevel: level,
		cacheTime: cacheTime,
	}
	return nil
}

func (s *Server) registerStruct(obj interface{}, name string, level int, cacheTime int) error {
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
			authLevel: level,
			cacheTime: cacheTime,
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
	requestLevel := s.options.Auth.Level(reqToken)
	return requestLevel >= method.authLevel, nil
}

// Run server
func (s *Server) Run() {
	mux := http.NewServeMux()
	for k := range s.methodMap {
		mux.HandleFunc("/"+k+"/", func(w http.ResponseWriter, r *http.Request) {
			setupCORS(&w)
			log.Print(k)
			in, key, err := s.getInArgs(k, r)
			if err != nil {
				log.Print(err.Error())
				w.Write([]byte(err.Error()))
				return
			}
			cache, ok := s.checkCache(key)
			if ok {
				w.Write(cache)
				return
			}
			method := s.methodMap[k]
			resp, err := method.Call(in)
			if err != nil {
				log.Print(err.Error())
				w.Write([]byte(err.Error()))
			} else {
				s.cache(key, resp, k)
				w.Write(resp)
			}
		})
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		// The "/" pattern matches everything, so we need to check
		// that we're at the root here.
		if req.URL.Path != "/" {
			http.NotFound(w, req)
			return
		}
		fmt.Fprintf(w, "Welcome to the home page!")
	})
	log.Fatal(http.ListenAndServe(s.addr, mux))
}

func setupCORS(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

func (s *Server) getInArgs(methodName string, r *http.Request) ([]reflect.Value, string, error) {
	var requestData map[string]interface{}
	body, _ := ioutil.ReadAll(r.Body)
	err := json.Unmarshal(body, &requestData)
	log.Print(requestData)
	if err != nil {
		log.Print(err.Error())
		return nil, "", err
	}
	ok, err := s.Verify(fmt.Sprintf("%v", requestData["Auth"]), methodName)
	if err != nil {
		log.Print(err.Error())
		return nil, "", err
	}
	if !ok {
		err = ErrAuth
		log.Print(err.Error())
		return nil, "", err
	}
	inArgs := requestData["InArgs"]
	log.Printf("%v", inArgs)
	if reflect.TypeOf(inArgs).Kind() != reflect.Slice && reflect.TypeOf(inArgs).Kind() != reflect.Array {
		err = ErrInput
		log.Print(err.Error())
		return nil, "", err
	}
	inArgsValue := reflect.ValueOf(inArgs)
	in := make([]reflect.Value, inArgsValue.Len())
	for i := range in {
		in[i] = inArgsValue.Index(i).Elem()
	}
	return in, methodName + "_" + fmt.Sprintf("%v", inArgs), nil
}

func (s *Server) checkCache(key string) ([]byte, bool) {
	log.Print(key)
	val, err := s.options.cache.Get([]byte(key))
	if err != nil {
		log.Print(err.Error())
		return nil, false
	}
	return val, true
}

func (s *Server) cache(key string, val []byte, methodName string) bool {
	log.Print("cache")
	err := s.options.cache.Set([]byte(key), val, s.methodMap[methodName].cacheTime)
	if err != nil {
		log.Print(err.Error())
		return false
	}
	return true
}
