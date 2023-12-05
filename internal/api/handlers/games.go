package handlers

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"party-buddy/internal/api/base"
	"party-buddy/internal/api/middleware"
	"party-buddy/internal/schemas/api"
)

type GetGameHandler struct{}

func (GetGameHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	val, ok := mux.Vars(r)["game-id"]
	if !ok {
		msg := "game-id not provided"
		base.WriteErrorResponse(w, http.StatusNotFound, api.ErrNotFound, msg)
		log.Printf("request: %v %s -> err: %v", r.Method, r.URL, msg)
		return
	}

	gameID, err := uuid.Parse(val)
	if err != nil {
		msg := "invalid game-id"
		base.WriteErrorResponse(w, http.StatusNotFound, api.ErrNotFound, msg)
		log.Printf("request: %v %s -> err: %v", r.Method, r.URL, msg)
		return
	}

	tx := middleware.TxFromContext(r.Context())
	gameInfo, err := gameIDToIDGameInfo(r.Context(), tx, gameID)
	if err != nil {
		var errConv api.ErrorFromConverters
		errors.As(err, &errConv)
		log.Printf("request: %v %s -> err: %v", r.Method, r.URL, errConv)
		base.WriteErrorResponse(w, errConv.StatusCode, errConv.ApiError.Kind, errConv.ApiError.Message)
		return
	}

	encoder := json.NewEncoder(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = encoder.Encode(gameInfo)
	log.Printf("request: %v %s -> OK", r.Method, r.URL)
}
