package sync

import (
	"errors"
	"sync/atomic"
	"unsafe"
)

var (
	ErrPriorityExisted = errors.New("error: priority is existed")
	ErrQueueEmpty      = errors.New("error:priority queue is empty")
)

type LockFreePriorityQueue struct {
	next unsafe.Pointer
}

type node struct {
	priority uint64
	value    interface{}
	next     unsafe.Pointer
}

func NewLKQueue() *LockFreePriorityQueue {
	n := unsafe.Pointer(&node{})
	lk := &LockFreePriorityQueue{next: n}
	return lk
}

func (q *LockFreePriorityQueue) Push(v interface{}, priority uint64) error {
	n := &node{
		priority: priority,
		value:    v}
PushCAS:
	head := Load(&q.next)
	for {

		if head.value == nil {
			ok := cas(&q.next, head, n)
			if !ok {
				goto PushCAS
			}
			return nil
		}
		if atomic.LoadUint64(&head.priority) > priority {
			ok := cas(&q.next, head, n)
			if ok {
				cas(&n.next, nil, head)
			} else {
				goto PushCAS
			}
			return nil
		} else if atomic.LoadUint64(&head.priority) == priority {
			return ErrPriorityExisted
		} else if atomic.LoadUint64(&head.priority) < priority {
			if head.next == nil {
				ok := cas(&head.next, nil, n)
				if !ok {
					goto PushCAS
				}
				return nil
			}
			nextnode := Load(&head.next)
			if atomic.LoadUint64(&nextnode.priority) > priority {
				ok := cas(&head.next, nextnode, n)
				if ok {
					cas(&n.next, nil, nextnode)
				} else {
					goto PushCAS
				}
			}
			head = Load(&head.next)
		}
	}
}

func (q *LockFreePriorityQueue) PopHead() interface{} {
popCAS:
	head := Load(&q.next)
	if head.value == nil {
		return nil
	} else {
		v := head.value
		if head.next == nil {
			n := unsafe.Pointer(&node{})
			ok := atomic.CompareAndSwapPointer(&head.next, head.next, n)
			if !ok {
				goto popCAS
			}
			return v
		}
		tailNext := Load(&head.next)
		ok := cas(&q.next, head, tailNext)
		if !ok {
			goto popCAS
		}
		return v
	}
}

func (q *LockFreePriorityQueue) GetHeadPriority() (uint64, error) {
	if q.next != nil {
		head := Load(&q.next)
		return atomic.LoadUint64(&head.priority), nil
	}
	return 0, ErrQueueEmpty

}

func Load(p *unsafe.Pointer) (n *node) {
	return (*node)(atomic.LoadPointer(p))
}

func cas(p *unsafe.Pointer, old, new *node) (ok bool) {
	return atomic.CompareAndSwapPointer(
		p, unsafe.Pointer(old), unsafe.Pointer(new))
}
