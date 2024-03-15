package weak

import (
	"sync"
)

// Map is a thread safe map for objects to be kept as weak references, useful for cache/etc
type Map[K comparable, T any] struct {
	m map[K]*Ref[T]
	l sync.RWMutex
}

// Object implementing Destroyable added to a Map will have their Destroy() method called
// when the object is about to be removed. This replaces use of a finalizer.
type Destroyable interface {
	Destroy()
}

// NewMap returns a new weak reference map
func NewMap[K comparable, T any]() *Map[K, T] {
	res := &Map[K, T]{
		m: make(map[K]*Ref[T]),
	}

	return res
}

// Get returns the value at index k in the map. If no such value exists, nil is returned.
func (w *Map[K, T]) Get(k K) *T {
	w.l.RLock()
	defer w.l.RUnlock()

	v, ok := w.m[k]
	if !ok {
		return nil
	}

	return v.Get()
}

// Set inserts the value v if it does not already exists, and return it. If a value v
// already exists, then the previous value is returned.
func (w *Map[K, T]) Set(k K, v *T) *T {
	w.l.Lock()
	defer w.l.Unlock()

	// already exists?
	wr, ok := w.m[k]
	if ok {
		v2 := wr.Get()
		if v2 != nil {
			// return past (still alive) value
			return v2
		}
	}

	// store new value
	wr = NewRefDestroyer(v, func(dv *T, wr *Ref[T]) {
		w.destroy(wr, dv, k)
	})
	w.m[k] = wr

	return v
}

// Delete removes element at key k from the map. This doesn't call Destroy immediately
// as this would typically happen when the object is actually cleared by the garbage
// collector and instances of said object may still be used.
func (w *Map[K, T]) Delete(k K) {
	w.l.Lock()
	defer w.l.Unlock()

	delete(w.m, k)
}

func (w *Map[K, T]) destroy(wr *Ref[T], ptr *T, k K) {
	w.l.Lock()
	defer w.l.Unlock()

	wr2, ok := w.m[k]
	if !ok {
		return
	}
	if wr == wr2 {
		delete(w.m, k)
	}

	if v, ok := any(ptr).(Destroyable); ok {
		go v.Destroy()
	}
}
