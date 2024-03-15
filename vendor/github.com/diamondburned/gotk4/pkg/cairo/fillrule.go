package cairo

// #include <stdlib.h>
// #include <cairo.h>
// #include <cairo-gobject.h>
import "C"
import (
	"unsafe"
)

// FillRule is a representation of Cairo's cairo_fill_rule_t.
type FillRule int

const (
	FillRuleWinding FillRule = C.CAIRO_FILL_RULE_WINDING
	FillRuleEvenOdd FillRule = C.CAIRO_FILL_RULE_EVEN_ODD
)

func marshalFillRule(p uintptr) (interface{}, error) {
	c := C.g_value_get_enum((*C.GValue)(unsafe.Pointer(p)))
	return FillRule(c), nil
}
