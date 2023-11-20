package handlers

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"image/jpeg"
	"net/http"
	"party-buddy/internal/api"
	"party-buddy/internal/db"
)

// GetImageHandler get an image from fs.
// Before reading file it uses r.Context() to get transaction and context to check if image is uploaded
func GetImageHandler(w http.ResponseWriter, r *http.Request) {
	encoder := json.NewEncoder(w)
	vars := mux.Vars(r)
	val, ok := vars["img-id"]
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		dto := api.Errorf(api.ErrNotFound, "img-id not provided")
		_ = encoder.Encode(dto)
		return
	}
	imgID, err := uuid.Parse(val)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		dto := api.Errorf(api.ErrNotFound, "invalid url")
		_ = encoder.Encode(dto)
		return
	}

	tx := r.Context().Value("tx").(pgx.Tx)
	ctx := r.Context().Value("ctx").(context.Context)

	imgMetadata, err := db.GetImageMetadataByID(tx, ctx, uuid.NullUUID{UUID: imgID, Valid: true})

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		dto := api.Errorf(api.ErrNotFound, "no record in db")
		_ = encoder.Encode(dto)
		return
	}

	if !imgMetadata.Uploaded {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		dto := api.Errorf(api.ErrNotFound, "image is not uploaded")
		_ = encoder.Encode(dto)
		return
	}

	img, err := db.GetImageFromFS(imgMetadata.ID.UUID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		dto := api.Errorf(api.ErrInternal, "image not found in storage")
		_ = encoder.Encode(dto)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.WriteHeader(http.StatusOK)
	_ = jpeg.Encode(w, img, nil)

}
