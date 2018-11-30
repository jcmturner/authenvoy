package httphandling

import (
	"encoding/json"
	"net/http"

	"github.com/jcmturner/authenvoy/appcode"
	"github.com/jcmturner/authenvoy/config"
)

// WrapCommonHandler wraps the handler in the authentication handler if required
// and the accessLogger wrapper.
func WrapCommonHandler(inner http.Handler, c *config.Config) http.Handler {
	//Wrap with access logger
	inner = accessLogger(inner, c)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w = setHeaders(w)
		inner.ServeHTTP(w, r)
		return
	})
}

func setHeaders(w http.ResponseWriter) http.ResponseWriter {
	w.Header().Set("Cache-Control", "no-store")
	//OWASP recommended headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "deny")
	return w
}

// JSONGenericResponse is a generic JSON response structure
type JSONGenericResponse struct {
	Message  string
	HTTPCode int
}

func respondGeneric(w http.ResponseWriter, httpCode int, message string) {
	e := JSONGenericResponse{
		Message:  message,
		HTTPCode: httpCode,
	}
	respondWithJSON(w, httpCode, e)
}

func respondWithJSON(w http.ResponseWriter, httpCode int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(httpCode)
	w.Write(response)
}
