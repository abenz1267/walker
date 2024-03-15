// Package gextras contains supplemental types to gotk3.
package gextras

// #cgo pkg-config: glib-2.0
// #include <glib.h>
// #include <gmodule.h> // HashTable
import "C"

import (
	"unsafe"
)

// ZeroString points to a null-terminated string of length 0.
var ZeroString unsafe.Pointer

func init() {
	ZeroString = unsafe.Pointer(C.malloc(1))
	*(*byte)(ZeroString) = 0
}

type record struct{ intern *internRecord }

type internRecord struct{ c unsafe.Pointer }

// StructNative returns the underlying C pointer of the given Go record struct
// pointer. It can be used like so:
//
//	rec := NewRecord(...) // T = *Record
//	c := (*namespace_record)(StructPtr(unsafe.Pointer(rec)))
func StructNative(ptr unsafe.Pointer) unsafe.Pointer {
	return (*record)(ptr).intern.c
}

// StructIntern returns the given struct's internal struct pointer.
func StructIntern(ptr unsafe.Pointer) *struct{ C unsafe.Pointer } {
	return (*struct{ C unsafe.Pointer })(unsafe.Pointer((*record)(ptr).intern))
}

// SetStructNative sets the native value inside the Go struct value that the
// given dst pointer points to. It can be used like so:
//
//	var rec Record
//	SetStructNative(&rec, cvalue) // T(cvalue) = *namespace_record
func SetStructNative(dst, native unsafe.Pointer) {
	(*record)(dst).intern.c = native
}

// NewStructNative creates a new Go struct from the given native pointer. The
// finalizer is NOT set.
func NewStructNative(native unsafe.Pointer) unsafe.Pointer {
	r := record{intern: &internRecord{native}}
	return unsafe.Pointer(&r)
}

// HashTableSize returns the size of the *GHashTable.
func HashTableSize(ptr unsafe.Pointer) int {
	return int(C.g_hash_table_size((*C.GHashTable)(ptr)))
}

// MoveHashTable calls f on every value of the given *GHashTable and frees each
// element in the process if rm is true.
func MoveHashTable(ptr unsafe.Pointer, rm bool, f func(k, v unsafe.Pointer)) {
	var k, v C.gpointer
	var iter C.GHashTableIter
	C.g_hash_table_iter_init(&iter, (*C.GHashTable)(ptr))

	for C.g_hash_table_iter_next(&iter, &k, &v) != 0 {
		f(unsafe.Pointer(k), unsafe.Pointer(v))
	}

	if rm {
		C.g_hash_table_unref((*C.GHashTable)(ptr))
	}
}

// ListSize returns the length of the list.
func ListSize(ptr unsafe.Pointer) int {
	return int(C.g_list_length((*C.GList)(ptr)))
}

// MoveList calls f on every value of the given *GList. If rm is true, then the
// GList is freed.
func MoveList(ptr unsafe.Pointer, rm bool, f func(v unsafe.Pointer)) {
	for v := (*C.GList)(ptr); v != nil; v = v.next {
		f(unsafe.Pointer(v.data))
	}

	if rm {
		C.g_list_free((*C.GList)(ptr))
	}
}

// SListSize returns the length of the singly-linked list.
func SListSize(ptr unsafe.Pointer) int {
	return int(C.g_slist_length((*C.GSList)(ptr)))
}

// MoveSList is similar to MoveList, except it's used for singly-linked lists.
func MoveSList(ptr unsafe.Pointer, rm bool, f func(v unsafe.Pointer)) {
	for v := (*C.GSList)(ptr); v != nil; v = v.next {
		f(unsafe.Pointer(v.data))
	}

	if rm {
		C.g_slist_free((*C.GSList)(ptr))
	}
}
