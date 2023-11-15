package handlers

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"net/http"
	"party-buddy/internal/db"
)

// ImgTestHandler is used for generating metadata
// this is DEBUG handler
//
// TODO: remove before production
func ImgTestHandler(dbpool *db.DBPool, w http.ResponseWriter, _ *http.Request) {
	imgStorage := db.InitImageStorage(dbpool)

	newImgUUID, err := imgStorage.NewImgMetadataForOwner(context.Background(), uuid.New())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "error generating image metadata: %v", err.Error())
		return
	}

	entities, err := imgStorage.GetMetadataByIDs(context.Background(), []uuid.NullUUID{newImgUUID})
	log.Printf("Got %v entities", len(entities))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "Generated uuid: %v\nerror getting image metadata: %v", newImgUUID.UUID.String(), err.Error())
		return
	}

	if len(entities) <= 0 {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "Generated uuid: %v\nGot zero entities", newImgUUID.UUID.String())
		return
	}

	w.WriteHeader(http.StatusOK)
	first := entities[0]
	_, _ = fmt.Fprintf(w, "generated img info: {\n\tid: %v\n\towner_id: %v\n\tread_only: %v\n\tuploaded: %v\n\tcreated_at: %v\n}",
		first.ID.UUID.String(),
		first.OwnerID.UUID.String(),
		first.ReadOnly,
		first.Uploaded,
		first.CreatedAt.String())
}

// UploadImgHandler is used for /api/v1/images/{image-id}
// read image from body and stores it by given id in path
// this is NOT final variant
//
// TODO:
//  1. validation
//  2. json return values
//  3. proper errors
//  4. and many other I suppose
func UploadImgHandler(dbpool *db.DBPool, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	strID, ok := vars["img-id"]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, "Image id not provided")
		return
	}
	imgID, err := uuid.Parse(strID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintf(w, "Bad image id: %v", err.Error())
		return
	}

	var img image.Image
	contentType := r.Header.Get("Content-Type")
	switch contentType {
	case "image/jpeg":
		{
			img, err = jpeg.Decode(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = fmt.Fprintf(w, "Bad image/jpeg format: %v", err.Error())
				return
			}
		}
	case "image/png":
		{
			img, err = png.Decode(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = fmt.Fprintf(w, "Bad image/png format: %v", err.Error())
				return
			}
		}
	default:
		{
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprint(w, "Bad content type, expected image/png or image/jpeg")
			return
		}
	}

	imgStorage := db.InitImageStorage(dbpool)
	err = imgStorage.Store(context.Background(), img, imgID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintf(w, "Failed to upload image: %v", err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, "Success")
}

// GetImgHandler is used for /api/v1/images/{image-id}
// return image by given id in path
// this is NOT final variant
//
// TODO:
//  1. validation
//  2. json return values
//  3. proper errors
//  4. and many other I suppose
func GetImgHandler(dbpool *db.DBPool, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	strID, ok := vars["img-id"]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, "Image id not provided")
		return
	}
	imgID, err := uuid.Parse(strID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintf(w, "Bad image id: %v", err.Error())
		return
	}

	imgStorage := db.InitImageStorage(dbpool)

	img, err := imgStorage.GetById(context.Background(), imgID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintf(w, "Failed to get image: %v", err.Error())
		return
	}

	err = png.Encode(w, img)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "image/png")
}
