// Package gzip implements a gzip compression handler middleware for Negroni.
package gzip

import (
	"compress/gzip"
	"github.com/codegangsta/negroni"
	"net/http"
	"strings"
)

// These compression constants are copied from the compress/gzip package.
const (
	encodingGzip = "gzip"

	headerAcceptEncoding  = "Accept-Encoding"
	headerContentEncoding = "Content-Encoding"
	headerContentLength   = "Content-Length"
	headerContentType     = "Content-Type"
	headerVary            = "Vary"
	headerSecWebSocketKey = "Sec-WebSocket-Key"

	BestCompression    = gzip.BestCompression
	BestSpeed          = gzip.BestSpeed
	DefaultCompression = gzip.DefaultCompression
	NoCompression      = gzip.NoCompression
)

type status int

const (
	COMPRESSION_CHECK status = iota
	COMPRESSION_DISABLED
	COMPRESSION_ENABLED
)

// gzipResponseWriter is the ResponseWriter that negroni.ResponseWriter is
// wrapped in.
type gzipResponseWriter struct {
	r *http.Request
	w *gzip.Writer
	negroni.ResponseWriter
	status           status
	allowCompression AllowCompressionFunc
}

type AllowCompressionFunc func(w http.ResponseWriter, r *http.Request) bool

type Compression interface {
	AllowCompression(w http.ResponseWriter, r *http.Request) bool
}

func (grw *gzipResponseWriter) WriteHeader(code int) {
	if grw.status == COMPRESSION_CHECK {
		if grw.allowCompression == nil || grw.allowCompression(grw, grw.r) {
			grw.status = COMPRESSION_ENABLED
			headers := grw.Header()
			// Delete any existing content length header.
			// see http://stackoverflow.com/questions/3819280/content-length-when-using-http-compression
			headers.Del(headerContentLength)
			// Set the appropriate gzip headers.
			headers.Set(headerContentEncoding, encodingGzip)
			headers.Set(headerVary, headerAcceptEncoding)
		} else {
			grw.status = COMPRESSION_DISABLED
		}
	}
	grw.ResponseWriter.WriteHeader(code)
}

// Write writes bytes to the gzip.Writer. It will also set the Content-Type
// header using the net/http library content type detection if the Content-Type
// header was not set yet.
func (grw *gzipResponseWriter) Write(b []byte) (int, error) {
	if grw.status == COMPRESSION_CHECK {
		if len(grw.Header().Get(headerContentType)) == 0 {
			// Ensure Content-Type detection runs on uncompressed data.
			// Otherwise Content-Type is set it to application/x-gzip.
			grw.Header().Set(headerContentType, http.DetectContentType(b))
		}
		grw.WriteHeader(http.StatusOK)
	}

	if grw.status == COMPRESSION_ENABLED {
		return grw.w.Write(b)
	} else {
		return grw.ResponseWriter.Write(b)
	}
}

// handler struct contains the ServeHTTP method and the compressionLevel to be
// used.
type handler struct {
	compressionLevel int
	allowCompression AllowCompressionFunc
}

func Default() *handler {
	return New(gzip.DefaultCompression, nil)
}

// Gzip returns a handler which will handle the Gzip compression in ServeHTTP.
// Valid values for level are identical to those in the compress/gzip package.
//
// An optional callback can be registered to enable/disable compression.
// The handler runs the the first time data is written to the http.ResponseWriter.
// At this time all response headers have been set.
// So you can easily enable/disable compression based on the 'Content-Type' or
// other response headers if necessary. (e.g 'Content-Range', 'Content-Length' ...)
func New(level int, fn AllowCompressionFunc) *handler {
	return &handler{
		compressionLevel: level,
		allowCompression: fn,
	}
}

// ServeHTTP wraps the http.ResponseWriter with a gzip.Writer.
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	// Skip compression if the client doesn't accept gzip encoding.
	if !strings.Contains(r.Header.Get(headerAcceptEncoding), encodingGzip) {
		next(w, r)
		return
	}

	// Skip compression if client attempt WebSocket connection
	if len(r.Header.Get(headerSecWebSocketKey)) > 0 {
		next(w, r)
		return
	}

	// Skip compression if already compressed
	if w.Header().Get(headerContentEncoding) == encodingGzip {
		next(w, r)
		return
	}

	// Create new gzip Writer. Skip compression if an invalid compression
	// level was set.
	gz, err := gzip.NewWriterLevel(w, h.compressionLevel)
	if err != nil {
		next(w, r)
		return
	}

	// Wrap the original http.ResponseWriter with negroni.ResponseWriter
	// and create the gzipResponseWriter.
	nrw := negroni.NewResponseWriter(w)
	grw := gzipResponseWriter{
		r:                r,
		w:                gz,
		ResponseWriter:   nrw,
		allowCompression: h.allowCompression,
		status:           COMPRESSION_CHECK,
	}

	defer func() {
		if grw.status == COMPRESSION_ENABLED {
			// Calling .Close() does write the GZIP header.
			// This should only happend when compression is enabled.
			gz.Close()
		}
	}()

	// Call the next handler supplying the gzipResponseWriter instead of
	// the original.
	next(&grw, r)
}
