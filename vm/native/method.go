package native

import "reflect"

type nativeContractMethod struct {
	receiver            reflect.Value
	method              reflect.Value
	paramTypes          []reflect.Type
	resultDataOutSIndex int
	resultErrorOutIndex int
}
