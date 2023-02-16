package filter

import (
	"github.com/TopiaNetwork/topia/eventhub"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewFilter(t *testing.T) {
	type testData struct {
		testField1 string
		testField2 []byte
		testField3 uint64
		testField4 int
		testField5 float64
	}

	evMsgForRegi := eventhub.EventMsg{
		Name: "testEvent1",
		Data: testData{},
	}

	registerEventMsgs(evMsgForRegi)

	f, err := NewFilter("testEvent1", "testField1 = 'test' and testField2 = 'test2' and testField3 >= 100")
	assert.Nil(t, err)

	evMsg1 := eventhub.EventMsg{
		Name: "testEvent1",
		Data: testData{
			testField1: "test",
			testField2: []byte("test2"),
			testField3: 101,
		},
	}
	pass := f.Pass(evMsg1)
	assert.True(t, pass)
}

func TestIllegalFilterString(t *testing.T) {
	type testData struct {
		testField1 string
		testField2 []byte
		testField3 uint64
		testField4 int
		testField5 float64
	}

	evMsgForRegi := eventhub.EventMsg{
		Name: "testEvent1",
		Data: testData{},
	}

	registerEventMsgs(evMsgForRegi)

	filterStrs := []string{
		"notExistField = 0",
		"testField1",
		"testField1 > 10",
		"testField3 contains something",
	}

	for _, str := range filterStrs {
		_, err := NewFilter("testEvent1", str)
		assert.NotNil(t, err)
	}

}

func TestSomeLegalFilterStringExamples(t *testing.T) {
	type testData struct {
		testField1 string
		testField2 []byte
		testField3 uint64
		testField4 int
		testField5 float64
	}

	evMsgForRegi := eventhub.EventMsg{
		Name: "testEvent1",
		Data: &testData{},
	}

	registerEventMsgs(evMsgForRegi)

	legalExamples := []string{
		"",
		"testField1='something'",
		"testField2 Contains something and testField3=111 or testField4 = 100",
	}

	for _, str := range legalExamples {
		_, err := NewFilter("testEvent1", str)
		assert.Nil(t, err)
	}
}

func TestSomePassExamples(t *testing.T) {
	type testData struct {
		testField1 string
		testField2 []byte
		testField3 uint64
		testField4 int
		testField5 float64
	}

	evMsgForRegi := eventhub.EventMsg{
		Name: "testEvent1",
		Data: &testData{},
	}

	registerEventMsgs(evMsgForRegi)

	examples := []struct {
		filterStr string
		evMsg     eventhub.EventMsg
	}{
		{
			filterStr: "testField1 contains thing",
			evMsg: eventhub.EventMsg{
				Name: "testEvent1",
				Data: &testData{
					testField1: "something",
				},
			},
		},
		{
			filterStr: "testField1 CONTAINS thing",
			evMsg: eventhub.EventMsg{
				Name: "testEvent1",
				Data: &testData{
					testField1: "something",
				},
			},
		},
		{
			filterStr: "testField1 contains 'thing'",
			evMsg: eventhub.EventMsg{
				Name: "testEvent1",
				Data: &testData{
					testField1: "something",
				},
			},
		},
		{
			filterStr: "testField2 contains thing",
			evMsg: eventhub.EventMsg{
				Name: "testEvent1",
				Data: &testData{
					testField2: []byte("something"),
				},
			},
		},
		{
			filterStr: "testField3>=10 and testField4 < 20",
			evMsg: eventhub.EventMsg{
				Name: "testEvent1",
				Data: &testData{
					testField3: 10,
					testField4: 15,
				},
			},
		},
		{
			filterStr: "   testField3>=10 and testField4 < 20   ",
			evMsg: eventhub.EventMsg{
				Name: "testEvent1",
				Data: &testData{
					testField3: 10,
					testField4: 15,
				},
			},
		},
		{
			filterStr: "testField3>=10 and testField4 < 20 or testField1 contains ^%#",
			evMsg: eventhub.EventMsg{
				Name: "testEvent1",
				Data: &testData{
					testField1: "@#$^%#$#&",
					testField3: 1,
					testField4: 15,
				},
			},
		},
		{
			filterStr: "testField1 Exists",
			evMsg: eventhub.EventMsg{
				Name: "testEvent1",
				Data: &testData{
					testField1: "hi",
				},
			},
		},
	}

	for _, example := range examples {
		f, err := NewFilter("testEvent1", example.filterStr)
		assert.Nil(t, err)

		pass := f.Pass(example.evMsg)
		assert.True(t, pass)
	}
}
