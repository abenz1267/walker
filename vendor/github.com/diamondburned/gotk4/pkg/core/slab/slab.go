package slab

import (
	"sync"
	"sync/atomic"
)

// atomicContainer is a struct containing an interface that is used for swapping
// into Value.
type atomicContainer struct {
	data interface{}
}

type slabEntry struct {
	Value atomic.Value
	Index uintptr
	Once  bool
}

// Slab is an implementation of the internal registry free list. A zero-value
// instance is a valid instance. A slab is safe to use concurrently.
type Slab struct {
	list []slabEntry  // 3 words
	mu   sync.RWMutex // 3 words (assuming 64-bit)
	free uintptr      // 1 word
}

// Grow grows the slab to the given capacity.
func (s *Slab) Grow(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cap(s.list) < n {
		new := make([]slabEntry, len(s.list), n)
		copy(new, s.list)
		s.list = new
	}
}

// Put stores the entry inside the slab. If once is true, then when the entry is
// retrieved using Get, it will also be wiped off the list.
func (s *Slab) Put(entry interface{}, once bool) uintptr {
	slabEntry := slabEntry{atomic.Value{}, 0, once}
	if once {
		// Wrap the entry value inside an atomic container for type consistency.
		slabEntry.Value.Store(atomicContainer{entry})
	} else {
		slabEntry.Value.Store(entry)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.free == uintptr(len(s.list)) {
		index := uintptr(len(s.list))
		s.list = append(s.list, slabEntry)
		s.free++

		return index
	}

	index := s.free
	s.free = s.list[index].Index
	s.list[index] = slabEntry

	return index
}

// Get gets the entry at the given index.
func (s *Slab) Get(i uintptr) interface{} {
	s.mu.RLock()

	// Perform simple bound check.
	if i >= uintptr(len(s.list)) {
		s.mu.RUnlock()
		return nil
	}

	entry := s.list[i]
	var v interface{}

	// Perform an atomic value retrieve.
	if entry.Once {
		// Use Swap here, so that future Get is guaranteed to return an empty
		// atomicContainer while we're acquiring the lock in Pop.
		container := entry.Value.Swap(atomicContainer{}).(atomicContainer)
		s.mu.RUnlock()
		// Reacquire the lock and free the entry in the list.
		s.Delete(i)
		// Set v if the container is not empty.
		if container.data != nil {
			v = container.data
		}
	} else {
		v = entry.Value.Load()
		s.mu.RUnlock()
	}

	return v
}

// Pop removes the entry at the given index and returns the old value.
func (s *Slab) Pop(i uintptr) interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Perform simple bound check.
	if i >= uintptr(len(s.list)) {
		return nil
	}

	popped := s.list[i].Value.Load()
	s.list[i] = slabEntry{atomic.Value{}, s.free, false}
	s.free = i

	return popped
}

// Delete removes the entry at the given index.
func (s *Slab) Delete(i uintptr) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Perform simple bound check.
	if i < uintptr(len(s.list)) {
		s.list[i] = slabEntry{atomic.Value{}, s.free, false}
		s.free = i
	}
}
