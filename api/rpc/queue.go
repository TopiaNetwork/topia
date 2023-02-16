package rpc

import (
	"errors"
	"github.com/TopiaNetwork/topia/eventhub"
	"sync"
)

// Queue
type Queue struct {
	head   *node
	tail   *node
	size   int
	outBuf chan eventhub.EventMsg // buffer for outMsg, must initialize with buffer
	mu     sync.Mutex
}
type node struct {
	data eventhub.EventMsg
	next *node
}

func newQueue() *Queue {
	return &Queue{outBuf: make(chan eventhub.EventMsg, 1)}
}

// Add msg to queue (tail)
func (q *Queue) enqueue(eventMsg eventhub.EventMsg) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// if queue is full
	if q.size == MaxSubQueueLength {
		return errors.New("subscription queue is full")
	}

	newNode := &node{data: eventMsg}
	if q.tail != nil {
		q.tail.next = newNode
		q.tail = newNode
	} else {
		q.head = newNode
		q.tail = newNode
	}
	q.size++

	select {
	case q.outBuf <- q.head.data:
	default: // outBuf is full
	}
	return nil
}

func (q *Queue) removeHead() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.head == nil {
		return
	}

	q.head = q.head.next
	if q.head == nil {
		q.tail = nil
	} else {
		select {
		case q.outBuf <- q.head.data:
		default: // outBuf is full
		}
	}
	q.size--

	return
}

func (q *Queue) clear() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.head = nil
	q.tail = nil
	q.size = 0
	close(q.outBuf)
}

func (q *Queue) wait() <-chan eventhub.EventMsg {
	return q.outBuf
}

// Get head node data
func (q *Queue) peek() (eventhub.EventMsg, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.head == nil {
		return eventhub.EventMsg{}, errors.New("empty queue")
	}
	return q.head.data, nil
}
