package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"party-buddy/internal/schemas/api"
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

type OurNotFoundHandler struct{}

func (o OurNotFoundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	encoder := json.NewEncoder(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	dto := api.Errorf(api.ErrNotFound, "page was not found")
	_ = encoder.Encode(dto)
	log.Printf("request: %v %s -> err: %v", r.Method, r.URL, dto)
}

type OurMethodNotAllowedHandler struct{}

func (o OurMethodNotAllowedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	encoder := json.NewEncoder(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	dto := api.Errorf(api.ErrMethodNotAllowed, "method not allowed for endpoint: %v", r.URL.Path)
	_ = encoder.Encode(dto)
	log.Printf("request: %v %s -> err: %v", r.Method, r.URL, dto)
}
