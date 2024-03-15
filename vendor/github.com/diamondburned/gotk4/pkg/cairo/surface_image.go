package cairo

// #include <stdlib.h>
// #include <cairo.h>
// #include <cairo-gobject.h>
// #include <cairo-pdf.h>
import "C"

import (
	"image"
	"image/draw"
	"runtime"
	"unsafe"

	"github.com/diamondburned/gotk4/pkg/cairo/swizzle"
)

// CreatePNGSurface is a wrapper around cairo_image_surface_create_from_png().
func CreatePNGSurfaceFromPNG(fileName string) (*Surface, error) {
	cstr := C.CString(fileName)
	defer C.free(unsafe.Pointer(cstr))

	surfaceNative := C.cairo_image_surface_create_from_png(cstr)

	status := Status(C.cairo_surface_status(surfaceNative))
	if status != StatusSuccess {
		return nil, status
	}

	return &Surface{surface: surfaceNative}, nil
}

// CreateImageSurfaceForData is a wrapper around cairo_image_surface_create_for_data().
func CreateImageSurfaceForData(data []byte, format Format, width, height, stride int) *Surface {
	surfaceNative := C.cairo_image_surface_create_for_data((*C.uchar)(unsafe.Pointer(&data[0])),
		C.cairo_format_t(format), C.int(width), C.int(height), C.int(stride))

	status := Status(C.cairo_surface_status(surfaceNative))
	if status != StatusSuccess {
		panic("cairo_image_surface_create_for_data: " + status.Error())
	}

	s := wrapSurface(surfaceNative)
	runtime.SetFinalizer(s, (*Surface).destroy)

	return s
}

// CreateImageSurface is a wrapper around cairo_image_surface_create().
func CreateImageSurface(format Format, width, height int) *Surface {
	surfaceNative := C.cairo_image_surface_create(C.cairo_format_t(format),
		C.int(width), C.int(height))

	status := Status(C.cairo_surface_status(surfaceNative))
	if status != StatusSuccess {
		panic("cairo_image_surface_create: " + status.Error())
	}

	s := wrapSurface(surfaceNative)
	runtime.SetFinalizer(s, (*Surface).destroy)

	return s
}

// CreateSurfaceFromImage is a better wrapper around cairo_image_surface_create_for_data().
func CreateSurfaceFromImage(img image.Image) *Surface {
	var s *Surface

	switch img := img.(type) {
	case *image.RGBA:
		s = CreateImageSurface(FormatARGB32, img.Rect.Dx(), img.Rect.Dy())
		pix := s.Data()
		// Copy is pretty fast. Copy the RGBA data to the image directly.
		copy(pix, img.Pix)
		// Swizzle the RGBA bytes to the correct order.
		swizzle.BGRA(pix)

	case *image.NRGBA:
		s = CreateImageSurface(FormatARGB32, img.Rect.Dx(), img.Rect.Dy())

		pix := s.Data()
		// I'm not sure how slower this is than just doing a fast copy and
		// calculate onto the malloc'd bytes, but since we're mostly doing
		// calculations for each pixel, it likely doesn't matter.
		for i := 0; i < len(pix); i += 4 {
			alpha8 := img.Pix[i+3]
			alpha16 := uint16(alpha8)
			pix[i+0] = uint8(uint16(img.Pix[i+2]) * alpha16 / 0xFF)
			pix[i+1] = uint8(uint16(img.Pix[i+1]) * alpha16 / 0xFF)
			pix[i+2] = uint8(uint16(img.Pix[i+0]) * alpha16 / 0xFF)
			pix[i+3] = alpha8
		}

	case *image.Alpha:
		s = CreateImageSurface(FormatA8, img.Rect.Dx(), img.Rect.Dy())
		copy(s.Data(), img.Pix)

	default:
		bounds := img.Bounds()
		s = CreateImageSurface(FormatARGB32, bounds.Dx(), bounds.Dy())

		// Create a new image.RGBA that uses the malloc'd byte array as the
		// backing array, then draw directly on it.
		rgba := image.RGBA{
			Pix:    s.Data(),
			Stride: bounds.Dx(),
			Rect:   bounds,
		}
		draw.Draw(&rgba, bounds, img, image.Point{}, draw.Over)
		// The drawn result is in RGBA, so swizzle it to the right format.
		swizzle.BGRA(rgba.Pix)
	}

	s.MarkDirty()
	return s
}
