package db

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/spf13/viper"
	"image"
	"image/png"
	"log"
	"os"
	"strings"
)

type ImageStorage struct {
	pool *DBPool
}

var imgPath string

// GetImgDirectory returns the image directory path, which ends with os.PathSeparator
func GetImgDirectory() string {
	if imgPath != "" {
		return imgPath
	}

	imgPath = viper.GetString("img.path")
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

// NewImgMetadataForOwner creates new image metadata record in db
// and returns the id (Type: uuid) of the record
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

// Store is used for storing new image by given image ID.
// It starts transaction, during the transaction method does the following:
//  1. checks read_only (fails if true)
//  2. opens img file (creates new if there is no file)
//  3. writes img to the file in png format // TODO: discuss
//  4. closes file
//  5. sets the uploaded to true
//
// If something goes wrong the transaction is rolled back and the file is deleted (if it had been opened)
//
// TODO: add owner_id checking?
// TODO: what should we do with updating image?
func (is ImageStorage) Store(ctx context.Context, img image.Image, imgID uuid.UUID) error {
	transaction, err := is.pool.Pool().Begin(ctx)
	if err != nil {
		return err
	}

	dbImgId := uuid.NullUUID{UUID: imgID, Valid: true}
	var isReadOnly, isUploaded pgtype.Bool
	err = transaction.QueryRow(ctx, `
		SELECT read_only, uploaded FROM images WHERE id = $1
		`, dbImgId).Scan(&isReadOnly, &isUploaded)

	if err != nil {
		_ = transaction.Rollback(ctx)
		log.Printf("Failed to get readonly, uploaded image metadata")
		return err
	}

	if !isReadOnly.Valid {
		_ = transaction.Rollback(ctx)
		return ErrInvalidDBValue
	}

	if isReadOnly.Bool {
		_ = transaction.Rollback(ctx)
		return ErrImageIsReadOnly
	}

	file, err := os.OpenFile(GetImgDirectory()+imgID.String(), os.O_WRONLY|os.O_CREATE, 0200)
	if err != nil {
		_ = transaction.Rollback(ctx)
		return err
	}

	err = png.Encode(file, img)
	if err != nil {
		_ = transaction.Rollback(ctx)
		return err
	}

	_ = file.Close()

	err = transaction.QueryRow(ctx, `
		UPDATE images SET uploaded = TRUE WHERE id = $1
		`, dbImgId).Scan()

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		_ = transaction.Rollback(ctx)
		_ = os.Remove(GetImgDirectory() + imgID.String())
		return err
	}

	err = transaction.Commit(ctx)
	if err != nil && !errors.Is(err, pgx.ErrTxCommitRollback) {
		_ = os.Remove(GetImgDirectory() + imgID.String())
		return err
	}

	return nil
}

// GetById returns the image in png (TODO: discuss)
// by given uuid
// TODO: add owner_id checking?
func (is ImageStorage) GetById(ctx context.Context, imgID uuid.UUID) (image.Image, error) {

	dbImgId := uuid.NullUUID{UUID: imgID, Valid: true}
	var isUploaded pgtype.Bool
	err := is.pool.Pool().QueryRow(ctx, `
		SELECT uploaded FROM images WHERE id = $1
		`, dbImgId).Scan(&isUploaded)

	if err != nil {
		log.Printf("Failed to get uploaded image metadata")
		return nil, err
	}

	if !isUploaded.Valid {
		return nil, ErrInvalidDBValue
	}

	if !isUploaded.Bool {
		return nil, ErrImageIsNotUploaded
	}

	file, err := os.OpenFile(GetImgDirectory()+imgID.String(), os.O_RDONLY, 0400)
	if err != nil {
		return nil, err
	}

	img, err := png.Decode(file)
	if err != nil {
		return nil, err
	}

	_ = file.Close()

	return img, nil
}

// TODO: GetMetadata
// TODO: GetDataFromRefsCountView
// TODO: GetImage
