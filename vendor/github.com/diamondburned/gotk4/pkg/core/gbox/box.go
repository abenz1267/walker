package gbox

// #cgo pkg-config: glib-2.0
// #cgo CFLAGS: -Wno-deprecated-declarations
// #include <glib.h>
// extern void callbackDelete(guintptr _0);
import "C"

import (
	"github.com/diamondburned/gotk4/pkg/core/slab"
)

// Holy moly. This is unbelievable. It truly is unbelievable. I can't even
// believe myself.
//
// Check this out. C.gpointer is resolved to an unsafe.Pointer. That... sounds
// fine? I mean, a gpointer is a void*, so that checks out with the C specs.
//
// C functions usually take in a user_data parameter of type C.gpointer, which
// package gbox stores Go values by giving it an incremental ID and sending it
// to the function. This usually works, except the programs will randomly crash
// when we do this.
//
// What the hell? Why? It turns out that, because we cast a uintptr into a
// C.gpointer, we're effectively casting it to an unsafe.Pointer. As a result,
// Go will try to scan the pointer randomly, sees that it's not a pointer but is
// actually some wacky value, and panics.
//
// The real what-the-fuck here is why-the-fuck is it doing this to a C pointer?
// I guess for safety, but it seems so ridiculous to generate this kind of code.
//
// It turns out that this problem isn't unique to this library (or gotk3), nor
// was it a new issue. It also turns out that this is exactly what
// mattn/go-pointer is meant to work around, which is funny, because I've always
// thought it did what gbox did before.
//
// It also turns out that the new runtime/cgo package made this quite
// misleading. When you read the examples multiple times, you'll notice that
// they all use C.uintptr_t, not any other type. This is extremely important,
// because the fact that these functions use that type instead of void* means
// that they circumvent all checks.
//
// Now, we can take the easy way and do what mattn/go-pointer actually does: we
// allocate a pointer with nothing in it, and we use that pointer as the key.
// Cgo scans the pointer, sees that it's a valid pointer, and proceeds.
//
// OR, we can do it the stupid way. We can see what Go does to determine an
// invalid pointer and do some pointer trickery to fool it into believing we
// have a valid pointer.
//
// If you inspect the panic message caused by the runtime when it stumbles on
// the weird error, you'll see a runtime.adjustpointers routine in the trace.
// Inspecting the routine closer reveals this following snippet of code:
//
//	if f.valid() && 0 < p && p < minLegalPointer && debug.invalidptr != 0 {
//	    // Looks like a junk value in a pointer slot.
//	    // Live analysis wrong?
//	    getg().m.traceback = 2
//	    print("runtime: bad pointer in frame ", funcname(f), " at ", pp, ": ", hex(p), "\n")
//	    throw("invalid pointer found on stack")
//	}
//
// The check should make this pretty obvious: one of the conditions are failing,
// causing the runtime to panic with the "invalid pointer found on stack"
// message. The upper part of the stack trace tells us that the address at p is
// 0x31, and the code is telling us that 0x31 is not a good value.
//
// To find out why, let's check what minLegalPointer is:
//
//	―❤―▶ grepr minLegalPointer
//	./malloc.go:316:	// minLegalPointer is the smallest possible legal pointer.
//	./malloc.go:321:	minLegalPointer uintptr = 4096
//
// There we go! The returned value was 0x31, which is less than 4096, so the
// runtime trips on that and panics. Now, if we can just add up exactly that
// value, we can trick the runtime into thinking that it is, in fact, a valid
// pointer. Isn't that great? Surely this can't blow up when we reach a higher
// number. Who cares?
const minLegalPointer = 4096

var registry slab.Slab

func init() {
	registry.Grow(1024)
}

// Assign assigns the given value and returns the fake pointer.
func Assign(v interface{}) uintptr {
	return registry.Put(v, false) + minLegalPointer
}

// AssignOnce stores the given value so that, when the value is retrieved, it
// will immediately be deleted.
func AssignOnce(v interface{}) uintptr {
	return registry.Put(v, true) + minLegalPointer
}

// Get gets the value from the given fake pointer. The context must match the
// given value in Assign.
func Get(ptr uintptr) interface{} {
	return registry.Get(ptr - minLegalPointer)
}

// Delete deletes a boxed value. It is exposed to C under the name
// "callbackDelete".
func Delete(ptr uintptr) {
	registry.Delete(ptr - minLegalPointer)
}

//export callbackDelete
func callbackDelete(ptr uintptr) {
	registry.Delete(ptr - minLegalPointer)
}

// Pop gets a value and deletes it atomically.
func Pop(ptr uintptr) interface{} {
	return registry.Pop(ptr - minLegalPointer)
}
