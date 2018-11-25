package httphandling

import "net/http"

type ResponseWriterWrapper struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func NewResponseWriterWrapper(w http.ResponseWriter) *ResponseWriterWrapper {
	return &ResponseWriterWrapper{ResponseWriter: w}
}

func (w *ResponseWriterWrapper) Status() int {
	return w.status
}

func (w *ResponseWriterWrapper) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}

func (w *ResponseWriterWrapper) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
	// Check after in case there's error handling in the wrapped ResponseWriter.
	if w.wroteHeader {
		return
	}
	w.status = code
	w.wroteHeader = true
}
