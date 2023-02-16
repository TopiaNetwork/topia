package filter

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestConditionPass(t *testing.T) {
	type msgData struct {
		TestField int
	}

	data := msgData{TestField: 333}

	cond := condition{
		field:    "testfield",
		operator: Eq,
		data:     333,
		dataType: reflect.Int,
	}

	pass := cond.pass(reflect.TypeOf(data), reflect.ValueOf(data))
	assert.True(t, pass)

	data = msgData{TestField: 444}
	cond = condition{
		field:    "testfield",
		operator: Gt,
		data:     333,
		dataType: reflect.Int,
	}

	pass = cond.pass(reflect.TypeOf(data), reflect.ValueOf(data))
	assert.True(t, pass)
}

func TestConditionPassInnerStruct(t *testing.T) {
	type msgData struct {
		someStruct struct {
			testField int
		}
	}

	data := msgData{someStruct: struct{ testField int }{testField: 333}}

	cond := condition{
		field:    "testfield",
		operator: Eq,
		data:     333,
		dataType: reflect.Int,
	}

	pass := cond.pass(reflect.TypeOf(data), reflect.ValueOf(data))
	assert.True(t, pass)

	data2 := msgData{someStruct: struct{ testField int }{testField: 444}}
	pass = cond.pass(reflect.TypeOf(data2), reflect.ValueOf(data2))
	assert.False(t, pass)
}
