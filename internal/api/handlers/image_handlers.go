package handlers

import (
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"image/jpeg"
	"log"
	"net/http"
	"party-buddy/internal/api/middleware"
	"party-buddy/internal/db"
	"party-buddy/internal/schemas/api"
)

type GetImageHandler struct{}

// GetImageHandler get an image from fs.
// Before reading file it uses r.Context() to get transaction and context to check if image is uploaded
func (g GetImageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	val, ok := vars["img-id"]
	if !ok {
		msg := "img-id not provided"
		WriteErrorResponse(w, http.StatusNotFound, api.ErrNotFound, msg)
		log.Printf("request: %v %s -> err: %v", r.Method, r.URL, msg)
		return
	}
	imgID, err := uuid.Parse(val)
	if err != nil {
		msg := "invalid url"
		WriteErrorResponse(w, http.StatusNotFound, api.ErrNotFound, msg)
		log.Printf("request: %v %s -> err: %v", r.Method, r.URL, msg)
		return
	}

	tx := middleware.TxFromContext(r.Context())

	imgMetadata, err := db.GetImageMetadataByID(tx, r.Context(), uuid.NullUUID{UUID: imgID, Valid: true})

	if err != nil {
		msg := "not found"
		WriteErrorResponse(w, http.StatusNotFound, api.ErrNotFound, msg)
		log.Printf("request: %v %s -> err: %v", r.Method, r.URL, msg)
		return
	}

	// TODO: middleware.AuthInfoFromContext(r.Context())

	if !imgMetadata.Uploaded {
		msg := "image is not uploaded"
		WriteErrorResponse(w, http.StatusNotFound, api.ErrNotFound, msg)
		log.Printf("request: %v %s -> err: %v", r.Method, r.URL, msg)
		return
	}

	img, err := db.GetImageFromFS(imgMetadata.ID.UUID)
	if err != nil {
		msg := "image not found in storage"
		WriteErrorResponse(w, http.StatusInternalServerError, api.ErrInternal, msg)
		log.Printf("request: %v %s -> err: %v", r.Method, r.URL, msg)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.WriteHeader(http.StatusOK)
	_ = jpeg.Encode(w, img, nil)
	log.Printf("request: %v %s -> OK", r.Method, r.URL)
}
