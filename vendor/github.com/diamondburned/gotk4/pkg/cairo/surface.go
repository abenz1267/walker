package cairo

// #include <stdlib.h>
// #include <cairo.h>
// #include <cairo-gobject.h>
// #include <cairo-pdf.h>
// cairo_status_t _gotk4_cairo_write_func(void*, unsigned char*, unsigned int);
import "C"

import (
	"io"
	"runtime"
	"sync"
	"unsafe"

	"github.com/diamondburned/gotk4/pkg/core/gbox"
)

/*
 * cairo_surface_t
 */

// Surface is a representation of Cairo's cairo_surface_t.
type Surface struct {
	_       [0]sync.Mutex
	surface *C.cairo_surface_t
}

// CreatePDFSurface is a wrapper around cairo_pdf_surface_create().
func CreatePDFSurface(fileName string, width float64, height float64) (*Surface, error) {
	cstr := C.CString(fileName)
	defer C.free(unsafe.Pointer(cstr))

	surfaceNative := C.cairo_pdf_surface_create(cstr, C.double(width), C.double(height))

	status := Status(C.cairo_surface_status(surfaceNative))
	if status != StatusSuccess {
		return nil, status
	}

	s := wrapSurface(surfaceNative)

	runtime.SetFinalizer(s, (*Surface).destroy)

	return s, nil
}

// native returns a pointer to the underlying cairo_surface_t.
func (v *Surface) native() *C.cairo_surface_t {
	if v == nil {
		return nil
	}
	return v.surface
}

// Native returns a pointer to the underlying cairo_surface_t.
func (v *Surface) Native() uintptr {
	return uintptr(unsafe.Pointer(v.native()))
}

func marshalSurface(p uintptr) (interface{}, error) {
	c := C.g_value_get_boxed((*C.GValue)(unsafe.Pointer(p)))
	return WrapSurface(uintptr(c)), nil
}

func wrapSurface(surface *C.cairo_surface_t) *Surface {
	return &Surface{surface: surface}
}

// NewSurface creates a gotk3 cairo Surface from a pointer to a
// C cairo_surface_t.  This is primarily designed for use with other
// gotk3 packages and should be avoided by applications.
func NewSurface(s uintptr, needsRef bool) *Surface {
	surface := WrapSurface(s)
	if needsRef {
		surface.reference()
	}
	runtime.SetFinalizer(surface, (*Surface).destroy)
	return surface
}

func WrapSurface(s uintptr) *Surface {
	ptr := (*C.cairo_surface_t)(unsafe.Pointer(s))
	return wrapSurface(ptr)
}

// Closes the surface. The surface must not be used afterwards.
func (v *Surface) Close() {
	v.destroy()
}

// CreateSimilar is a wrapper around cairo_surface_create_similar().
func (v *Surface) CreateSimilar(content Content, width, height int) *Surface {
	c := C.cairo_surface_create_similar(v.native(),
		C.cairo_content_t(content), C.int(width), C.int(height))
	s := wrapSurface(c)
	runtime.SetFinalizer(s, (*Surface).destroy)
	return s
}

// TODO cairo_surface_create_similar_image (since 1.12)

// CreateForRectangle is a wrapper around cairo_surface_create_for_rectangle().
func (v *Surface) CreateForRectangle(x, y, width, height float64) *Surface {
	c := C.cairo_surface_create_for_rectangle(v.native(), C.double(x),
		C.double(y), C.double(width), C.double(height))
	s := wrapSurface(c)
	runtime.SetFinalizer(s, (*Surface).destroy)
	return s
}

// reference is a wrapper around cairo_surface_reference().
func (v *Surface) reference() {
	v.surface = C.cairo_surface_reference(v.native())
}

// destroy is a wrapper around cairo_surface_destroy().
func (v *Surface) destroy() {
	if v.surface != nil {
		C.cairo_surface_destroy(v.native())
		v.surface = nil
	}
}

// Status is a wrapper around cairo_surface_status().
func (v *Surface) Status() Status {
	c := C.cairo_surface_status(v.native())
	return Status(c)
}

// Flush is a wrapper around cairo_surface_flush().
func (v *Surface) Flush() {
	C.cairo_surface_flush(v.native())
}

// TODO(jrick) Device (requires Device bindings)
// cairo_surface_get_device

// TODO(jrick) FontOptions (require FontOptions bindings)
// cairo_surface_get_font_options

// TODO(jrick) Content (requires Content bindings)
// cairo_surface_get_content

// MarkDirty is a wrapper around cairo_surface_mark_dirty().
func (v *Surface) MarkDirty() {
	C.cairo_surface_mark_dirty(v.native())
}

// MarkDirtyRectangle is a wrapper around cairo_surface_mark_dirty_rectangle().
func (v *Surface) MarkDirtyRectangle(x, y, width, height int) {
	C.cairo_surface_mark_dirty_rectangle(v.native(), C.int(x), C.int(y),
		C.int(width), C.int(height))
}

// SetDeviceOffset is a wrapper around cairo_surface_set_device_offset().
func (v *Surface) SetDeviceOffset(x, y float64) {
	C.cairo_surface_set_device_offset(v.native(), C.double(x), C.double(y))
}

