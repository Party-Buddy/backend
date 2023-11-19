package handlers

import (
	"context"
	"github.com/gorilla/mux"
	"net/http"
	"party-buddy/internal/api/middleware"
	"party-buddy/internal/db"
)

// ConfigureMux configures the handlers for HTTP routes and methods
func ConfigureMux(ctx context.Context, pool *db.DBPool) *mux.Router {
	r := mux.NewRouter()
	r.NotFoundHandler = OurNotFoundHandler{}
	r.MethodNotAllowedHandler = OurMethodNotAllowedHandler{}

	// TODO: delete before production
	r.HandleFunc("/", IndexHandler).Methods(http.MethodGet)

	dbm := middleware.DBUsingMiddleware{Pool: pool, Ctx: ctx}

	// TODO: use auth middleware
	r.Handle("/api/v1/images/{img-id}", dbm.Middleware(GetImageHandler{})).Methods(http.MethodGet)
	return r
}
