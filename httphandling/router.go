package httphandling

import (
	"git-codecommit.eu-west-2.amazonaws.com/v1/repos/awskmsluks/config"
	"github.com/gorilla/mux"
	"net/http"
)

const (
	APIVersion = "v1"
)

type Route struct {
	Method         string
	Pattern        string
	Name           string
	Authentication bool
	HandlerFunc    http.HandlerFunc
}

// NewRouter returns a newly configured HTTP mux router.
func NewRouter(c *config.Config) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	var handler http.Handler
	handler = route.HandlerFunc
	handler = WrapCommonHandler(handler, route.Authentication, c)

	addRoutes(router, getDataKeyRoutes(c), c)
	addRoutes(router, getAdmin(c), c)

	return router
}

func addRoutes(router *mux.Router, routes []Route, c *config.Config) *mux.Router {
	for _, route := range routes {
		var handler http.Handler
		handler = route.HandlerFunc
		handler = WrapCommonHandler(handler, route.Authentication, c)

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}
	return router
}
