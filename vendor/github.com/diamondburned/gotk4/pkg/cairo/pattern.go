package cairo

// #include <stdlib.h>
// #include <cairo.h>
// #include <cairo-gobject.h>
import "C"

import (
	"runtime"
	"unsafe"
)

//--------------------------------------------[ cairo_pattern_t  ==  Pattern ]--

// Filter is a representation of Cairo's cairo_filter_t.
type Filter int

const (
	FilterFast     Filter = C.CAIRO_FILTER_FAST
	FilterGood     Filter = C.CAIRO_FILTER_GOOD
	FilterBest     Filter = C.CAIRO_FILTER_BEST
	FilterNearest  Filter = C.CAIRO_FILTER_NEAREST
	FilterBilinear Filter = C.CAIRO_FILTER_BILINEAR
	FilterGaussian Filter = C.CAIRO_FILTER_GAUSSIAN
)

func marshalFilter(p uintptr) (interface{}, error) {
	c := C.g_value_get_enum((*C.GValue)(unsafe.Pointer(p)))
	return Filter(c), nil
}

// Pattern is a representation of Cairo's cairo_pattern_t.
type Pattern struct {
	pattern *C.cairo_pattern_t
}

// NewPatternFromRGB is a wrapper around cairo_pattern_create_rgb().
func NewPatternFromRGB(red, green, blue float64) (*Pattern, error) {
	c := C.cairo_pattern_create_rgb(C.double(red), C.double(green), C.double(blue))
	return newPatternFromNative(c)
}

// NewPatternFromRGBA is a wrapper around cairo_pattern_create_rgba().
func NewPatternFromRGBA(red, green, blue, alpha float64) (*Pattern, error) {
	c := C.cairo_pattern_create_rgba(C.double(red), C.double(green), C.double(blue), C.double(alpha))
	return newPatternFromNative(c)
}

// NewPatternForSurface is a wrapper around cairo_pattern_create_for_surface().
func NewPatternForSurface(s *Surface) (*Pattern, error) {
	c := C.cairo_pattern_create_for_surface(s.native())
	return newPatternFromNative(c)
}

// NewPatternLinear is a wrapper around cairo_pattern_create_linear().
func NewPatternLinear(x0, y0, x1, y1 float64) (*Pattern, error) {
	c := C.cairo_pattern_create_linear(C.double(x0), C.double(y0), C.double(x1), C.double(y1))
	return newPatternFromNative(c)
}

// NewPatternRadial is a wrapper around cairo_pattern_create_radial().
func NewPatternRadial(x0, y0, r0, x1, y1, r1 float64) (*Pattern, error) {
	c := C.cairo_pattern_create_radial(C.double(x0), C.double(y0), C.double(r0),
		C.double(x1), C.double(y1), C.double(r1))
	return newPatternFromNative(c)
}

func newPatternFromNative(patternNative *C.cairo_pattern_t) (*Pattern, error) {
	ptr := wrapPattern(patternNative)
	e := ptr.Status().ToError()
	if e != nil {
		return nil, e
	}
	runtime.SetFinalizer(ptr, (*Pattern).destroy)
	return ptr, nil
}

// native returns a pointer to the underlying cairo_pattern_t.
func (v *Pattern) native() *C.cairo_pattern_t {
	if v == nil {
		return nil
	}
	return v.pattern
}

// Native returns a pointer to the underlying cairo_pattern_t.
func (v *Pattern) Native() uintptr {
	return uintptr(unsafe.Pointer(v.native()))
}

func marshalPattern(p uintptr) (interface{}, error) {
	c := C.g_value_get_boxed((*C.GValue)(unsafe.Pointer(p)))
	pattern := (*C.cairo_pattern_t)(unsafe.Pointer(c))
	return wrapPattern(pattern), nil
}

func wrapPattern(pattern *C.cairo_pattern_t) *Pattern {
	return &Pattern{pattern}
}

// reference is a wrapper around cairo_pattern_reference().
func (v *Pattern) reference() {
	v.pattern = C.cairo_pattern_reference(v.native())
}

// destroy is a wrapper around cairo_pattern_destroy().
func (v *Pattern) destroy() {
	C.cairo_pattern_destroy(v.native())
}

// Status is a wrapper around cairo_pattern_status().
func (v *Pattern) Status() Status {
	c := C.cairo_pattern_status(v.native())
	return Status(c)
}

// AddColorStopRGB is a wrapper around cairo_pattern_add_color_stop_rgb().
func (v *Pattern) AddColorStopRGB(offset, red, green, blue float64) error {
	C.cairo_pattern_add_color_stop_rgb(v.native(), C.double(offset),
		C.double(red), C.double(green), C.double(blue))
	return v.Status().ToError()
}

// AddColorStopRGBA is a wrapper around cairo_pattern_add_color_stop_rgba().
func (v *Pattern) AddColorStopRGBA(offset, red, green, blue, alpha float64) error {
	C.cairo_pattern_add_color_stop_rgba(v.native(), C.double(offset),
		C.double(red), C.double(green), C.double(blue), C.double(alpha))
	return v.Status().ToError()
}

// PatternSetFilter is a wrapper around cairo_pattern_set_filter().
func (v *Pattern) PatternSetFilter(filter Filter) {
	C.cairo_pattern_set_filter(v.native(), C.cairo_filter_t(filter))
}

// PatternGetFilter is a wrapper around cairo_pattern_get_filter().
func (v *Pattern) PatternGetFilter() Filter {
	return Filter(C.cairo_pattern_get_filter(v.native()))
}
