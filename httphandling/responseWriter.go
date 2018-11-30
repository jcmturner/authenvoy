package httphandling

import "net/http"

// ResponseWriterWrapper is a wapper for the response writer
type ResponseWriterWrapper struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

// NewResponseWriterWrapper returns a ResponseWriterWrapper
func NewResponseWriterWrapper(w http.ResponseWriter) *ResponseWriterWrapper {
	return &ResponseWriterWrapper{ResponseWriter: w}
}

// Status returns the status code
func (w *ResponseWriterWrapper) Status() int {
	return w.status
}

// Write to the ResponseWriterWrapper
func (w *ResponseWriterWrapper) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}

// WriteHeader writes the status code header
func (w *ResponseWriterWrapper) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
	// Check after in case there's error handling in the wrapped ResponseWriter.
	if w.wroteHeader {
		return
	}
	w.status = code
	w.wroteHeader = true
}
