package cairo

// #include <stdlib.h>
// #include <cairo.h>
// #include <cairo-gobject.h>
import "C"
import (
	"unsafe"
)

// LineJoin is a representation of Cairo's cairo_line_join_t.
type LineJoin int

const (
	LineJoinMiter LineJoin = C.CAIRO_LINE_JOIN_MITER
	LineJoinRound LineJoin = C.CAIRO_LINE_JOIN_ROUND
	LineJoinBevel LineJoin = C.CAIRO_LINE_JOIN_BEVEL
)

func marshalLineJoin(p uintptr) (interface{}, error) {
	c := C.g_value_get_enum((*C.GValue)(unsafe.Pointer(p)))
	return LineJoin(c), nil
}
