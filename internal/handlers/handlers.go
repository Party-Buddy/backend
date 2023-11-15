package handlers

import (
	"fmt"
	"net/http"
	"party-buddy/internal/db"
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	_, err := fmt.Fprint(w, "Hello, World!")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func HandlerWithInjectedDBPool(dbpool *db.DBPool, handler func(pool *db.DBPool, w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		handler(dbpool, w, r)
	}
}
