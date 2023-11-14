package db

import (
	"context"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"image"
	"image/png"
	"os"
	"strings"
)

type ImageStorage struct {
	pool *DBPool
}

func GetImgDirectory() string {
	imgPath := viper.GetString("img.path")
	if imgPath == "" {
		return ""
	}
	if !strings.HasSuffix(imgPath, string(os.PathSeparator)) {
		imgPath += string(os.PathSeparator)
	}
	return imgPath
}

func InitImageStorage(pool *DBPool) ImageStorage {
	return ImageStorage{pool: pool}
}

func (is ImageStorage) NewImgMetadataForOwner(ctx context.Context, owner uuid.UUID) (uuid.NullUUID, error) {
	var retImgUUID uuid.NullUUID

	err := is.pool.Pool().QueryRow(ctx, `
		INSERT INTO images (id, uploaded, read_only, owner_id, created_at) VALUES 
			(DEFAULT, false, false, $1, DEFAULT) RETURNING id
		`, uuid.NullUUID{UUID: owner, Valid: true}).Scan(&retImgUUID)

	if err != nil {
		return retImgUUID, err
	}
	if !retImgUUID.Valid {
		return retImgUUID, ErrGeneratedUUIDInvalid
	}

	return retImgUUID, nil
}

func (is ImageStorage) StoreImage(img image.Image, imgID uuid.UUID) error {
	file, err := os.Open(GetImgDirectory() + imgID.String())
	if err != nil {
		return err
	}

	err = png.Encode(file, img)
	if err != nil {
		return err
	}

	return file.Close()
}
