package cairo

// #include <stdlib.h>
// #include <cairo.h>
// #include <cairo-gobject.h>
import "C"
import (
	"strings"
	"unsafe"
)

// Status is a representation of Cairo's cairo_status_t.
type Status int

const (
	StatusSuccess                Status = C.CAIRO_STATUS_SUCCESS
	StatusNoMemory               Status = C.CAIRO_STATUS_NO_MEMORY
	StatusInvalidRestore         Status = C.CAIRO_STATUS_INVALID_RESTORE
	StatusInvalidPopGroup        Status = C.CAIRO_STATUS_INVALID_POP_GROUP
	StatusNoCurrentPoint         Status = C.CAIRO_STATUS_NO_CURRENT_POINT
	StatusInvalidMatrix          Status = C.CAIRO_STATUS_INVALID_MATRIX
	StatusInvalidStatus          Status = C.CAIRO_STATUS_INVALID_STATUS
	StatusNullPointer            Status = C.CAIRO_STATUS_NULL_POINTER
	StatusInvalidString          Status = C.CAIRO_STATUS_INVALID_STRING
	StatusInvalidPathData        Status = C.CAIRO_STATUS_INVALID_PATH_DATA
	StatusReadError              Status = C.CAIRO_STATUS_READ_ERROR
	StatusWriteError             Status = C.CAIRO_STATUS_WRITE_ERROR
	StatusSurfaceFinished        Status = C.CAIRO_STATUS_SURFACE_FINISHED
	StatusSurfaceTypeMismatch    Status = C.CAIRO_STATUS_SURFACE_TYPE_MISMATCH
	StatusPatternTypeMismatch    Status = C.CAIRO_STATUS_PATTERN_TYPE_MISMATCH
	StatusInvalidContent         Status = C.CAIRO_STATUS_INVALID_CONTENT
	StatusInvalidFormat          Status = C.CAIRO_STATUS_INVALID_FORMAT
	StatusInvalidVisual          Status = C.CAIRO_STATUS_INVALID_VISUAL
	StatusFileNotFound           Status = C.CAIRO_STATUS_FILE_NOT_FOUND
	StatusInvalidDash            Status = C.CAIRO_STATUS_INVALID_DASH
	StatusInvalidDSCComment      Status = C.CAIRO_STATUS_INVALID_DSC_COMMENT
	StatusInvalidIndex           Status = C.CAIRO_STATUS_INVALID_INDEX
	StatusClipNotRepresentable   Status = C.CAIRO_STATUS_CLIP_NOT_REPRESENTABLE
	StatusTempFileError          Status = C.CAIRO_STATUS_TEMP_FILE_ERROR
	StatusInvalidStride          Status = C.CAIRO_STATUS_INVALID_STRIDE
	StatusFontTypeMismatch       Status = C.CAIRO_STATUS_FONT_TYPE_MISMATCH
	StatusUserFontImmutable      Status = C.CAIRO_STATUS_USER_FONT_IMMUTABLE
	StatusUserFontError          Status = C.CAIRO_STATUS_USER_FONT_ERROR
	StatusNegativeCount          Status = C.CAIRO_STATUS_NEGATIVE_COUNT
	StatusInvalidClusters        Status = C.CAIRO_STATUS_INVALID_CLUSTERS
	StatusInvalidSlant           Status = C.CAIRO_STATUS_INVALID_SLANT
	StatusInvalidWeight          Status = C.CAIRO_STATUS_INVALID_WEIGHT
	StatusInvalidSize            Status = C.CAIRO_STATUS_INVALID_SIZE
	StatusUserFontNotImplemented Status = C.CAIRO_STATUS_USER_FONT_NOT_IMPLEMENTED
	StatusDeviceTypeMismatch     Status = C.CAIRO_STATUS_DEVICE_TYPE_MISMATCH
	StatusDeviceError            Status = C.CAIRO_STATUS_DEVICE_ERROR
	// STATUS_INVALID_MESH_CONSTRUCTION Status = C.CAIRO_STATUS_INVALID_MESH_CONSTRUCTION (since 1.12)
	// STATUS_DEVICE_FINISHED           Status = C.CAIRO_STATUS_DEVICE_FINISHED (since 1.12)
)

