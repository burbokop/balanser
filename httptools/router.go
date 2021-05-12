package httptools

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

type Router struct {
	routes []Route
}

func NewRouter(routes []Route) *Router {
	return &Router{routes: routes}
}

func (r *Router) Start(port int) {
	router := mux.NewRouter().StrictSlash(true)
	for _, route := range r.routes {
		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(route.HandlerFunc)
	}
	err := http.ListenAndServe(":"+fmt.Sprint(port), router)
	if err != nil {
		log.Fatal(err)
	}
}
