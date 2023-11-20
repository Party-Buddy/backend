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

	dbm := middleware.DBUsingMiddleware{Pool: pool}
	r.Use(dbm.Middleware)

	// TODO: delete before production
	r.HandleFunc("/", IndexHandler).Methods(http.MethodGet)

	// TODO: use auth middleware
	r.Handle("/api/v1/images/{img-id}", middleware.AuthMiddleware(GetImageHandler{})).Methods(http.MethodGet)

	return r
}
