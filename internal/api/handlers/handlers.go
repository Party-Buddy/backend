package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"party-buddy/internal/api"
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

func (o OurNotFoundHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	encoder := json.NewEncoder(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	dto := api.Errorf(api.ErrNotFound, "")
	_ = encoder.Encode(dto)
}

type OurMethodNotAllowedHandler struct{}

func (o OurMethodNotAllowedHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	encoder := json.NewEncoder(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	dto := api.Errorf("method-not-allowed", "")
	_ = encoder.Encode(dto)
}
