package common

import (
	"golang.org/x/net/context"
	"reflect"

	"github.com/TopiaNetwork/topia/state"
)

type VMContext struct {
	context.Context
	state.CompositionState
	NodeID       string
	ContractName string
	Method       string
	Args         []reflect.Value
}

type VMResult struct {
	Code   ReturnCode
	ErrMsg string
	Data   interface{}
}
