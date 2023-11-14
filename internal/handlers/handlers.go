package handlers

import (
	"context"
	"fmt"
	"github.com/google/uuid"
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

func ImgTestHandler(w http.ResponseWriter, _ *http.Request) {
	conf, err := db.GetDBConfig()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "db is not configured: %v", err.Error())
		return
	}
	dbpool, err := db.InitDBPool(context.Background(), conf)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "failed to init connection pool: %v", err.Error())
		return
	}

	imgStorage := db.InitImageStorage(&dbpool)

	newImgUUID, err := imgStorage.NewImgMetadataForOwner(context.Background(), uuid.New())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "error generating image metadata: %v", err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = fmt.Fprintf(w, "generated img uuid: %v", newImgUUID.UUID.String())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
