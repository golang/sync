package singleflight

import (
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sync/lock"
)

type SimpleSingleFlight struct {
	sync.Map
}

func (this *SimpleSingleFlight) leadLock(key string) *lock.LeadLock {
	iface, ok := this.Load(key)
	if !ok {
		iface, _ = this.LoadOrStore(key, lock.NewLeadLock())
	}
	return iface.(*lock.LeadLock)
}

type result struct {
	v   interface{}
	err error
}

func (this *SimpleSingleFlight) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	var res *result
	defer func() {
		if e, ok := res.err.(*panicError); ok {
			panic(e)
		} else if res.err == errGoexit {
			runtime.Goexit()
		}
	}()

	ll := this.leadLock(key)
	ptr := ll.Lock()
	if ptr != nil {
		res = (*result)(ptr)
		return res.v, res.err
	}

	res = &result{}
	func() {
		var getBack bool
		defer func() {
			if !getBack {
				if r := recover(); r != nil {
					res.err = newPanicError(r)
				} else {
					res.err = errGoexit
				}
			}
			ll.Unlock(unsafe.Pointer(res))
		}()
		res.v, res.err = fn()
		getBack = true
	}()

	return res.v, res.err
}
