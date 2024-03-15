package cairo

// #include <stdlib.h>
// #include <cairo.h>
// #include <cairo-gobject.h>
import "C"
import (
	"unsafe"
)

// Antialias is a representation of Cairo's cairo_antialias_t.
type Antialias int

const (
	AntialiasDefault  Antialias = C.CAIRO_ANTIALIAS_DEFAULT
	AntialiasNone     Antialias = C.CAIRO_ANTIALIAS_NONE
	AntialiasGray     Antialias = C.CAIRO_ANTIALIAS_GRAY
	AntialiasSubpixel Antialias = C.CAIRO_ANTIALIAS_SUBPIXEL
	AntialiasFast     Antialias = C.CAIRO_ANTIALIAS_FAST // (since 1.12)
	AntialiasGood     Antialias = C.CAIRO_ANTIALIAS_GOOD // (since 1.12)
	AntialiasBest     Antialias = C.CAIRO_ANTIALIAS_BEST // (since 1.12)
)

func marshalAntialias(p uintptr) (interface{}, error) {
	c := C.g_value_get_enum((*C.GValue)(unsafe.Pointer(p)))
	return Antialias(c), nil
}
