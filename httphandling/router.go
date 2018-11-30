package httphandling

import (
	"github.com/gorilla/mux"
	"github.com/jcmturner/authenvoy/config"
)

const (
	// APIVersion is the version prefix on the ReST URL
	APIVersion = "v1"
)

// NewRouter returns a newly configured HTTP mux router.
func NewRouter(c *config.Config) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	handler := WrapCommonHandler(authenticate(c), c)
	router.
		Methods("POST").
		Path("/" + APIVersion + "/authenticate").
		Name("authenticate").
		Handler(handler)
	return router
}
