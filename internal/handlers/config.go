package handlers

import "github.com/gorilla/mux"

// ConfigureMux configures the handlers for HTTP routes and methods
func ConfigureMux() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/", IndexHandler).Methods("GET")
	r.HandleFunc("/test/images", ImgTestHandler).Methods("GET")
	return r
}
