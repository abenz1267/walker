// region.go

package cairo

// #include <cairo.h>
// #include <cairo-gobject.h>
import "C"

import (
	"runtime"
	"unsafe"

	"github.com/diamondburned/gotk4/pkg/core/glib"
)

func init() {
	tm := []glib.TypeMarshaler{
		// Enums
		{glib.Type(C.cairo_gobject_region_overlap_get_type()), marshalRegionOverlap},

		// Boxed
		{glib.Type(C.cairo_gobject_region_get_type()), marshalRegion},
	}
	glib.RegisterGValueMarshalers(tm)
}

// RegionOverlap is a representation of Cairo's cairo_region_overlap_t.
type RegionOverlap int

const (
	RegionOverlapIn   RegionOverlap = C.CAIRO_REGION_OVERLAP_IN
	RegionOverlapOut  RegionOverlap = C.CAIRO_REGION_OVERLAP_OUT
	RegionOverlapPart RegionOverlap = C.CAIRO_REGION_OVERLAP_PART
)

func marshalRegionOverlap(p uintptr) (interface{}, error) {
	c := C.g_value_get_enum((*C.GValue)(unsafe.Pointer(p)))
	return RegionOverlap(c), nil
}

/*
 * Rectangle
 */

// Rectangle is a representation of Cairo's cairo_rectangle_int_t.
type Rectangle struct {
	X, Y          int
	Width, Height int
}

// commodity function to ceate Rectangle cairo object.
func RectangleNew(x, y, width, height int) *Rectangle {
	r := new(Rectangle)
	r.X = x
	r.Y = y
	r.Width = width
	r.Height = height
	return r
}

func (v *Rectangle) native() *C.cairo_rectangle_int_t {
	r := new(C.cairo_rectangle_int_t)
	r.x = C.int(v.X)
	r.y = C.int(v.Y)
	r.width = C.int(v.Width)
	r.height = C.int(v.Height)
	return r
}

func toRectangle(cr *C.cairo_rectangle_int_t) *Rectangle {
	return &Rectangle{
		X: int(cr.x), Y: int(cr.y),
		Width: int(cr.width), Height: int(cr.height)}
}

/*
 * Region
 */

// Region is a representation of Cairo's cairo_region_t.
type Region struct {
	region *C.cairo_region_t
}

// native returns a pointer to the underlying cairo_region_t.
func (v *Region) native() *C.cairo_region_t {
	if v == nil {
		return nil
	}
	return v.region
}

// Native returns a pointer to the underlying cairo_region_t.
func (v *Region) Native() uintptr {
	return uintptr(unsafe.Pointer(v.native()))
}

func marshalRegion(p uintptr) (interface{}, error) {
	c := C.g_value_get_boxed((*C.GValue)(unsafe.Pointer(p)))
	region := (*C.cairo_region_t)(unsafe.Pointer(c))
	return wrapRegion(region), nil
}

func wrapRegion(region *C.cairo_region_t) *Region {
	return &Region{region}
}

// newRegionFromNative that handle finalizer.
func newRegionFromNative(regionNative *C.cairo_region_t) (*Region, error) {
	ptr := wrapRegion(regionNative)
	e := ptr.Status().ToError()
	if e != nil {
		return nil, e
	}
	runtime.SetFinalizer(ptr, (*Region).destroy)
	return ptr, nil
}

// RegionCreate is a wrapper around cairo_region_create().
func RegionCreate() (*Region, error) {

	return newRegionFromNative(C.cairo_region_create())
}

// CreateRectangle is a wrapper around cairo_region_create_rectangle().
func (v *Region) CreateRectangle(rectangle *Rectangle) (*Region, error) {

	return newRegionFromNative(C.cairo_region_create_rectangle(
		rectangle.native()))
}

// CreateRectangles is a wrapper around cairo_region_create_rectangles().
func (v *Region) CreateRectangles(rectangles ...*Rectangle) (*Region, error) {

	length := len(rectangles)

	cRectangles := make([]C.cairo_rectangle_int_t, length)

	for i := 0; i < length; i++ {
		cRectangles[i] = *rectangles[i].native()
	}

	pRect := &cRectangles[0]

	return newRegionFromNative(
		C.cairo_region_create_rectangles(
			pRect,
			C.int(length)))
}

// Copy is a wrapper around cairo_region_copy().
func (v *Region) Copy() (*Region, error) {

	return newRegionFromNative(C.cairo_region_copy(v.native()))
}

// reference is a wrapper around cairo_region_reference().
func (v *Region) reference() {
	v.region = C.cairo_region_reference(v.native())
}

// destroy is a wrapper around cairo_region_destroy().
func (v *Region) destroy() {
	C.cairo_region_destroy(v.native())
}

// Status is a wrapper around cairo_region_status().
func (v *Region) Status() Status {
	c := C.cairo_region_status(v.native())
	return Status(c)
}

// GetExtents is a wrapper around cairo_region_get_extents().
func (v *Region) Extents(extents *Rectangle) {

	C.cairo_region_get_extents(v.native(), extents.native())
}

// NumRectangles is a wrapper around cairo_region_num_rectangles().
func (v *Region) NumRectangles() int {

	return int(C.cairo_region_num_rectangles(v.native()))
}

