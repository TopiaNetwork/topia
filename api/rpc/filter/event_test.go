package filter

import (
	"github.com/TopiaNetwork/topia/eventhub"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestIsValidFieldName(t *testing.T) {
	evMsg1 := eventhub.EventMsg{
		Name: "testEvent1",
		Data: struct {
			testField1 string
			testField2 []byte
			testField3 uint64
			testField4 int
			testField5 float64
		}{},
	}

	registerEventMsgs(evMsg1)

	kind, valid := isValidFieldName("testEvent1", "testField1")
	assert.True(t, valid)
	assert.Equal(t, reflect.String, kind)

	kind, valid = isValidFieldName("testEvent1", "testField2")
	assert.True(t, valid)
	assert.Equal(t, reflect.Slice, kind)

	kind, valid = isValidFieldName("testEvent1", "testField3")
	assert.True(t, valid)
	assert.Equal(t, reflect.Uint64, kind)

	kind, valid = isValidFieldName("testEvent1", "testField4")
	assert.True(t, valid)
	assert.Equal(t, reflect.Int, kind)

	kind, valid = isValidFieldName("testEvent1", "testField5")
	assert.True(t, valid)
	assert.Equal(t, reflect.Float64, kind)
}