// GetDeviceOffset is a wrapper around cairo_surface_get_device_offset().
func (v *Surface) DeviceOffset() (x, y float64) {
	var xOffset, yOffset C.double
	C.cairo_surface_get_device_offset(v.native(), &xOffset, &yOffset)
	return float64(xOffset), float64(yOffset)
}

// SetFallbackResolution is a wrapper around
// cairo_surface_set_fallback_resolution().
func (v *Surface) SetFallbackResolution(xPPI, yPPI float64) {
	C.cairo_surface_set_fallback_resolution(v.native(), C.double(xPPI),
		C.double(yPPI))
}

// GetFallbackResolution is a wrapper around cairo_surface_get_fallback_resolution().
func (v *Surface) FallbackResolution() (xPPI, yPPI float64) {
	var x, y C.double
	C.cairo_surface_get_fallback_resolution(v.native(), &x, &y)
	return float64(x), float64(y)
}

// GetType is a wrapper around cairo_surface_get_type().
func (v *Surface) Type() SurfaceType {
	c := C.cairo_surface_get_type(v.native())
	return SurfaceType(c)
}

// TODO(jrick) SetUserData (depends on UserDataKey and DestroyFunc)
// cairo_surface_set_user_data

// TODO(jrick) UserData (depends on UserDataKey)
// cairo_surface_get_user_data

// CopyPage is a wrapper around cairo_surface_copy_page().
func (v *Surface) CopyPage() {
	C.cairo_surface_copy_page(v.native())
}

// ShowPage is a wrapper around cairo_surface_show_page().
func (v *Surface) ShowPage() {
	C.cairo_surface_show_page(v.native())
}

// HasShowTextGlyphs is a wrapper around cairo_surface_has_show_text_glyphs().
func (v *Surface) HasShowTextGlyphs() bool {
	c := C.cairo_surface_has_show_text_glyphs(v.native())
	return gobool(c)
}

// TODO(jrick) SetMimeData (depends on DestroyFunc)
// cairo_surface_set_mime_data

// GetMimeData is a wrapper around cairo_surface_get_mime_data().  The
// returned mimetype data is returned as a Go byte slice.
func (v *Surface) MimeData(mimeType MimeType) []byte {
	cstr := C.CString(string(mimeType))
	defer C.free(unsafe.Pointer(cstr))
	var data *C.uchar
	var length C.ulong
	C.cairo_surface_get_mime_data(v.native(), cstr, &data, &length)
	return C.GoBytes(unsafe.Pointer(data), C.int(length))
}

// WriteToPNG is a wrapper around cairo_surface_write_png(). It writes the Cairo
// surface to the given file in PNG format.
func (v *Surface) WriteToPNG(fileName string) error {
	cstr := C.CString(fileName)
	defer C.free(unsafe.Pointer(cstr))

	status := Status(C.cairo_surface_write_to_png(v.surface, cstr))

	if status != StatusSuccess {
		return status
	}

	return nil
}

type writerBox struct {
	w io.Writer
}

//export _gotk4_cairo_write_func
func _gotk4_cairo_write_func(userdata unsafe.Pointer, data *C.uchar, len C.uint) C.cairo_status_t {
	wb, ok := gbox.Get(uintptr(unsafe.Pointer(userdata))).(writerBox)
	if !ok {
		return C.CAIRO_STATUS_WRITE_ERROR
	}

	bytes := unsafe.Slice((*byte)(unsafe.Pointer(data)), uint(len))

	_, err := wb.w.Write(bytes)
	if err != nil {
		return C.CAIRO_STATUS_WRITE_ERROR
	}

	return C.CAIRO_STATUS_SUCCESS
}

// WriteToPNGWriter is a wrapper around cairo_surface_write_png_stream(). It
// writes the Cairo surface to the given io.Writer in PNG format.
func (v *Surface) WriteToPNGWriter(w io.Writer) error {
	dataptr := gbox.Assign(writerBox{w})
	defer gbox.Delete(dataptr)

	status := Status(C.cairo_surface_write_to_png_stream(
		v.surface,
		(*[0]byte)(C._gotk4_cairo_write_func),
		unsafe.Pointer(dataptr),
	))

	if status != StatusSuccess {
		return status
	}

	return nil
}

// TODO(jrick) SupportsMimeType (since 1.12)
// cairo_surface_supports_mime_type

// TODO(jrick) MapToImage (since 1.12)
// cairo_surface_map_to_image

// TODO(jrick) UnmapImage (since 1.12)
// cairo_surface_unmap_image

// GetHeight is a wrapper around cairo_image_surface_get_height().
func (v *Surface) Height() int {
	return int(C.cairo_image_surface_get_height(v.surface))
}

// GetWidth is a wrapper around cairo_image_surface_get_width().
func (v *Surface) Width() int {
	return int(C.cairo_image_surface_get_width(v.surface))
}

// GetStride is a wrapper around cairo_image_surface_get_stride().
func (v *Surface) Stride() int {
	return int(C.cairo_image_surface_get_stride(v.surface))
}

// GetData is a wrapper around cairo_image_surface_get_data().
func (v *Surface) Data() []byte {
	ptr := unsafe.Pointer(C.cairo_image_surface_get_data(v.surface))
	return unsafe.Slice((*byte)(ptr), v.Height()*v.Stride())
}
