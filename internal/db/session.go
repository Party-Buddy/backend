package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CreateSessionImageRef stores a new use of an image referenced by a session.
func CreateSessionImageRef(tx pgx.Tx, ctx context.Context, sid uuid.UUID, imageId uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO session_image_refs (image_id, session_id)
			VALUES ($1, $2)
		`,
		uuid.NullUUID{UUID: imageId, Valid: true},
		uuid.NullUUID{UUID: sid, Valid: true},
	)

	return err
}

// RemoveSessionImageRefs removes all uses of images originating from a session.
func RemoveSessionImageRefs(tx pgx.Tx, ctx context.Context, sid uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		DELETE FROM session_image_refs
			WHERE session_id = $1
		`,
		uuid.NullUUID{UUID: sid, Valid: true},
	)

	return err
}