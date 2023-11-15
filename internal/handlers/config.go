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

	// TODO: remove before production
	r.HandleFunc("/test/images", HandlerWithInjectedDBPool(pool, ImgTestHandler)).
		Methods(http.MethodGet)

	// TODO: refactor
	r.HandleFunc("/api/v1/images/{img-id}",
		HandlerWithInjectedDBPool(pool, UploadImgHandler)).
		Methods(http.MethodPut)

	// TODO: refactor
	r.HandleFunc("/api/v1/images/{img-id}",
		HandlerWithInjectedDBPool(pool, GetImgHandler)).
		Methods(http.MethodGet)
	return r
}
