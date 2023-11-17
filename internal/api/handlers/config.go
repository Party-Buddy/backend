package handlers

import (
	"github.com/gorilla/mux"
	"net/http"
	"party-buddy/internal/db"
)

// ConfigureMux configures the handlers for HTTP routes and methods
func ConfigureMux(pool *db.DBPool) *mux.Router {
	r := mux.NewRouter()
	r.NotFoundHandler = OurNotFoundHandler{}
	r.MethodNotAllowedHandler = OurMethodNotAllowedHandler{}

	r.HandleFunc("/", IndexHandler).Methods(http.MethodGet)

	// TODO: use auth middleware
	r.HandleFunc("/api/v1/images/{img-id}", GetImageHandler(pool)).Methods(http.MethodGet)
	return r
}
