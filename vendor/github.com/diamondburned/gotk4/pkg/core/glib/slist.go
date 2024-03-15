package glib

// #include <glib.h>
// #include <glib-object.h>
// #include "glib.go.h"
import "C"

import (
	"runtime"
	"unsafe"
)

// TODO(go1.18): Make SList a generic type.

// SList is a representation of Glib's GSList. A SList must be manually freed
// by either calling Free() or FreeFull()
type SList struct {
	list *C.struct__GSList
	// If set, dataWrap is called every time Data()
	// is called to wrap raw underlying
	// value into appropriate type.
	dataWrap func(unsafe.Pointer) interface{}
}

func WrapSList(obj uintptr) *SList {
	return wrapSList((*C.struct__GSList)(unsafe.Pointer(obj)))
}

func wrapSList(obj *C.struct__GSList) *SList {
	if obj == nil {
		return nil
	}

	//NOTE a list should be freed by calling either
	//g_slist_free() or g_slist_free_full(). However, it's not possible to use a
	//finalizer for this.
	return &SList{list: obj}
}

func (v *SList) wrapNewHead(obj *C.struct__GSList) *SList {
	if obj == nil {
		return nil
	}
	return &SList{
		list:     obj,
		dataWrap: v.dataWrap,
	}
}

func (v *SList) Native() uintptr {
	return uintptr(unsafe.Pointer(v.list))
}

func (v *SList) native() *C.struct__GSList {
	if v == nil || v.list == nil {
		return nil
	}
	return v.list
}

// DataWapper sets wrap functions, which is called during NthData()
// and Data(). It's used to cast raw C data into appropriate
// Go structures and types every time that data is retreived.
func (v *SList) DataWrapper(fn func(unsafe.Pointer) interface{}) {
	if v == nil {
		return
	}
	v.dataWrap = fn
}

func (v *SList) Append(data uintptr) *SList {
	defer runtime.KeepAlive(v)

	ret := C.g_slist_append(v.native(), C.gpointer(data))
	if ret == v.native() {
		return v
	}

	return v.wrapNewHead(ret)
}

// Length is a wrapper around g_slist_length().
func (v *SList) Length() uint {
	defer runtime.KeepAlive(v)

	return uint(C.g_slist_length(v.native()))
}

// Next is a wrapper around the next struct field
func (v *SList) Next() *SList {
	defer runtime.KeepAlive(v)

	n := v.native()
	if n == nil {
		return nil
	}

	return wrapSList(n.next)
}

// dataRaw is a wrapper around the data struct field
func (v *SList) dataRaw() unsafe.Pointer {
	n := v.native()
	if n == nil {
		return nil
	}
	return unsafe.Pointer(n.data)
}

// DataRaw is a wrapper around the data struct field
func (v *SList) DataRaw() unsafe.Pointer {
	defer runtime.KeepAlive(v)

	n := v.native()
	if n == nil {
		return nil
	}
	return unsafe.Pointer(n.data)
}

// Data acts the same as data struct field, but it returns raw unsafe.Pointer as interface.
// TODO: Align with List struct and add member + logic for `dataWrap func(unsafe.Pointer) interface{}`?
func (v *SList) Data() interface{} {
	defer runtime.KeepAlive(v)

	ptr := v.dataRaw()
	if v.dataWrap != nil {
		return v.dataWrap(ptr)
	}
	return ptr
}

// Foreach acts the same as g_slist_foreach().
// No user_data argument is implemented because of Go clojure capabilities.
func (v *SList) Foreach(fn func(item interface{})) {
	for l := v; l != nil; l = l.Next() {
		fn(l.Data())
	}
}

// Free is a wrapper around g_slist_free().
func (v *SList) Free() {
	C.g_slist_free(v.native())
	runtime.KeepAlive(v)
}

// FreeFull is a wrapper around g_slist_free_full().
func (v *SList) FreeFull() {
	//TODO implement GDestroyNotify callback
	C.g_slist_free_full(v.native(), nil)
	runtime.KeepAlive(v)
}

// GSList * 	g_slist_alloc ()
// GSList * 	g_slist_prepend ()
// GSList * 	g_slist_insert ()
// GSList * 	g_slist_insert_before ()
// GSList * 	g_slist_insert_sorted ()
// GSList * 	g_slist_remove ()
// GSList * 	g_slist_remove_link ()
// GSList * 	g_slist_delete_link ()
// GSList * 	g_slist_remove_all ()
// void 	g_slist_free_1 ()
// GSList * 	g_slist_copy ()
// GSList * 	g_slist_copy_deep ()
// GSList * 	g_slist_reverse ()
// GSList * 	g_slist_insert_sorted_with_data ()
// GSList * 	g_slist_sort ()
// GSList * 	g_slist_sort_with_data ()
// GSList * 	g_slist_concat ()
// GSList * 	g_slist_last ()
// GSList * 	g_slist_nth ()
// gpointer 	g_slist_nth_data ()
// GSList * 	g_slist_find ()
// GSList * 	g_slist_find_custom ()
// gint 	g_slist_position ()
// gint 	g_slist_index ()
