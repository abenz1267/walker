package weak

import (
	"runtime"
	"sync/atomic"
	"unsafe"
)

// Ref is a weak reference to a Go object
type Ref[T any] struct {
	hidden uintptr
	state  refState
}

func (wr *Ref[T]) value() *T {
	v := atomic.LoadUintptr(&wr.hidden)
	if v == 0 {
		return nil
	}
	return (*T)(unsafe.Pointer(v))
}

// Get returns the value for a given weak reference pointer
func (wr *Ref[T]) Get() *T {
	for {
		if wr.state.CaS(refALIVE, refINUSE) {
			val := wr.value()
			// all good
			wr.state.Set(refALIVE) // set back to alive
			return val
		}
		if wr.state.Get() == refDEAD {
			return nil
		}
		runtime.Gosched()
	}
}

// NewRef returns a reference to the object v that may be cleared by the garbage collector
func NewRef[T any](v *T) *Ref[T] {
	if v == nil {
		return &Ref[T]{0, refDEAD}
	}
	wr := &Ref[T]{uintptr(unsafe.Pointer(v)), refALIVE}
	var f func(p *T)
	f = func(p *T) {
		if wr.state.CaS(refALIVE, refDEAD) {
			// we're now refdead, clear the pointer value
			atomic.StoreUintptr(&wr.hidden, 0)
			return
		}
		// this was not ALIVE, it means it was likely INUSE, re-set finalizer and wait
		runtime.SetFinalizer(p, f)
	}
	runtime.SetFinalizer(v, f)

	return wr
}

// NewRefDestroyer returns a reference to the object v that may be cleared by the garbage collector,
// in which case destroy will be called.
func NewRefDestroyer[T any](v *T, destroy func(v *T, wr *Ref[T])) *Ref[T] {
	if v == nil {
		return &Ref[T]{0, refDEAD}
	}
	wr := &Ref[T]{uintptr(unsafe.Pointer(v)), refALIVE}
	var f func(p *T)
	f = func(p *T) {
		if wr.state.CaS(refALIVE, refDEAD) {
			atomic.StoreUintptr(&wr.hidden, 0)
			if destroy != nil {
				go destroy(p, wr)
			}
			return
		}
		// this was not ALIVE, it means it was likely INUSE, re-set finalizer and wait
		runtime.SetFinalizer(p, f)
	}
	runtime.SetFinalizer(v, f)

	return wr
}
