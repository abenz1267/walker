package cairo

// #include <stdlib.h>
// #include <cairo.h>
// #include <cairo-gobject.h>
import "C"
import (
	"unsafe"
)

// Operator is a representation of Cairo's cairo_operator_t.
type Operator int

const (
	OperatorClear         Operator = C.CAIRO_OPERATOR_CLEAR
	OperatorSource        Operator = C.CAIRO_OPERATOR_SOURCE
	OperatorOver          Operator = C.CAIRO_OPERATOR_OVER
	OperatorIn            Operator = C.CAIRO_OPERATOR_IN
	OperatorOut           Operator = C.CAIRO_OPERATOR_OUT
	OperatorAtop          Operator = C.CAIRO_OPERATOR_ATOP
	OperatorDest          Operator = C.CAIRO_OPERATOR_DEST
	OperatorDestOver      Operator = C.CAIRO_OPERATOR_DEST_OVER
	OperatorDestIn        Operator = C.CAIRO_OPERATOR_DEST_IN
	OperatorDestOut       Operator = C.CAIRO_OPERATOR_DEST_OUT
	OperatorDestAtop      Operator = C.CAIRO_OPERATOR_DEST_ATOP
	OperatorXOR           Operator = C.CAIRO_OPERATOR_XOR
	OperatorAdd           Operator = C.CAIRO_OPERATOR_ADD
	OperatorSaturate      Operator = C.CAIRO_OPERATOR_SATURATE
	OperatorMultiply      Operator = C.CAIRO_OPERATOR_MULTIPLY
	OperatorScreen        Operator = C.CAIRO_OPERATOR_SCREEN
	OperatorOverlay       Operator = C.CAIRO_OPERATOR_OVERLAY
	OperatorDarken        Operator = C.CAIRO_OPERATOR_DARKEN
	OperatorLighten       Operator = C.CAIRO_OPERATOR_LIGHTEN
	OperatorColorDodge    Operator = C.CAIRO_OPERATOR_COLOR_DODGE
	OperatorColorBurn     Operator = C.CAIRO_OPERATOR_COLOR_BURN
	OperatorHardLight     Operator = C.CAIRO_OPERATOR_HARD_LIGHT
	OperatorSoftLight     Operator = C.CAIRO_OPERATOR_SOFT_LIGHT
	OperatorDifference    Operator = C.CAIRO_OPERATOR_DIFFERENCE
	OperatorExclusion     Operator = C.CAIRO_OPERATOR_EXCLUSION
	OperatorHSLHue        Operator = C.CAIRO_OPERATOR_HSL_HUE
	OperatorHSLSaturation Operator = C.CAIRO_OPERATOR_HSL_SATURATION
	OperatorHSLColor      Operator = C.CAIRO_OPERATOR_HSL_COLOR
	OperatorHSLLuminosity Operator = C.CAIRO_OPERATOR_HSL_LUMINOSITY
)

func marshalOperator(p uintptr) (interface{}, error) {
	c := C.g_value_get_enum((*C.GValue)(unsafe.Pointer(p)))
	return Operator(c), nil
}
