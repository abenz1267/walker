package cairo

// #include <stdlib.h>
// #include <cairo.h>
// #include <cairo-gobject.h>
import "C"
import (
	"unsafe"
)

// Format is a representation of Cairo's cairo_format_t.
type Format int

const (
	FormatInvalid   Format = C.CAIRO_FORMAT_INVALID
	FormatARGB32    Format = C.CAIRO_FORMAT_ARGB32
	FormatRGB24     Format = C.CAIRO_FORMAT_RGB24
	FormatA8        Format = C.CAIRO_FORMAT_A8
	FormatA1        Format = C.CAIRO_FORMAT_A1
	FormatRGB16_565 Format = C.CAIRO_FORMAT_RGB16_565
	FormatRGB30     Format = C.CAIRO_FORMAT_RGB30
)

func marshalFormat(p uintptr) (interface{}, error) {
	c := C.g_value_get_enum((*C.GValue)(unsafe.Pointer(p)))
	return Format(c), nil
}

// FormatStrideForWidth is a wrapper for cairo_format_stride_for_width().
func FormatStrideForWidth(format Format, width int) int {
	c := C.cairo_format_stride_for_width(C.cairo_format_t(format), C.int(width))
	return int(c)
}
