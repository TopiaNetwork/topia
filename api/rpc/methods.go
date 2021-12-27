package rpc

import (
	tplog "github.com/TopiaNetwork/topia/log"
	"net/http"
	"reflect"
	"strings"
)

const (
	MethodPerm_Read  byte = 0x01
	MethodPerm_Write byte = 0x02
	MethodPerm_Sign  byte = 0x04
)

type RPCMethod struct {
	method   reflect.Value  // rpc method
	args     []reflect.Type // function arg
	argNames []string       // name of each argument
	rtns     []reflect.Type // return arg
	ws       bool           // websocket only
	perm     byte           // permission for method
	cache    bool           // allow the RPC response can be cached by the proxy cache server
}

func NewRPCFunc(f interface{}, args string, perm byte, cache bool) *RPCMethod {
	return newRPCFunc(f, args, false, perm, cache)
}

func NewWSRPCFunc(f interface{}, args string, perm byte) *RPCMethod {
	return newRPCFunc(f, args, true, perm, false)
}

func newRPCFunc(f interface{}, args string, ws bool, perm byte, c bool) *RPCMethod {
	var argNames []string
	if args != "" {
		argNames = strings.Split(args, ",")
	}
	return &RPCMethod{
		method:   reflect.ValueOf(f),
		args:     methodArgTypes(f),
		rtns:     methodReturnTypes(f),
		argNames: argNames,
		ws:       ws,
		perm:     perm,
		cache:    c,
	}
}

func methodArgTypes(f interface{}) []reflect.Type {
	t := reflect.TypeOf(f)
	n := t.NumIn()
	typez := make([]reflect.Type, n)
	for i := 0; i < n; i++ {
		typez[i] = t.In(i)
	}
	return typez
}

func methodReturnTypes(f interface{}) []reflect.Type {
	t := reflect.TypeOf(f)
	n := t.NumOut()
	typez := make([]reflect.Type, n)
	for i := 0; i < n; i++ {
		typez[i] = t.Out(i)
	}
	return typez
}

func makeHTTPHandler(rpcMethod *RPCMethod, log tplog.Logger) func(http.ResponseWriter, *http.Request) {
	panic("implement me")
}

func makeJSONRPCHandler(funcMap map[string]*RPCMethod, log tplog.Logger) http.HandlerFunc {
	panic("implement me")
}

func RegisterRPCMethods(mux *http.ServeMux, methodMap map[string]*RPCMethod, log tplog.Logger) {
	// HTTP endpoints
	for methodName, methodFunc := range methodMap {
		mux.HandleFunc("/"+methodName, makeHTTPHandler(methodFunc, log))
	}

	// JSONRPC endpoints
	mux.HandleFunc("/", makeJSONRPCHandler(methodMap, log))
}
