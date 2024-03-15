package glib

// #include "glib.go.h"
import "C"

import (
	"log"
	"runtime"
	"unsafe"
)

/*
type paramPrototyper interface {
	paramSpec() *ParamSpec
}

type ParamPrototype[T any] struct {
	Name    string
	Nick    string
	Blurb   string
	Default T
	Flags   ParamFlags
}

type ParamNumberPrototype[T ~int | ~uint | ~int32 | ~uint32 | ~int64 | ~uint64] struct {
	Name    string
	Nick    string
	Blurb   string
	Min     T
	Max     T
	Default T
	Flags   ParamFlags
}

func (p *ParamNumberPrototype[T]) paramSpec() *ParamSpec {
	var z T
	switch reflect.TypeOf(z).Kind() {
	case reflect.Int, reflect.Int32:
		return NewIntParam(p.Name, p.Nick, p.Blurb, int32(p.Min), int32(p.Max), int32(p.Default), p.Flags)
	case reflect.Uint, reflect.Uint32:
	}
}
*/

// ParamSpec is a go representation of a C GParamSpec
type ParamSpec struct{ *paramSpec }

type paramSpec struct{ intern *C.GParamSpec }

func newParamSpecCommon(name, nick, blurb string) (cname, cnick, cblurb *C.gchar, free func()) {
	cname = (*C.gchar)(C.CString(name))
	cnick = (*C.gchar)(C.CString(nick))
	cblurb = (*C.gchar)(C.CString(blurb))
	free = func() {
		C.free(unsafe.Pointer(cname))
		C.free(unsafe.Pointer(cnick))
		C.free(unsafe.Pointer(cblurb))
	}

	if !gobool(C.g_param_spec_is_valid_name(cname)) {
		log.Panicf("invalid param spec name %q", name)
	}

	return
}

// NewStringParam returns a new ParamSpec that will hold a string value.
func NewStringParam(name, nick, blurb string, defaultValue string, flags ParamFlags) *ParamSpec {
	var cdefault *C.gchar
	if defaultValue != "" {
		cdefault = C.CString(defaultValue)
	}

	cname, cnick, cblurb, cfree := newParamSpecCommon(name, nick, blurb)
	defer cfree()

	paramSpec := C.g_param_spec_string(
		cname, cnick, cblurb,
		(*C.gchar)(cdefault),
		C.GParamFlags(flags),
	)
	return ParamSpecTake(unsafe.Pointer(paramSpec), false)
}

// NewBoolParam creates a new ParamSpec that will hold a boolean value.
func NewBoolParam(name, nick, blurb string, defaultValue bool, flags ParamFlags) *ParamSpec {
	cname, cnick, cblurb, cfree := newParamSpecCommon(name, nick, blurb)
	defer cfree()

	paramSpec := C.g_param_spec_boolean(
		cname, cnick, cblurb,
		gbool(defaultValue),
		C.GParamFlags(flags),
	)
	return ParamSpecTake(unsafe.Pointer(paramSpec), false)
}

// NewIntParam creates a new ParamSpec that will hold a signed integer value.
func NewIntParam(name, nick, blurb string, min, max, defaultValue int32, flags ParamFlags) *ParamSpec {
	cname, cnick, cblurb, cfree := newParamSpecCommon(name, nick, blurb)
	defer cfree()

	paramSpec := C.g_param_spec_int(
		cname, cnick, cblurb,
		C.gint(min),
		C.gint(max),
		C.gint(defaultValue),
		C.GParamFlags(flags),
	)
	return ParamSpecTake(unsafe.Pointer(paramSpec), false)
}

// NewUintParam creates a new ParamSpec that will hold an unsigned integer value.
func NewUintParam(name, nick, blurb string, min, max, defaultValue uint32, flags ParamFlags) *ParamSpec {
	cname, cnick, cblurb, cfree := newParamSpecCommon(name, nick, blurb)
	defer cfree()

	paramSpec := C.g_param_spec_uint(
		cname, cnick, cblurb,
		C.guint(min),
		C.guint(max),
		C.guint(defaultValue),
		C.GParamFlags(flags),
	)
	return ParamSpecTake(unsafe.Pointer(paramSpec), false)
}

// NewInt64Param creates a new ParamSpec that will hold a signed 64-bit integer value.
func NewInt64Param(name, nick, blurb string, min, max, defaultValue int64, flags ParamFlags) *ParamSpec {
	cname, cnick, cblurb, cfree := newParamSpecCommon(name, nick, blurb)
	defer cfree()

	paramSpec := C.g_param_spec_int64(
		cname, cnick, cblurb,
		C.gint64(min),
		C.gint64(max),
		C.gint64(defaultValue),
		C.GParamFlags(flags),
	)
	return ParamSpecTake(unsafe.Pointer(paramSpec), false)
}

