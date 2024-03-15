package glib

// #include "glib.go.h"
import "C"

import (
	"runtime"
	"unsafe"
)

// A Variant is a representation of GLib's GVariant.
type Variant struct {
	GVariant *C.GVariant
}

// native returns a pointer to the underlying GVariant.
func (v *Variant) native() *C.GVariant {
	if v == nil || v.GVariant == nil {
		return nil
	}
	return v.GVariant
}

// Native returns a pointer to the underlying GVariant.
func (v *Variant) Native() uintptr {
	return uintptr(unsafe.Pointer(v.native()))
}

// newVariant wraps a native GVariant.
// Does NOT handle reference counting! Use takeVariant() to take ownership of values.
func newVariant(p *C.GVariant) *Variant {
	if p == nil {
		return nil
	}
	return &Variant{GVariant: p}
}

// takeVariant wraps a native GVariant,
// takes ownership and sets up a finalizer to free the instance during GC.
func takeVariant(p *C.GVariant) *Variant {
	if p == nil {
		return nil
	}
	obj := &Variant{GVariant: p}

	if obj.IsFloating() {
		obj.RefSink()
	} else {
		obj.Ref()
	}

	runtime.SetFinalizer(obj, (*Variant).Unref)
	return obj
}

// IsFloating returns true if the variant has a floating reference count.
// Reference counting is usually handled in the gotk layer,
// most applications should not call this.
func (v *Variant) IsFloating() bool {
	b := gobool(C.g_variant_is_floating(v.native()))
	runtime.KeepAlive(v)
	return b
}

// Ref is a wrapper around g_variant_ref.
// Reference counting is usually handled in the gotk layer,
// most applications should not need to call this.
func (v *Variant) Ref() {
	C.g_variant_ref(v.native())
	runtime.KeepAlive(v)
}

// RefSink is a wrapper around g_variant_ref_sink.
// Reference counting is usually handled in the gotk layer,
// most applications should not need to call this.
func (v *Variant) RefSink() {
	C.g_variant_ref_sink(v.native())
	runtime.KeepAlive(v)
}

// TakeRef is a wrapper around g_variant_take_ref.
// Reference counting is usually handled in the gotk layer,
// most applications should not need to call this.
func (v *Variant) TakeRef() {
	C.g_variant_take_ref(v.native())
	runtime.KeepAlive(v)
}

// Unref is a wrapper around g_variant_unref.
// Reference counting is usually handled in the gotk layer,
// most applications should not need to call this.
func (v *Variant) Unref() {
	C.g_variant_unref(v.native())
	runtime.KeepAlive(v)
}

// TypeString returns the g variant type string for this variant.
func (v *Variant) TypeString() string {
	// the string returned from this belongs to GVariant and must not be freed.
	s := C.GoString((*C.char)(C.g_variant_get_type_string(v.native())))
	runtime.KeepAlive(v)
	return s
}

// IsContainer returns true if the variant is a container and false otherwise.
func (v *Variant) IsContainer() bool {
	b := gobool(C.g_variant_is_container(v.native()))
	runtime.KeepAlive(v)
	return b
}

// String is a wrapper around g_variant_get_string. If the Variant type is not a
// string, then Print is called instead. This is done to satisfy fmt.Stringer;
// it behaves similarly to reflect.Value.String().
func (v *Variant) String() string {
	defer runtime.KeepAlive(v)

	if C.g_variant_get_type(v.native()) != C.G_VARIANT_TYPE_STRING {
		return v.Print(false)
	}

	// The string value remains valid as long as the GVariant exists, do NOT free the cstring in this function.
	var len C.gsize
	gc := C.g_variant_get_string(v.native(), &len)

	// This is opposed to g_variant_dup_string, which copies the string.
	// g_variant_dup_string is not implemented,
	// as we copy the string value anyways when converting to a go string.

	return C.GoStringN((*C.char)(gc), (C.int)(len))
}

// Type returns the VariantType for this variant.
func (v *Variant) Type() *VariantType {
	// The return value is valid for the lifetime of value and must not be freed.
	t := newVariantType(C.g_variant_get_type(v.native()))
	runtime.KeepAlive(v)
	return t
}

