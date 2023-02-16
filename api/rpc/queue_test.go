package rpc

import (
	"github.com/TopiaNetwork/topia/eventhub"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEnqueue(t *testing.T) {
	q := newQueue()
	msg := eventhub.EventMsg{
		Name: "test_event",
		Data: "test_event_msg",
	}
	msg2 := eventhub.EventMsg{
		Name: "test_event2",
		Data: "test_event_msg2",
	}

	err := q.enqueue(msg)
	assert.Nil(t, err)

	err = q.enqueue(msg2)
	assert.Nil(t, err)
}

func TestEnqueueOverflow(t *testing.T) {
	q := newQueue()
	msg := eventhub.EventMsg{
		Name: "test_event",
		Data: "test_event_msg",
	}

	for i := 0; i < MaxSubQueueLength; i++ {
		err := q.enqueue(msg)
		assert.Nil(t, err)
	}

	err := q.enqueue(msg)
	assert.NotNil(t, err)
}

func TestGetMsgFromQueue(t *testing.T) {
	q := newQueue()
	msg := eventhub.EventMsg{
		Name: "test_event",
		Data: "test_event_msg",
	}

	err := q.enqueue(msg)
	assert.Nil(t, err)

	getMsg := <-q.wait()
	assert.Equal(t, msg, getMsg)
}

func TestClearQueue(t *testing.T) {
	q := newQueue()
	msg := eventhub.EventMsg{
		Name: "test_event",
		Data: "test_event_msg",
	}

	err := q.enqueue(msg)
	assert.Nil(t, err)

	q.clear()
	assert.Equal(t, 0, q.size)
}

func TestRemoveHead(t *testing.T) {
	q := newQueue()
	msg := eventhub.EventMsg{
		Name: "test_event",
		Data: "test_event_msg",
	}
	msg2 := eventhub.EventMsg{
		Name: "test_event2",
		Data: "test_event_msg2",
	}

	err := q.enqueue(msg)
	assert.Nil(t, err)

	err = q.enqueue(msg2)
	assert.Nil(t, err)

	q.removeHead()
	assert.Equal(t, msg2, q.head.data)
}
