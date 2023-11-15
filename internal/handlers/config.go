package handlers

import (
	"github.com/gorilla/mux"
	"net/http"
	"party-buddy/internal/db"
)

// ConfigureMux configures the handlers for HTTP routes and methods
func ConfigureMux(pool *db.DBPool) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/", IndexHandler).Methods(http.MethodGet)

	return r
}
