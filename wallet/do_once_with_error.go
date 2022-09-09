package wallet

import (
	"sync"
	"sync/atomic"
)

/*
This file provides similar function with sync.Once, but provides error return.
*/

type onceWithErr struct {
	done uint32
	m    sync.Mutex
}

// Do input func once if input func return nil.
// If input func doesn't return nil then return error and input func can Do again until return nil.
func (o *onceWithErr) Do(f func() error) error {
	if atomic.LoadUint32(&o.done) == 0 {
		return o.doSlow(f)
	}
	return nil
}

func (o *onceWithErr) doSlow(f func() error) error {
	o.m.Lock()
	defer o.m.Unlock()
	if o.done == 0 {
		err := f()
		if err == nil { // f returns no error, then change status of done
			atomic.StoreUint32(&o.done, 1)
			return nil
		} else {
			return err
		}
	} else {
		panic("can not happen")
	}
}