// NewUint64Param creates a new ParamSpec that will hold an unsigned 64-bit integer value.
func NewUint64Param(name, nick, blurb string, min, max, defaultValue uint64, flags ParamFlags) *ParamSpec {
	cname, cnick, cblurb, cfree := newParamSpecCommon(name, nick, blurb)
	defer cfree()

	paramSpec := C.g_param_spec_uint64(
		cname, cnick, cblurb,
		C.guint64(min),
		C.guint64(max),
		C.guint64(defaultValue),
		C.GParamFlags(flags),
	)
	return ParamSpecTake(unsafe.Pointer(paramSpec), false)
}

// NewFloat32Param creates a new ParamSpec that will hold a 32-bit float value.
func NewFloat32Param(name, nick, blurb string, min, max, defaultValue float32, flags ParamFlags) *ParamSpec {
	cname, cnick, cblurb, cfree := newParamSpecCommon(name, nick, blurb)
	defer cfree()

	paramSpec := C.g_param_spec_float(
		cname, cnick, cblurb,
		C.gfloat(min),
		C.gfloat(max),
		C.gfloat(defaultValue),
		C.GParamFlags(flags),
	)
	return ParamSpecTake(unsafe.Pointer(paramSpec), false)
}

// NewFloat64Param creates a new ParamSpec that will hold a 64-bit float value.
func NewFloat64Param(name, nick, blurb string, min, max, defaultValue float64, flags ParamFlags) *ParamSpec {
	cname, cnick, cblurb, cfree := newParamSpecCommon(name, nick, blurb)
	defer cfree()

	paramSpec := C.g_param_spec_double(
		cname, cnick, cblurb,
		C.gdouble(min),
		C.gdouble(max),
		C.gdouble(defaultValue),
		C.GParamFlags(flags),
	)
	return ParamSpecTake(unsafe.Pointer(paramSpec), false)
}

// NewBoxedParam creates a new ParamSpec containing a boxed type.
func NewBoxedParam(name, nick, blurb string, boxedType Type, flags ParamFlags) *ParamSpec {
	cname, cnick, cblurb, cfree := newParamSpecCommon(name, nick, blurb)
	defer cfree()

	paramSpec := C.g_param_spec_boxed(
		cname, cnick, cblurb,
		C.GType(boxedType),
		C.GParamFlags(flags),
	)
	return ParamSpecTake(unsafe.Pointer(paramSpec), false)
}

// ParamSpecFromNative wraps ptr into a ParamSpec.
func ParamSpecFromNative(ptr unsafe.Pointer) *ParamSpec {
	return &ParamSpec{&paramSpec{(*C.GParamSpec)(ptr)}}
}

// ParamSpecTake wraps ptr into a ParamSpec and ensures that it's properly GC'd.
func ParamSpecTake(ptr unsafe.Pointer, take bool) *ParamSpec {
	p := ParamSpecFromNative(ptr)
	if !take {
		C.g_param_spec_ref(p.intern)
	}
	runtime.SetFinalizer(p.paramSpec, func(p *paramSpec) {
		C.g_param_spec_unref(p.intern)
	})
	return p
}

// Name returns the name of this parameter.
func (p *ParamSpec) Name() string {
	return C.GoString(C.g_param_spec_get_name(p.intern))
}

// Blurb returns the blurb for this parameter.
func (p *ParamSpec) Blurb() string {
	return C.GoString(C.g_param_spec_get_blurb(p.intern))
}

// Flags returns the flags for this parameter.
func (p *ParamSpec) Flags() ParamFlags {
	return ParamFlags(p.intern.flags)
}

// ValueType returns the GType for the value inside this parameter.
func (p *ParamSpec) ValueType() Type {
	return Type(p.intern.value_type)
}

// OwnerType returns the Gtype for the owner of this parameter.
func (p *ParamSpec) OwnerType() Type {
	return Type(p.intern.owner_type)
}

// Unref the underlying paramater spec.
func (p *ParamSpec) Unref() { C.g_param_spec_unref(p.intern) }

// ParamFlags is a go cast of GParamFlags.
type ParamFlags int

// Has returns true if these flags contain the provided ones.
func (p ParamFlags) Has(b ParamFlags) bool { return p&b != 0 }

const (
	ParamReadable       ParamFlags = C.G_PARAM_READABLE
	ParamWritable                  = C.G_PARAM_WRITABLE
	ParamReadWrite                 = C.G_PARAM_READABLE | C.G_PARAM_WRITABLE
	ParamConstruct                 = C.G_PARAM_CONSTRUCT
	ParamConstructOnly             = C.G_PARAM_CONSTRUCT_ONLY
	ParamLaxValidation             = C.G_PARAM_LAX_VALIDATION
	ParamStaticName                = C.G_PARAM_STATIC_NAME
	ParamStaticNick                = C.G_PARAM_STATIC_NICK
	ParamStaticBlurb               = C.G_PARAM_STATIC_BLURB
	ParamExplicitNotify            = C.G_PARAM_EXPLICIT_NOTIFY
	ParamDeprecated                = C.G_PARAM_DEPRECATED
)
