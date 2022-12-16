package rpc

import (
	"errors"
	"reflect"
)

var (
	ErrNonPublic          = errors.New("registered non-public service")
	ErrNoAvailable        = errors.New("no service is available, or provide service is not open")
	ErrCrc32              = errors.New("checksumIEEE error")
	ErrSerialization404   = errors.New("serialization 404")
	ErrCompressor404      = errors.New("compressor 404")
	ErrTimeout            = errors.New("time out")
	ErrCircuitBreaker     = errors.New("circuit breaker")
	ErrInput              = errors.New("input error")
	ErrMethodNameRegister = errors.New("register method name error")
	ErrAuth               = errors.New("auth error")
	ErrAuthLevel          = errors.New("insufficient auth level")
	ErrMethodName         = errors.New("method name error")
)

func isErrorType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Implements(reflect.TypeOf((*error)(nil)).Elem())
}
