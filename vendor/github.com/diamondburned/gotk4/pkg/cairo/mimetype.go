package cairo

// MimeType is a representation of Cairo's CAIRO_MIME_TYPE_*
// preprocessor constants.
type MimeType string

const (
	MIMETypeJP2      MimeType = "image/jp2"
	MIMETypeJPEG     MimeType = "image/jpeg"
	MIMETypePNG      MimeType = "image/png"
	MIMETypeURI      MimeType = "image/x-uri"
	MIMETypeUniqueID MimeType = "application/x-cairo.uuid"
)
