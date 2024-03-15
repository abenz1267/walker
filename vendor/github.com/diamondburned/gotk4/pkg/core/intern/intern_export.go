package intern

// #cgo pkg-config: gobject-2.0
// #include "intern.h"
import "C"

import (
	"unsafe"
)

// goToggleNotify is called by GLib on each toggle notification. It doesn't
// actually free anything and relies on Box's finalizer to free both the box and
// the C GObject.
//
//export goToggleNotify
func goToggleNotify(_ C.gpointer, obj *C.GObject, isLastInt C.gboolean) {
	gobject := unsafe.Pointer(obj)
	isLast := isLastInt != C.FALSE

	shared.mu.Lock()
	defer shared.mu.Unlock()

	var box *Box
	if isLast {
		box = makeWeak(gobject)
	} else {
		box = makeStrong(gobject)
	}

	if box == nil {
		if toggleRefs != nil {
			toggleRefs.Println(objInfo(unsafe.Pointer(obj)), "goToggleNotify: box not found")
		}
		return
	}

	if box.finalize {
		if toggleRefs != nil {
			toggleRefs.Println(objInfo(unsafe.Pointer(obj)), "goToggleNotify: resurrecting finalized object")
		}
		box.finalize = false
		return
	}

	if toggleRefs != nil {
		toggleRefs.Println(objInfo(unsafe.Pointer(obj)), "goToggleNotify: is last =", isLast)
	}
}

// finishRemovingToggleRef is called after the toggle reference removal routine
// is dispatched in the main loop. It removes the GObject from the global maps.
//
//export goFinishRemovingToggleRef
func goFinishRemovingToggleRef(gobject unsafe.Pointer) {
	if toggleRefs != nil {
		toggleRefs.Printf("goFinishRemovingToggleRef: called on %p", gobject)
	}

	shared.mu.Lock()
	defer shared.mu.Unlock()

	box, strong := gets(gobject)
	if box == nil {
		if toggleRefs != nil {
			toggleRefs.Printf(
				"goFinishRemovingToggleRef: object %p not found in weak map",
				gobject)
		}
		return
	}

	if toggleRefs != nil {
		toggleRefs.Printf(
			"goFinishRemovingToggleRef: object %p found in weak map containing box %p",
			gobject, box)
	}

	if strong {
		if toggleRefs != nil {
			toggleRefs.Printf(
				"goFinishRemovingToggleRef: object %p still strong",
				gobject)
		}
		return
	}

	if !box.finalize {
		if toggleRefs != nil {
			toggleRefs.Printf(
				"goFinishRemovingToggleRef: object %p not finalizing, instead resurrected",
				gobject)
		}
		return
	}

	shared.weak.Delete(gobject)

	if toggleRefs != nil {
		toggleRefs.Printf("goFinishRemovingToggleRef: removed %p from weak ref, will be finalized soon", gobject)
	}

	if objectProfile != nil {
		objectProfile.Remove(gobject)
	}
}
