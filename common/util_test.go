package common

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
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

type TestStruct struct {
	a int
	b string
}

func TestErrGroup(t *testing.T) {
	var tsArray = []*TestStruct{
		&TestStruct{
			5,
			"test1",
		},
		&TestStruct{
			6,
			"test2",
		},
		&TestStruct{
			7,
			"test3",
		},
		&TestStruct{
			8,
			"test4",
		},
		&TestStruct{
			9,
			"test5",
		},
	}

	var eg errgroup.Group
	for _, ts := range tsArray {
		ts := ts
		eg.Go(func() error {
			fmt.Printf("%v\n", ts)
			return nil
		})
	}
	eg.Wait()

}

type TestStru struct {
	id  string
	val int
}

var TSVal = []*TestStru{
	&TestStru{"test1", 10},
	&TestStru{"test2", 9},
	&TestStru{"test3", 8},
	&TestStru{"test4", 7},
	&TestStru{"test5", 6},
}

func StrComp(ts1 *TestStru, ts2 *TestStru) bool {
	return ts1.id == ts2.id
}

func TestStringEqual(t *testing.T) {
	index := sort.Search(len(TSVal), func(i int) bool {
		return TSVal[i].val == 10
	})

	fmt.Println(index)

}
