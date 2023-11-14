package db

import (
	"context"
	"github.com/google/uuid"
)

func NewImgMetadataForOwner(ctx context.Context, pool DBPool, owner uuid.UUID) (uuid.NullUUID, error) {
	var retImgUUID uuid.NullUUID

	err := pool.Pool().QueryRow(ctx, `
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
