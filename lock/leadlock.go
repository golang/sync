package lock

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type cond struct {
	uptr unsafe.Pointer
	mux  sync.Mutex
	cond *sync.Cond
}

func newCond() *cond {
	cond := &cond{}
	cond.cond = sync.NewCond(&cond.mux)
	return cond
}

func (this *cond) wait() {
	this.mux.Lock()
	for this.uptr == nil {
		this.cond.Wait()
	}
	this.mux.Unlock()
}

func (this *cond) wake(uptr unsafe.Pointer) {
	this.mux.Lock()
	this.uptr = uptr
	this.mux.Unlock()
	this.cond.Broadcast()
}

type LeadLock struct {
	nr    int32
	cond  *cond
	batch *cond
	mux   sync.Mutex
}

func NewLeadLock() *LeadLock {
	leadlock := &LeadLock{}
	leadlock.cond = newCond()
	return leadlock
}

func (this *LeadLock) Lock() unsafe.Pointer {
	var uptr unsafe.Pointer
	n := atomic.AddInt32(&this.nr, 1)
	ptr := (*unsafe.Pointer)(unsafe.Pointer(&this.cond))
	if n > 1 {
		cond := (*cond)(atomic.LoadPointer(ptr))
		cond.wait()
		uptr = cond.uptr
	} else {
		this.mux.Lock()
		atomic.StoreInt32(&this.nr, 0)
		this.batch = (*cond)(atomic.SwapPointer(ptr, unsafe.Pointer(newCond())))
	}
	return uptr
}

func (this *LeadLock) Unlock(uptr unsafe.Pointer) {
	ptr := unsafe.Pointer(&this.batch)
	batch := atomic.SwapPointer((*unsafe.Pointer)(ptr), unsafe.Pointer(nil))
	if batch == nil {
		return
	}
	(*cond)(batch).wake(uptr)
	this.mux.Unlock()
}
