package sync

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type KeyValueItem struct {
	K interface{}
	V interface{}
}
type LKQueueSet struct {
	mu        sync.RWMutex
	Map       map[interface{}]struct{}
	Priority  uint64
	Root      []byte
	ListBytes []byte
	List      []*KeyValueItem
}

func NewLKQueueMapItem() *LKQueueSet {
	item := &LKQueueSet{
		Map: make(map[interface{}]struct{}, 0),
	}
	return item
}

func (ls *LKQueueSet) AddListData(setItem *LKQueueSet) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	for _, v := range setItem.List {
		if _, ok := ls.Map[v.K]; !ok {
			ls.Map[v.K] = struct{}{}
			ls.List = append(ls.List, v)
		}
	}
}

func (lm *LKQueueSet) PopListData() []*KeyValueItem {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	list := lm.List
	if len(list) != 0 {
		lm.List = make([]*KeyValueItem, 0)
		return list
	} else {
		return nil
	}
}

type LockFreeSetQueue struct {
	next unsafe.Pointer
}

type nodeSet struct {
	priority uint64
	value    *LKQueueSet
	next     unsafe.Pointer
}

func NewLKQueueSet() *LockFreeSetQueue {
	n := unsafe.Pointer(&nodeSet{})
	lk := &LockFreeSetQueue{next: n}
	return lk
}

func (q *LockFreeSetQueue) Push(v *LKQueueSet, priority uint64) error {
	n := &nodeSet{
		priority: priority,
		value:    v}
PushCAS:
	head := LoadSet(&q.next)
	for {

		if head.value == nil {
			ok := casSet(&q.next, head, n)
			if !ok {
				goto PushCAS
			}
			return nil
		}
		if atomic.LoadUint64(&head.priority) > priority {
			ok := casSet(&q.next, head, n)
			if ok {
				casSet(&n.next, nil, head)
			} else {
				goto PushCAS
			}
			return nil
		} else if atomic.LoadUint64(&head.priority) == priority {

			head.value.AddListData(n.value)

		} else if atomic.LoadUint64(&head.priority) < priority {
			if head.next == nil {
				ok := casSet(&head.next, nil, n)
				if !ok {
					goto PushCAS
				}
				return nil
			}
			nextnode := LoadSet(&head.next)
			if atomic.LoadUint64(&nextnode.priority) > priority {
				ok := casSet(&head.next, nextnode, n)
				if ok {
					casSet(&n.next, nil, nextnode)
				} else {
					goto PushCAS
				}
			}
			head = LoadSet(&head.next)
		}
	}
}
func (q *LockFreeSetQueue) GetHeadRoot() []byte {
	head := LoadSet(&q.next)
	if head.value == nil {
		return nil
	} else {
		return head.value.Root

	}
}

func (q *LockFreeSetQueue) PopHeadList() []*KeyValueItem {
	head := LoadSet(&q.next)
	if head.value == nil {
		return nil
	} else {
		list := head.value.PopListData()
		return list
	}
}

func (q *LockFreeSetQueue) dropHead() {
popCAS:
	head := LoadSet(&q.next)
	if head.value == nil {
		return
	} else {
		if head.next == nil {
			n := unsafe.Pointer(&nodeSet{})
			ok := atomic.CompareAndSwapPointer(&head.next, head.next, n)
			if !ok {
				goto popCAS
			}
			return
		}
		tailNext := LoadSet(&head.next)
		ok := casSet(&q.next, head, tailNext)
		if !ok {
			goto popCAS
		}
		return
	}
}

func (q *LockFreeSetQueue) GetHeadPriority() (uint64, error) {
	if q.next != nil {
		head := LoadSet(&q.next)
		return atomic.LoadUint64(&head.priority), nil
	}
	return 0, ErrQueueEmpty

}

func LoadSet(p *unsafe.Pointer) (n *nodeSet) {
	return (*nodeSet)(atomic.LoadPointer(p))
}

func casSet(p *unsafe.Pointer, old, new *nodeSet) (ok bool) {
	return atomic.CompareAndSwapPointer(
		p, unsafe.Pointer(old), unsafe.Pointer(new))
}
