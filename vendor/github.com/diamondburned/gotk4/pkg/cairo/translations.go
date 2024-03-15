package cairo

// #include <stdlib.h>
// #include <cairo.h>
// #include <cairo-gobject.h>
import "C"

// Translate is a wrapper around cairo_translate.
func (v *Context) Translate(tx, ty float64) {
	C.cairo_translate(v.native(), C.double(tx), C.double(ty))
}

// Scale is a wrapper around cairo_scale.
func (v *Context) Scale(sx, sy float64) {
	C.cairo_scale(v.native(), C.double(sx), C.double(sy))
}

// Rotate is a wrapper around cairo_rotate.
func (v *Context) Rotate(angle float64) {
	C.cairo_rotate(v.native(), C.double(angle))
}

// Transform is a wrapper around cairo_transform.
func (v *Context) Transform(matrix *Matrix) {
	C.cairo_transform(v.native(), matrix.native())
}

// SetMatrix is a wrapper around cairo_set_matrix.
func (v *Context) SetMatrix(matrix *Matrix) {
	C.cairo_set_matrix(v.native(), matrix.native())
}

// GetMatrix is a wrapper around cairo_get_matrix.
func (v *Context) Matrix() *Matrix {
	var matrix C.cairo_matrix_t
	C.cairo_get_matrix(v.native(), &matrix)
	return &Matrix{
		Xx: float64(matrix.xx),
		Yx: float64(matrix.yx),
		Xy: float64(matrix.xy),
		Yy: float64(matrix.yy),
		X0: float64(matrix.x0),
		Y0: float64(matrix.y0),
	}
}

// IdentityMatrix is a wrapper around cairo_identity_matrix().
//
// Resets the current transformation matrix (CTM) by setting it equal to the
// identity matrix. That is, the user-space and device-space axes will be
// aligned and one user-space unit will transform to one device-space unit.
func (v *Context) IdentityMatrix() {
	C.cairo_identity_matrix(v.native())
}

// UserToDevice is a wrapper around cairo_user_to_device.
func (v *Context) UserToDevice(x, y float64) (float64, float64) {
	C.cairo_user_to_device(v.native(), (*C.double)(&x), (*C.double)(&y))
	return x, y
}

// UserToDeviceDistance is a wrapper around cairo_user_to_device_distance.
func (v *Context) UserToDeviceDistance(dx, dy float64) (float64, float64) {
	C.cairo_user_to_device_distance(v.native(), (*C.double)(&dx), (*C.double)(&dy))
	return dx, dy
}

// DeviceToUser  is a wrapper around cairo_device_to_user.
func (v *Context) DeviceToUser(x, y float64) (float64, float64) {
	C.cairo_device_to_user(v.native(), (*C.double)(&x), (*C.double)(&y))
	return x, y
}

// DeviceToUserDistance is a wrapper around cairo_device_to_user_distance.
func (v *Context) DeviceToUserDistance(x, y float64) (float64, float64) {
	C.cairo_device_to_user_distance(v.native(), (*C.double)(&x), (*C.double)(&y))
	return x, y
}
