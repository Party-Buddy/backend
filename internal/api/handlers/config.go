package handlers

import (
	"github.com/gorilla/mux"
	"net/http"
	"party-buddy/internal/api/middleware"
	"party-buddy/internal/db"
)

// ConfigureMux configures the handlers for HTTP routes and methods
func ConfigureMux(pool *db.DBPool) *mux.Router {
	r := mux.NewRouter()
	r.NotFoundHandler = OurNotFoundHandler{}
	r.MethodNotAllowedHandler = OurMethodNotAllowedHandler{}

	// TODO: delete before production
	r.HandleFunc("/", IndexHandler).Methods(http.MethodGet)

	dbm := middleware.DBUsingMiddleware{Pool: pool}

	// TODO: use auth middleware
	r.HandleFunc("/api/v1/images/{img-id}", GetImageHandler).Methods(http.MethodGet)

	r.Use(dbm.Middleware)
	return r
}
