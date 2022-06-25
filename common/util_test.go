package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestClone(t *testing.T) {
	b := []byte{0x01, 0x02, 0x05, 0x05, 0x07}
	a := new([]byte)

	err := Clone(a, b)

	t.Logf("%v, %v", a, err)
}

func TestAnyContain(t *testing.T) {
	type TestStru struct {
		a int
		b int
	}

	tsArray := []interface{}{
		&TestStru{10, 20},
		&TestStru{30, 10},
		&TestStru{10, 90},
	}

	isExist := IsContainItem(&TestStru{30, 10}, tsArray)

	assert.Equal(t, false, isExist)
}

func TestRemoveIfExistString(t *testing.T) {
	testString := []string{
		"16Uiu2HAm5pFAjWt8DBenfaqb6WRCCJniQK29dgsPrCsVuPvxXXMb",
		"16Uiu2HAm9AMMkvH9t8Q23NGtN8aGa6nHP7VETVnDRUPCT7SkRjbu",
		"16Uiu2HAmK8NLUBrHkXMQFJGr47JgyD4UeYfswgDyAoUv9vNvezH4",
	}

	rtnString := RemoveIfExistString("16Uiu2HAm9AMMkvH9t8Q23NGtN8aGa6nHP7VETVnDRUPCT7SkRjbu", testString)

	assert.Equal(t, 2, len(rtnString))
}