// IsType returns true if the variant's type matches t.
func (v *Variant) IsType(t *VariantType) bool {
	b := gobool(C.g_variant_is_of_type(v.native(), t.native()))
	runtime.KeepAlive(v)
	runtime.KeepAlive(t)
	return b
}

// Print wraps g_variant_print(). It returns a string understood by
// g_variant_parse().
func (v *Variant) Print(typeAnnotate bool) string {
	gc := C.g_variant_print(v.native(), gbool(typeAnnotate))
	runtime.KeepAlive(v)

	defer C.g_free(C.gpointer(gc))

	return C.GoString((*C.char)(gc))
}

// GoValue converts the variant's value to the Go value. It only supports the
// following types for now:
//
//	s: string
//	b: bool
//	d: float64
//	n: int16
//	i: int32
//	x: int64
//	y: byte
//	q: uint16
//	u: uint32
//	t: uint64
//
// Variants with unsupported types will cause the function to return nil.
func (v *Variant) GoValue() interface{} {
	var val interface{}
	switch C.g_variant_get_type(v.native()) {
	case C.G_VARIANT_TYPE_STRING:
		val = v.String()
	case C.G_VARIANT_TYPE_BOOLEAN:
		val = gobool(C.g_variant_get_boolean(v.native()))
	case C.G_VARIANT_TYPE_DOUBLE:
		val = float64(C.g_variant_get_double(v.native()))
	case C.G_VARIANT_TYPE_INT16:
		val = int16(C.g_variant_get_int16(v.native()))
	case C.G_VARIANT_TYPE_INT32:
		val = int32(C.g_variant_get_int32(v.native()))
	case C.G_VARIANT_TYPE_INT64:
		val = int64(C.g_variant_get_int64(v.native()))
	case C.G_VARIANT_TYPE_BYTE:
		val = uint8(C.g_variant_get_byte(v.native()))
	case C.G_VARIANT_TYPE_UINT16:
		val = uint16(C.g_variant_get_uint16(v.native()))
	case C.G_VARIANT_TYPE_UINT32:
		val = uint32(C.g_variant_get_uint32(v.native()))
	case C.G_VARIANT_TYPE_UINT64:
		val = uint64(C.g_variant_get_uint64(v.native()))
	}

	runtime.KeepAlive(v)
	return val
}

// A VariantType is a wrapper for the GVariantType, which encodes type
// information for GVariants.
type VariantType struct {
	GVariantType *C.GVariantType
}

func (v *VariantType) native() *C.GVariantType {
	return v.GVariantType
}

func (v *VariantType) Native() uintptr {
	if v == nil || v.GVariantType == nil {
		return uintptr(unsafe.Pointer(nil))
	}
	return uintptr(unsafe.Pointer(v.native()))
}

// String returns a copy of this VariantType's type string.
func (v *VariantType) String() string {
	ch := C.g_variant_type_dup_string(v.native())
	runtime.KeepAlive(v)

	defer C.g_free(C.gpointer(ch))
	return C.GoString((*C.char)(ch))
}

// newVariantType wraps a native GVariantType.
// Does not create a finalizer.
// Use takeVariantType for instances which need to be freed after use.
func newVariantType(v *C.GVariantType) *VariantType {
	if v == nil {
		return nil
	}
	return &VariantType{v}
}

// takeVariantType wraps a native GVariantType
// and sets up a finalizer to free the instance during GC.
func takeVariantType(v *C.GVariantType) *VariantType {
	if v == nil {
		return nil
	}
	obj := &VariantType{v}
	runtime.SetFinalizer(obj, (*VariantType).Free)
	return obj
}

