package closure

import (
	"sync"
	"unsafe"
)

// Registry describes the local closure registry of each object.
type Registry struct {
	reg sync.Map // unsafe.Pointer(*C.GClosure) -> *FuncStack
}

// NewRegistry creates an empty closure registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register registers the given GClosure callback.
func (r *Registry) Register(gclosure unsafe.Pointer, callback *FuncStack) {
	r.reg.Store(uintptr(gclosure), callback)
}

// Load loads the given GClosure's callback. Nil is returned if it's not found.
func (r *Registry) Load(gclosure unsafe.Pointer) *FuncStack {
	fs, ok := r.reg.Load(uintptr(gclosure))
	if !ok {
		return nil
	}
	return fs.(*FuncStack)
}

// Delete deletes the given GClosure callback.
func (r *Registry) Delete(gclosure unsafe.Pointer) {
	r.reg.Delete(uintptr(gclosure))
}
