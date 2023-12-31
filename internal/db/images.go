package db

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"image"
	"image/jpeg"
	"os"
	"party-buddy/internal/configuration"
)

// CreateImageMetadata creates new image metadata record in db
// and returns the id (Type: uuid) of the record
func CreateImageMetadata(tx pgx.Tx, ctx context.Context, owner uuid.UUID) (uuid.NullUUID, error) {
	var retImgUUID uuid.NullUUID

	err := tx.QueryRow(ctx, `
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

// GetImageMetadataByIDs returns array of image metadata by each id in imgIDs
// If you need a many image metadata in the same time use this function instead of cycle with GetImageMetadataByID
func GetImageMetadataByIDs(tx pgx.Tx, ctx context.Context, imgIDs []uuid.NullUUID) ([]ImageEntity, error) {
	rows, err := tx.Query(ctx, `
		SELECT * FROM images WHERE id = ANY ($1)
		`, imgIDs)
	if err != nil {
		return []ImageEntity{}, err
	}

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[ImageEntity])
	if err != nil {
		return []ImageEntity{}, err
	}
	return entities, nil
}

// GetImageMetadataByID returns image metadata by given id
func GetImageMetadataByID(tx pgx.Tx, ctx context.Context, imgID uuid.NullUUID) (ImageEntity, error) {
	rows, err := tx.Query(ctx, `
		SELECT * FROM images WHERE id = $1
		`, imgID)
	if err != nil {
		return ImageEntity{}, err
	}

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[ImageEntity])
	if err != nil {
		return ImageEntity{}, err
	}
	if len(entities) != 1 {
		return ImageEntity{}, ErrToManyEntitiesWithID
	}
	return entities[0], nil
}

// SetImageUploaded sets for image with id imgID field uploaded to value
func SetImageUploaded(tx pgx.Tx, ctx context.Context, imgID uuid.NullUUID, value bool) error {
	return tx.QueryRow(ctx, `
		UPDATE images SET uploaded = $2 WHERE id = $1
		`, imgID, pgtype.Bool{Bool: value, Valid: true}).Scan()
}

// SetImageReadOnly sets for image with id imgID field read_only to value
func SetImageReadOnly(tx pgx.Tx, ctx context.Context, imgID uuid.NullUUID, value bool) error {
	return tx.QueryRow(ctx, `
		UPDATE images SET read_only = $2 WHERE id = $1
		`, imgID, pgtype.Bool{Bool: value, Valid: true}).Scan()
}

func GetImageFromFS(imgID uuid.UUID) (image.Image, error) {
	imgDir := configuration.GetImgDirectory()
	imgPath := imgDir + imgID.String()

	file, err := os.Open(imgPath)
	if err != nil {
		return nil, err
	}

	img, err := jpeg.Decode(file)
	if err != nil {
		return nil, err
	}
	return img, nil
}
