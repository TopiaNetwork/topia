package common

import (
	"golang.org/x/net/context"
	"reflect"
)

type VMContext struct {
	context.Context
	VMServant
	NodeID       string
	ContractName string
	Method       string
	Args         []reflect.Value
}

type VMResult struct {
	Code    ReturnCode
	ErrMsg  string
	GasUsed uint64
	Data    interface{}
}
