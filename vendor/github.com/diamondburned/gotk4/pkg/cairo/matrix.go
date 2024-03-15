package cairo

// #include <stdlib.h>
// #include <cairo.h>
// #include <cairo-gobject.h>
import "C"

import (
	"unsafe"
)

// Matrix struct
type Matrix struct {
	Xx, Yx float64
	Xy, Yy float64
	X0, Y0 float64
}

// NewMatrix creates a new identiy matrix
func NewMatrix(xx, yx, xy, yy, x0, y0 float64) *Matrix {
	return &Matrix{
		Xx: xx,
		Yx: yx,
		Xy: xy,
		Yy: yy,
		X0: x0,
		Y0: y0,
	}
}

// Native returns native c pointer to a matrix
func (m *Matrix) native() *C.cairo_matrix_t {
	return (*C.cairo_matrix_t)(unsafe.Pointer(m))
}

// Native returns native c pointer to a matrix
func (m *Matrix) Native() uintptr {
	return uintptr(unsafe.Pointer(m.native()))
}

// InitIdentity initializes this matrix to identity matrix
func (m *Matrix) InitIdentity() {
	C.cairo_matrix_init_identity(m.native())
}

// InitTranslate initializes a matrix with the given translation
func (m *Matrix) InitTranslate(tx, ty float64) {
	C.cairo_matrix_init_translate(m.native(), C.double(tx), C.double(ty))
}

// InitScale initializes a matrix with the give scale
func (m *Matrix) InitScale(sx, sy float64) {
	C.cairo_matrix_init_scale(m.native(), C.double(sx), C.double(sy))
}

// InitRotate initializes a matrix with the given rotation
func (m *Matrix) InitRotate(radians float64) {
	C.cairo_matrix_init_rotate(m.native(), C.double(radians))
}

// Translate translates a matrix by the given amount
func (m *Matrix) Translate(tx, ty float64) {
	C.cairo_matrix_translate(m.native(), C.double(tx), C.double(ty))
}

// Scale scales the matrix by the given amounts
func (m *Matrix) Scale(sx, sy float64) {
	C.cairo_matrix_scale(m.native(), C.double(sx), C.double(sy))
}

// Rotate rotates the matrix by the given amount
func (m *Matrix) Rotate(radians float64) {
	C.cairo_matrix_rotate(m.native(), C.double(radians))
}

// Invert inverts the matrix
func (m *Matrix) Invert() {
	C.cairo_matrix_invert(m.native())
}

// Multiply multiplies the matrix by another matrix
func (m *Matrix) Multiply(a, b Matrix) {
	C.cairo_matrix_multiply(m.native(), a.native(), b.native())
}

// TransformDistance ...
func (m *Matrix) TransformDistance(dx, dy float64) (float64, float64) {
	C.cairo_matrix_transform_distance(m.native(),
		(*C.double)(unsafe.Pointer(&dx)), (*C.double)(unsafe.Pointer(&dy)))
	return dx, dy
}

// TransformPoint ...
func (m *Matrix) TransformPoint(x, y float64) (float64, float64) {
	C.cairo_matrix_transform_point(m.native(),
		(*C.double)(unsafe.Pointer(&x)), (*C.double)(unsafe.Pointer(&y)))
	return x, y
}
