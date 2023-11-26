package handlers

import (
	"github.com/gorilla/mux"
	"net/http"
	"party-buddy/internal/api/middleware"
	"party-buddy/internal/db"
	"party-buddy/internal/session"
	"party-buddy/internal/validate"
)

// ConfigureMux configures the handlers for HTTP routes and methods
func ConfigureMux(pool *db.DBPool, manager *session.Manager) *mux.Router {
	r := mux.NewRouter()
	r.NotFoundHandler = OurNotFoundHandler{}
	r.MethodNotAllowedHandler = OurMethodNotAllowedHandler{}

	dbm := middleware.DBUsingMiddleware{Pool: pool}
	managerMid := middleware.ManagerUsingMiddleware{Manager: manager}
	validateMid := middleware.ValidateMiddleware{Factory: validate.NewValidationFactory()}

	r.Use(dbm.Middleware)
	r.Use(validateMid.Middleware)

	// TODO: delete before production
	r.HandleFunc("/", IndexHandler).Methods(http.MethodGet)

	r.Handle("/api/v1/images/{img-id}", middleware.AuthMiddleware(
		GetImageHandler{})).Methods(http.MethodGet)

	r.Handle("/api/v1/session", middleware.AuthMiddleware(
		managerMid.Middleware(SessionConnectHandler{}))).Methods(http.MethodGet)

	r.Handle("/api/v1/session", middleware.AuthMiddleware(
		managerMid.Middleware(SessionCreateHandler{}))).Methods(http.MethodPost)

	return r
}
