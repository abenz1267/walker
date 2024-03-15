// Package intern implements value interning for Cgo sharing.
package intern

// #cgo pkg-config: gobject-2.0
// #include "intern.h"
import "C"

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/KarpelesLab/weak"
	"github.com/diamondburned/gotk4/pkg/core/closure"
	"github.com/diamondburned/gotk4/pkg/core/gdebug"

	// Require a non-moving GC for heap pointers. Current GC is moving only by
	// the stack. See https://github.com/go4org/intern.
	_ "go4.org/unsafe/assume-no-moving-gc"
)

// Box is an opaque type holding extra data.
type Box struct {
	dummy    *boxDummy
	closures atomic.Pointer[closure.Registry]
	gobject  unsafe.Pointer
	finalize bool
}

type boxDummy struct {
	gobject unsafe.Pointer
}

// Object returns Box's C GObject pointer.
func (b *Box) GObject() unsafe.Pointer {
	return b.gobject
}

// Closures returns the closure registry for this Box.
func (b *Box) Closures() *closure.Registry {
	closures := b.closures.Load()
	if closures != nil {
		return closures
	}

	// If the closures are nil, then we'll have to create a new one.
	closures = closure.NewRegistry()
	if !b.closures.CompareAndSwap(nil, closures) {
		// If the CAS failed, then we'll read the value again.
		return b.closures.Load()
	}

	return closures
}

// Hack to force an object on the heap.
var never bool
var sink_ interface{}

//go:nosplit
func sink(v interface{}) {
	if never {
		sink_ = v
	}
}

var (
	traceObjects  = gdebug.NewDebugLoggerNullable("trace-objects")
	toggleRefs    = gdebug.NewDebugLoggerNullable("toggle-refs")
	objectProfile *pprof.Profile
)

func init() {
	if gdebug.HasKey("profile-objects") {
		objectProfile = pprof.NewProfile("gotk4-object-box")
	}
}

func objInfo(obj unsafe.Pointer) string {
	return fmt.Sprintf("%p (%s):", obj, C.GoString(C.gotk4_object_type_name(C.gpointer(obj))))
}

func objRefCount(obj unsafe.Pointer) int {
	return int(C.g_atomic_int_get((*C.gint)(unsafe.Pointer(&(*C.GObject)(obj).ref_count))))
}

// newBox creates a zero-value instance of Box.
func newBox(obj unsafe.Pointer) *Box {
	box := &Box{}
	box.gobject = obj

	// Cheat Go's GC by adding a finalizer to a dummy pointer that is inside Box
	// but is not Box itself.
	box.dummy = &boxDummy{gobject: obj}
	sink(box.dummy)
	runtime.SetFinalizer(box.dummy, finalizeBox)

	if objectProfile != nil {
		objectProfile.Add(obj, 3)
	}

	if traceObjects != nil {
		traceObjects.Printf("%p: %s", obj, debug.Stack())
	}

	// Force box on the heap. Objects on the stack can move, but not objects on
	// the heap. At least not for now; the assume-no-moving-gc import will
	// guard against that.
	sink(box)

	return box
}

// shared contains shared closure data.
var shared = struct {
	mu sync.RWMutex
	// weak stores *Box while the object is in Go's heap. The finalizer will
	// move *Box to strong if the reference is toggled. This is only the case,
	// because the finalizer will not run otherwise.
	weak *weak.Map[unsafe.Pointer, Box]
	// strong stores *Box while the object is still referenced by C but not Go.
	strong map[unsafe.Pointer]*Box
}{
	weak:   weak.NewMap[unsafe.Pointer, Box](),
	strong: make(map[unsafe.Pointer]*Box, 1024),
}

// TryGet gets the Box associated with the GObject or nil if it's gone. The
// caller must not retain the Box pointer anywhere.
func TryGet(gobject unsafe.Pointer) *Box {
	shared.mu.RLock()
	box, _ := gets(gobject)
	shared.mu.RUnlock()
	return box
}

// Get gets the interned box for the given GObject C pointer. If the object is
// new or unknown, then a new box is made. If the intern box already exists for
// a given C pointer, then that box is weakly referenced and returned. The box
// will be reference-counted; the caller must use ShouldFree to unreference it.
func Get(gobject unsafe.Pointer, take bool) *Box {
	// If the registry does not exist, then we'll have to globally register it.
	// If the registry is currently strongly referenced, then we must move it to
	// a weak reference.

	box := TryGet(gobject)
	if box != nil {
		return box
	}

	shared.mu.Lock()

	box, _ = gets(gobject)
	if box != nil {
		shared.mu.Unlock()
		return box
	}

	box = newBox(gobject)

	// add_toggle_ref's documentation states:
	//
	//    Since a (normal) reference must be held to the object before
	//    calling g_object_add_toggle_ref(), the initial state of the
	//    reverse link is always strong.
	//
	shared.strong[gobject] = box

	if toggleRefs != nil {
		toggleRefs.Println(objInfo(gobject),
			"Get: will introduce new box, current ref =", objRefCount(gobject))
	}

	shared.mu.Unlock()

	C.g_object_add_toggle_ref(
		(*C.GObject)(gobject),
		(*[0]byte)(C.goToggleNotify), nil,
	)

	// We should already have a strong reference. Sink the object in case. This
	// will force the reference to be truly strong.
	if C.g_object_is_floating(C.gpointer(gobject)) != C.FALSE {
		// First, we need to ref_sink the object to convert the floating
		// reference to a strong reference.
		C.g_object_ref_sink(C.gpointer(gobject))
		// Then, we need to unref it to balance the ref_sink.
		C.g_object_unref(C.gpointer(gobject))

		if toggleRefs != nil {
			toggleRefs.Println(objInfo(gobject),
				"Get: ref_sink'd the object, current ref =", objRefCount(gobject))
		}
	}

	// If we're "not taking," then we can assume our ownership over the object,
	// meaning the strong reference is now ours. That means we need to replace
	// it, not add.
	if !take {
		if toggleRefs != nil {
			toggleRefs.Println(objInfo(gobject),
				"Get: not taking, so unrefing the object, current ref =", objRefCount(gobject))
		}
		C.g_object_unref(C.gpointer(gobject))
	}

	if toggleRefs != nil {
		toggleRefs.Println(objInfo(gobject),
			"Get: introduced new box, current ref =", objRefCount(gobject))
	}

	// Undo the initial ref_sink.
	// C.g_object_unref(C.gpointer(gobject))

	return box
}