var keyStatus = map[Status]string{
	StatusSuccess:                "CAIRO_StatusSuccess",
	StatusNoMemory:               "CAIRO_STATUS_NO_MEMORY",
	StatusInvalidRestore:         "CAIRO_STATUS_INVALID_RESTORE",
	StatusInvalidPopGroup:        "CAIRO_STATUS_INVALID_POP_GROUP",
	StatusNoCurrentPoint:         "CAIRO_STATUS_NO_CURRENT_POINT",
	StatusInvalidMatrix:          "CAIRO_STATUS_INVALID_MATRIX",
	StatusInvalidStatus:          "CAIRO_STATUS_INVALID_STATUS",
	StatusNullPointer:            "CAIRO_STATUS_NULL_POINTER",
	StatusInvalidString:          "CAIRO_STATUS_INVALID_STRING",
	StatusInvalidPathData:        "CAIRO_STATUS_INVALID_PATH_DATA",
	StatusReadError:              "CAIRO_STATUS_READ_ERROR",
	StatusWriteError:             "CAIRO_STATUS_WRITE_ERROR",
	StatusSurfaceFinished:        "CAIRO_STATUS_SURFACE_FINISHED",
	StatusSurfaceTypeMismatch:    "CAIRO_STATUS_SURFACE_TYPE_MISMATCH",
	StatusPatternTypeMismatch:    "CAIRO_STATUS_PATTERN_TYPE_MISMATCH",
	StatusInvalidContent:         "CAIRO_STATUS_INVALID_CONTENT",
	StatusInvalidFormat:          "CAIRO_STATUS_INVALID_FORMAT",
	StatusInvalidVisual:          "CAIRO_STATUS_INVALID_VISUAL",
	StatusFileNotFound:           "CAIRO_STATUS_FILE_NOT_FOUND",
	StatusInvalidDash:            "CAIRO_STATUS_INVALID_DASH",
	StatusInvalidDSCComment:      "CAIRO_STATUS_INVALID_DSC_COMMENT",
	StatusInvalidIndex:           "CAIRO_STATUS_INVALID_INDEX",
	StatusClipNotRepresentable:   "CAIRO_STATUS_CLIP_NOT_REPRESENTABLE",
	StatusTempFileError:          "CAIRO_STATUS_TEMP_FILE_ERROR",
	StatusInvalidStride:          "CAIRO_STATUS_INVALID_STRIDE",
	StatusFontTypeMismatch:       "CAIRO_STATUS_FONT_TYPE_MISMATCH",
	StatusUserFontImmutable:      "CAIRO_STATUS_USER_FONT_IMMUTABLE",
	StatusUserFontError:          "CAIRO_STATUS_USER_FONT_ERROR",
	StatusNegativeCount:          "CAIRO_STATUS_NEGATIVE_COUNT",
	StatusInvalidClusters:        "CAIRO_STATUS_INVALID_CLUSTERS",
	StatusInvalidSlant:           "CAIRO_STATUS_INVALID_SLANT",
	StatusInvalidWeight:          "CAIRO_STATUS_INVALID_WEIGHT",
	StatusInvalidSize:            "CAIRO_STATUS_INVALID_SIZE",
	StatusUserFontNotImplemented: "CAIRO_STATUS_USER_FONT_NOT_IMPLEMENTED",
	StatusDeviceTypeMismatch:     "CAIRO_STATUS_DEVICE_TYPE_MISMATCH",
	StatusDeviceError:            "CAIRO_STATUS_DEVICE_ERROR",
}

func marshalStatus(p uintptr) (interface{}, error) {
	c := C.g_value_get_enum((*C.GValue)(unsafe.Pointer(p)))
	return Status(c), nil
}

// String returns a readable status messsage usable in texts.
func (s Status) String() string {
	str, ok := keyStatus[s]
	if !ok {
		str = "CAIRO_STATUS_UNDEFINED"
	}

	str = strings.Replace(str, "CAIRO_STATUS_", "", 1)
	str = strings.Replace(str, "_", " ", 0)
	return strings.ToLower(str)
}

// Error implements error. It calls String() unless s is StatusSuccess.
func (s Status) Error() string {
	if s == StatusSuccess {
		return "<nil>"
	}
	return s.String()
}

// ToError returns the error for the status. Returns nil if success.
func (s Status) ToError() error {
	if s == StatusSuccess {
		return nil
	}
	return s
}
