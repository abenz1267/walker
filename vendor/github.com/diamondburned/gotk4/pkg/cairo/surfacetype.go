package cairo

// #include <stdlib.h>
// #include <cairo.h>
// #include <cairo-gobject.h>
import "C"
import (
	"unsafe"
)

// SurfaceType is a representation of Cairo's cairo_surface_type_t.
type SurfaceType int

const (
	SurfaceTypeImage         SurfaceType = C.CAIRO_SURFACE_TYPE_IMAGE
	SurfaceTypePDF           SurfaceType = C.CAIRO_SURFACE_TYPE_PDF
	SurfaceTypePS            SurfaceType = C.CAIRO_SURFACE_TYPE_PS
	SurfaceTypeXlib          SurfaceType = C.CAIRO_SURFACE_TYPE_XLIB
	SurfaceTypeXCB           SurfaceType = C.CAIRO_SURFACE_TYPE_XCB
	SurfaceTypeGlitz         SurfaceType = C.CAIRO_SURFACE_TYPE_GLITZ
	SurfaceTypeQuartz        SurfaceType = C.CAIRO_SURFACE_TYPE_QUARTZ
	SurfaceTypeWin32         SurfaceType = C.CAIRO_SURFACE_TYPE_WIN32
	SurfaceTypeBeOS          SurfaceType = C.CAIRO_SURFACE_TYPE_BEOS
	SurfaceTypeDirectFB      SurfaceType = C.CAIRO_SURFACE_TYPE_DIRECTFB
	SurfaceTypeSVG           SurfaceType = C.CAIRO_SURFACE_TYPE_SVG
	SurfaceTypeOS2           SurfaceType = C.CAIRO_SURFACE_TYPE_OS2
	SurfaceTypeWin32Printing SurfaceType = C.CAIRO_SURFACE_TYPE_WIN32_PRINTING
	SurfaceTypeQuartzImage   SurfaceType = C.CAIRO_SURFACE_TYPE_QUARTZ_IMAGE
	SurfaceTypeScript        SurfaceType = C.CAIRO_SURFACE_TYPE_SCRIPT
	SurfaceTypeQt            SurfaceType = C.CAIRO_SURFACE_TYPE_QT
	SurfaceTypeRecording     SurfaceType = C.CAIRO_SURFACE_TYPE_RECORDING
	SurfaceTypeVG            SurfaceType = C.CAIRO_SURFACE_TYPE_VG
	SurfaceTypeGL            SurfaceType = C.CAIRO_SURFACE_TYPE_GL
	SurfaceTypeDRM           SurfaceType = C.CAIRO_SURFACE_TYPE_DRM
	SurfaceTypeTee           SurfaceType = C.CAIRO_SURFACE_TYPE_TEE
	SurfaceTypeXML           SurfaceType = C.CAIRO_SURFACE_TYPE_XML
	SurfaceTypeSKia          SurfaceType = C.CAIRO_SURFACE_TYPE_SKIA
	SurfaceTypeSubsurface    SurfaceType = C.CAIRO_SURFACE_TYPE_SUBSURFACE
	// SURFACE_TYPE_COGL           SurfaceType = C.CAIRO_SURFACE_TYPE_COGL (since 1.12)
)

func marshalSurfaceType(p uintptr) (interface{}, error) {
	c := C.g_value_get_enum((*C.GValue)(unsafe.Pointer(p)))
	return SurfaceType(c), nil
}
