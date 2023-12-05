package base

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
	msg := "page was not found"
	WriteErrorResponse(w, http.StatusNotFound, api.ErrNotFound, msg)
	log.Printf("request: %v %s -> err: %v", r.Method, r.URL, msg)
}

type OurMethodNotAllowedHandler struct{}

func (o OurMethodNotAllowedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	msg := fmt.Sprintf("method not allowed for endpoint: %v", r.URL.Path)
	WriteErrorResponse(w, http.StatusMethodNotAllowed, api.ErrMethodNotAllowed, msg)
	log.Printf("request: %v %s -> err: %v", r.Method, r.URL, msg)
}

// WriteErrorResponse writes error response to w with given status code.
// Body has the JSON format, constructed with api.Errorf, so kind and msg needed.
// After calling this function no data should be written to w.
func WriteErrorResponse(w http.ResponseWriter, code int, kind api.ErrorKind, message string) {
	encoder := json.NewEncoder(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	dto := api.Error{Kind: kind, Message: message}
	_ = encoder.Encode(dto)
}
