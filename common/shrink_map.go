package common

import "sync"

const (
	copyThreshold = 1000
	maxDeletion   = 10000
)

type ShrinkableMap struct {
	lock        sync.RWMutex
	deletionOld int
	deletionNew int
	dirtyOld    map[interface{}]interface{}
	dirtyNew    map[interface{}]interface{}
}

func NewShrinkMap() *ShrinkableMap {
	return &ShrinkableMap{
		dirtyOld: make(map[interface{}]interface{}),
		dirtyNew: make(map[interface{}]interface{}),
	}
}

func (m *ShrinkableMap) Del(key interface{}) {
	m.lock.Lock()
	if _, ok := m.dirtyOld[key]; ok {
		delete(m.dirtyOld, key)
		m.deletionOld++
	} else if _, ok := m.dirtyNew[key]; ok {
		delete(m.dirtyNew, key)
		m.deletionNew++
	}
	if m.deletionOld >= maxDeletion && len(m.dirtyOld) < copyThreshold {
		for k, v := range m.dirtyOld {
			m.dirtyNew[k] = v
		}
		m.dirtyOld = m.dirtyNew
		m.deletionOld = m.deletionNew
		m.dirtyNew = make(map[interface{}]interface{})
		m.deletionNew = 0
	}
	if m.deletionNew >= maxDeletion && len(m.dirtyNew) < copyThreshold {
		for k, v := range m.dirtyNew {
			m.dirtyOld[k] = v
		}
		m.dirtyNew = make(map[interface{}]interface{})
		m.deletionNew = 0
	}
	m.lock.Unlock()
}

func (m *ShrinkableMap) CallBackDelNoLock(key interface{}) {
	if _, ok := m.dirtyOld[key]; ok {
		delete(m.dirtyOld, key)
		m.deletionOld++
	} else if _, ok := m.dirtyNew[key]; ok {
		delete(m.dirtyNew, key)
		m.deletionNew++
	}
	if m.deletionOld >= maxDeletion && len(m.dirtyOld) < copyThreshold {
		for k, v := range m.dirtyOld {
			m.dirtyNew[k] = v
		}
		m.dirtyOld = m.dirtyNew
		m.deletionOld = m.deletionNew
		m.dirtyNew = make(map[interface{}]interface{})
		m.deletionNew = 0
	}
	if m.deletionNew >= maxDeletion && len(m.dirtyNew) < copyThreshold {
		for k, v := range m.dirtyNew {
			m.dirtyOld[k] = v
		}
		m.dirtyNew = make(map[interface{}]interface{})
		m.deletionNew = 0
	}
}

func (m *ShrinkableMap) Get(key interface{}) (interface{}, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if val, ok := m.dirtyOld[key]; ok {
		return val, true
	}

	val, ok := m.dirtyNew[key]
	return val, ok
}

func (m *ShrinkableMap) AllKeys() []interface{} {
	m.lock.RLock()
	defer m.lock.RUnlock()

	var keys []interface{}

	for k, _ := range m.dirtyOld {
		keys = append(keys, k)
	}

	for k, _ := range m.dirtyNew {
		keys = append(keys, k)
	}

	return keys
}

type IterCBFunc func(key interface{}, val interface{})

func (m *ShrinkableMap) IterateCallback(iterCBF IterCBFunc) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for k, v := range m.dirtyOld {
		iterCBF(k, v)
	}

	for k, v := range m.dirtyNew {
		iterCBF(k, v)
	}
}

func (m *ShrinkableMap) Set(key, value interface{}) {
	m.lock.Lock()
	if m.deletionOld <= maxDeletion {
		if _, ok := m.dirtyNew[key]; ok {
			delete(m.dirtyNew, key)
			m.deletionNew++
		}
		m.dirtyOld[key] = value
	} else {
		if _, ok := m.dirtyOld[key]; ok {
			delete(m.dirtyOld, key)
			m.deletionOld++
		}
		m.dirtyNew[key] = value
	}
	m.lock.Unlock()
}

func (m *ShrinkableMap) CallBackSetNoLock(key, value interface{}) {
	if m.deletionOld <= maxDeletion {
		if _, ok := m.dirtyNew[key]; ok {
			delete(m.dirtyNew, key)
			m.deletionNew++
		}
		m.dirtyOld[key] = value
	} else {
		if _, ok := m.dirtyOld[key]; ok {
			delete(m.dirtyOld, key)
			m.deletionOld++
		}
		m.dirtyNew[key] = value
	}
}

func (m *ShrinkableMap) Size() int {
	m.lock.RLock()
	size := len(m.dirtyOld) + len(m.dirtyNew)
	m.lock.RUnlock()
	return size
}