// Variant types for comparison. Note that variant types cannot be compared by
// value; use VariantType.Equal instead.
var (
	VariantTypeBoolean         = newVariantType(C.G_VARIANT_TYPE_BOOLEAN)
	VariantTypeByte            = newVariantType(C.G_VARIANT_TYPE_BYTE)
	VariantTypeInt16           = newVariantType(C.G_VARIANT_TYPE_INT16)
	VariantTypeUint16          = newVariantType(C.G_VARIANT_TYPE_UINT16)
	VariantTypeInt32           = newVariantType(C.G_VARIANT_TYPE_INT32)
	VariantTypeUint32          = newVariantType(C.G_VARIANT_TYPE_UINT32)
	VariantTypeInt64           = newVariantType(C.G_VARIANT_TYPE_INT64)
	VariantTypeUint64          = newVariantType(C.G_VARIANT_TYPE_UINT64)
	VariantTypeHandle          = newVariantType(C.G_VARIANT_TYPE_HANDLE)
	VariantTypeDouble          = newVariantType(C.G_VARIANT_TYPE_DOUBLE)
	VariantTypeString          = newVariantType(C.G_VARIANT_TYPE_STRING)
	VariantTypeObjectPath      = newVariantType(C.G_VARIANT_TYPE_OBJECT_PATH)
	VariantTypeSignature       = newVariantType(C.G_VARIANT_TYPE_SIGNATURE)
	VariantTypeVariant         = newVariantType(C.G_VARIANT_TYPE_VARIANT)
	VariantTypeAny             = newVariantType(C.G_VARIANT_TYPE_ANY)
	VariantTypeBasic           = newVariantType(C.G_VARIANT_TYPE_BASIC)
	VariantTypeMaybe           = newVariantType(C.G_VARIANT_TYPE_MAYBE)
	VariantTypeArray           = newVariantType(C.G_VARIANT_TYPE_ARRAY)
	VariantTypeTuple           = newVariantType(C.G_VARIANT_TYPE_TUPLE)
	VariantTypeUnit            = newVariantType(C.G_VARIANT_TYPE_UNIT)
	VariantTypeDictEntry       = newVariantType(C.G_VARIANT_TYPE_DICT_ENTRY)
	VariantTypeDictionary      = newVariantType(C.G_VARIANT_TYPE_DICTIONARY)
	VariantTypeStringArray     = newVariantType(C.G_VARIANT_TYPE_STRING_ARRAY)
	VariantTypeObjectPathArray = newVariantType(C.G_VARIANT_TYPE_OBJECT_PATH_ARRAY)
	VariantTypeBytestring      = newVariantType(C.G_VARIANT_TYPE_BYTESTRING)
	VariantTypeBytestringArray = newVariantType(C.G_VARIANT_TYPE_BYTESTRING_ARRAY)
	VariantTypeVardict         = newVariantType(C.G_VARIANT_TYPE_VARDICT)
)

// Free is a wrapper around g_variant_type_free.
// Reference counting is usually handled in the gotk layer,
// most applications should not call this.
func (v *VariantType) Free() {
	C.g_variant_type_free(v.native())
}

// NewVariantType is a wrapper around g_variant_type_new.
func NewVariantTypeNew(typeString string) *VariantType {
	cstr := (*C.gchar)(C.CString(typeString))
	defer C.free(unsafe.Pointer(cstr))

	c := C.g_variant_type_new(cstr)
	return takeVariantType(c)
}

// VariantTypeStringIsValid is a wrapper around g_variant_type_string_is_valid.
func VariantTypeStringIsValid(typeString string) bool {
	cstr := (*C.gchar)(C.CString(typeString))
	defer C.free(unsafe.Pointer(cstr))

	return gobool(C.g_variant_type_string_is_valid(cstr))
}

// Equal is a wrapper around g_variant_type_equal.
func (v *VariantType) Equal(to *VariantType) bool {
	b := gobool(C.g_variant_type_equal(C.gconstpointer(v.native()), C.gconstpointer(to.native())))
	runtime.KeepAlive(v)
	return b
}

// IsSubtypeOf is a wrapper around g_variant_type_is_subtype_of.
func (v *VariantType) IsSubtypeOf(supertype *VariantType) bool {
	b := gobool(C.g_variant_type_is_subtype_of(v.native(), supertype.native()))
	runtime.KeepAlive(v)
	return b
}
