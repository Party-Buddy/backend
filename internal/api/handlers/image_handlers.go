package handlers

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"image/jpeg"
	"net/http"
	"party-buddy/internal/api"
	"party-buddy/internal/db"
)

func GetImageHandler(dbpool *db.DBPool) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		encoder := json.NewEncoder(w)
		vars := mux.Vars(r)
		val, ok := vars["img-id"]
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			dto := api.Errorf(api.ErrNotFound, "")
			_ = encoder.Encode(dto)
			return
		}
		imgID, err := uuid.Parse(val)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			dto := api.Errorf(api.ErrNotFound, "")
			_ = encoder.Encode(dto)
			return
		}

		var imgMetadata db.ImageEntity
		err = dbpool.Pool().AcquireFunc(context.Background(), func(c *pgxpool.Conn) error {
			imgMetadata, err = db.GetImageMetadataByID(c, context.Background(), uuid.NullUUID{UUID: imgID, Valid: true})
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			dto := api.Errorf(api.ErrNotFound, "")
			_ = encoder.Encode(dto)
			return
		}

		if !imgMetadata.Uploaded {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			dto := api.Errorf(api.ErrNotFound, "")
			_ = encoder.Encode(dto)
			return
		}

		img, err := db.GetImageFromFS(imgMetadata.ID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			dto := api.Errorf(api.ErrNotFound, "")
			_ = encoder.Encode(dto)
			return
		}

		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_ = jpeg.Encode(w, img, nil)
	}
}
