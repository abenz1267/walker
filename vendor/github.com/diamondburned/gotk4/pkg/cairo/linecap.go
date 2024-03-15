package cairo

// #include <stdlib.h>
// #include <cairo.h>
// #include <cairo-gobject.h>
import "C"
import (
	"unsafe"
)

// LineCap is a representation of Cairo's cairo_line_cap_t.
type LineCap int

const (
	LineCapButt   LineCap = C.CAIRO_LINE_CAP_BUTT
	LineCapRound  LineCap = C.CAIRO_LINE_CAP_ROUND
	LineCapSquare LineCap = C.CAIRO_LINE_CAP_SQUARE
)

func marshalLineCap(p uintptr) (interface{}, error) {
	c := C.g_value_get_enum((*C.GValue)(unsafe.Pointer(p)))
	return LineCap(c), nil
}