// GetRectangle is a wrapper around cairo_region_get_rectangle().
func (v *Region) Rectangle(nth int) *Rectangle {

	cr := new(C.cairo_rectangle_int_t)
	C.cairo_region_get_rectangle(v.native(), C.int(nth), cr)

	return toRectangle(cr)
}

// IsEmpty is a wrapper around cairo_region_is_empty().
func (v *Region) IsEmpty() bool {

	return gobool(C.cairo_region_is_empty(v.native()))
}

// ContainsPoint is a wrapper around cairo_region_contains_point().
func (v *Region) ContainsPoint(x, y int) bool {

	return gobool(C.cairo_region_contains_point(
		v.native(), C.int(x), C.int(y)))
}

// ContainsRectangle is a wrapper around cairo_region_contains_rectangle().
func (v *Region) ContainsRectangle(rectangle *Rectangle) RegionOverlap {

	return RegionOverlap(
		C.cairo_region_contains_rectangle(
			v.native(), rectangle.native()))
}

// Equal is a wrapper around cairo_region_equal().
func (v *Region) Equal(region *Region) bool {

	return gobool(C.cairo_region_equal(v.native(), region.native()))
}

// Translate is a wrapper around cairo_region_translate().
func (v *Region) Translate(dx, dy int) {

	C.cairo_region_translate(v.native(), C.int(dx), C.int(dy))
}

// Intersect is a wrapper around cairo_region_intersect().
// Note: contrary to the original statement, the source
// 'Region' remains preserved.
func (v *Region) Intersect(other *Region) (*Region, error) {

	dst, err := v.Copy()
	if err != nil {
		return nil, err
	}
	err = Status(
		C.cairo_region_intersect(
			dst.native(),
			other.native())).ToError()
	if err != nil {
		return nil, err
	}

	return dst, nil
}

// IntersectRectangle is a wrapper around cairo_region_intersect_rectangle().
// Note: contrary to the original statement, the source 'Region' remains preserved.
func (v *Region) IntersectRectangle(rectangle *Rectangle) (*Region, error) {

	dst, err := v.Copy()
	if err != nil {
		return nil, err
	}
	err = Status(
		C.cairo_region_intersect_rectangle(
			dst.native(),
			rectangle.native())).ToError()
	if err != nil {
		return nil, err
	}

	return dst, nil
}

// Substract is a wrapper around cairo_region_subtract().
// Note: contrary to the original statement, the source
// 'Region' remains preserved.
func (v *Region) Substract(other *Region) (*Region, error) {

	dst, err := v.Copy()
	if err != nil {
		return nil, err
	}
	err = Status(
		C.cairo_region_subtract(
			dst.native(),
			other.native())).ToError()
	if err != nil {
		return nil, err
	}

	return dst, nil
}

// SubstractRectangle is a wrapper around cairo_region_subtract_rectangle().
// Note: contrary to the original statement, the source 'Region' remains preserved.
func (v *Region) SubstractRectangle(rectangle *Rectangle) (*Region, error) {

	dst, err := v.Copy()
	if err != nil {
		return nil, err
	}
	err = Status(
		C.cairo_region_subtract_rectangle(
			dst.native(),
			rectangle.native())).ToError()
	if err != nil {
		return nil, err
	}

	return dst, nil
}

// Union is a wrapper around cairo_region_union().
// Note: contrary to the original statement, the source
// 'Region' remains preserved.
func (v *Region) Union(other *Region) (*Region, error) {

	dst, err := v.Copy()
	if err != nil {
		return nil, err
	}
	err = Status(
		C.cairo_region_union(
			dst.native(),
			other.native())).ToError()
	if err != nil {
		return nil, err
	}

	return dst, nil
}

// UnionRectangle is a wrapper around cairo_region_union_rectangle().
// Note: contrary to the original statement, the source 'Region' remains preserved.
func (v *Region) UnionRectangle(rectangle *Rectangle) (*Region, error) {

	dst, err := v.Copy()
	if err != nil {
		return nil, err
	}
	err = Status(
		C.cairo_region_union_rectangle(
			dst.native(),
			rectangle.native())).ToError()
	if err != nil {
		return nil, err
	}

	return dst, nil
}

// Xor is a wrapper around cairo_region_xor().
// Note: contrary to the original statement, the source
// 'Region' remains preserved.
func (v *Region) Xor(other *Region) (*Region, error) {

	dst, err := v.Copy()
	if err != nil {
		return nil, err
	}
	err = Status(
		C.cairo_region_xor(
			dst.native(),
			other.native())).ToError()
	if err != nil {
		return nil, err
	}

	return dst, nil
}

// XorRectangle is a wrapper around cairo_region_xor_rectangle().
// Note: contrary to the original statement, the source 'Region' remains preserved.
func (v *Region) XorRectangle(rectangle *Rectangle) (*Region, error) {

	dst, err := v.Copy()
	if err != nil {
		return nil, err
	}
	err = Status(
		C.cairo_region_xor_rectangle(
			dst.native(),
			rectangle.native())).ToError()
	if err != nil {
		return nil, err
	}

	return dst, nil
}