// Free explicitly frees the box permanently. It must not be resurrected after
// this.
//
// Deprecated: this function is no longer needed.
func Free(box *Box) {
	panic("not implemented")
}

// finalizeBox only delays its finalization until GLib notifies us a toggle. It
// does so for as long as an object is stored only in the Go heap. Once the
// object is also shared, the toggle notifier will strongly reference the Box.
func finalizeBox(dummy *boxDummy) {
	if dummy == nil {
		panic("bug: finalizeBox called with nil dummy")
	}

	shared.mu.Lock()
	defer shared.mu.Unlock()

	box, strong := gets(dummy.gobject)
	if box == nil {
		// Silently ignore unknown objects.
		//
		// This is a trick to make sure that the box is really finalized. Turns
		// out it hates being finalized in goFinishRemovingToggleRef, so we just
		// don't call it there and let the GC do its thing.

		if traceObjects != nil {
			traceObjects.Printf("%p: finalizeBox: unknown object", dummy.gobject)
		}

		return
	}

	// Always delegate the finalization to the next cycle.
	// This won't be the case once goFinishRemovingToggleRef is called.
	runtime.SetFinalizer(dummy, finalizeBox)

	if box.finalize {
		// If the box is already finalizing, then we don't need to do anything.
		// Repeat this until box is gone from the registry.

		if toggleRefs != nil {
			toggleRefs.Printf(
				"%p: finalizeBox: already finalizing, waiting for goFinishRemovingToggleRef",
				dummy.gobject)
		}

		return
	}

	if strong {
		// If strong: the closures are strong-referenced, then they might still
		// be referenced from the C side, and those closures might access this
		// object. Don't free.

		if toggleRefs != nil {
			toggleRefs.Println(
				objInfo(dummy.gobject),
				"finalizeBox: moving finalize to next GC cycle since object is still strong")
		}

		return
	}

	// Mark the box as finalizing.
	box.finalize = true

	// Do this before we dispatch the remove_toggle_ref, because the
	// remove_toggle_ref might destroy the object.
	var objInfoS string
	if toggleRefs != nil {
		objInfoS = objInfo(dummy.gobject)
	}

	// Do this in the main loop instead. This is because finalizers are
	// called in a finalizer thread, and our remove_toggle_ref might be
	// destroying other main loop objects.
	C.g_main_context_invoke(
		nil, // nil means the default main context
		(*[0]byte)(C.gotk4_intern_remove_toggle_ref),
		C.gpointer(dummy.gobject))

	if toggleRefs != nil {
		toggleRefs.Printf(
			"%s finalizeBox: remove_toggle_ref queued for next main loop iteration for box %p",
			objInfoS, box)
	}
}

//go:nosplit
func gets(gobject unsafe.Pointer) (b *Box, strong bool) {
	if strong, ok := shared.strong[gobject]; ok {
		return strong, true
	}

	if weak := shared.weak.Get(gobject); weak != nil {
		// If forObject is false, then that probably means this was called
		// inside goMarshal while the Go object is still alive, otherwise
		// toggleNotify would've moved it over. We don't have to worry about
		// this being freed as long as we acquire the mutex.
		//
		// TODO: does this actually resurrect the value properly? We have a
		// mutex to guard this which is also used in the finalizer, so it
		// shouldn't explode, but still.
		return weak, false
	}

	return nil, false
}

// makeStrong forces the Box instance associated with the given object to be
// strongly referenced.
//
//go:nosplit
func makeStrong(gobject unsafe.Pointer) *Box {
	// TODO: double mutex check, similar to ShouldFree.

	box, strong := gets(gobject)
	if toggleRefs != nil {
		toggleRefs.Println(objInfo(gobject), "makeStrong: obtained box", box, "strong =", strong)
	}
	if box == nil {
		return nil
	}

	if !strong {
		shared.strong[gobject] = box
		shared.weak.Delete(gobject)

		// Clear weak.Map's finalizer.
		runtime.SetFinalizer(box, nil)
	}

	return box
}

// makeWeak forces the Box intsance associated with the given object to be
// weakly referenced.
//
//go:nosplit
func makeWeak(gobject unsafe.Pointer) *Box {
	box, strong := gets(gobject)
	if toggleRefs != nil {
		toggleRefs.Println(objInfo(gobject), "makeWeak: obtained box", box, "strong =", strong)
	}
	if box == nil {
		return nil
	}

	if strong {
		shared.weak.Set(gobject, box)
		delete(shared.strong, gobject)
	}

	return box
}
