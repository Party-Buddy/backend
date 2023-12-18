package db

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CreateSessionImageRef stores a new use of an image referenced by a session.
func CreateSessionImageRef(ctx context.Context, tx pgx.Tx, sid uuid.UUID, imageID uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO session_image_refs (image_id, session_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`,
		uuid.NullUUID{UUID: imageID, Valid: true},
		uuid.NullUUID{UUID: sid, Valid: true},
	)

	return err
}

// RemoveSessionImageRefs removes all uses of images originating from a session.
func RemoveSessionImageRefs(ctx context.Context, tx pgx.Tx, sid uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		DELETE FROM session_image_refs
			WHERE session_id = $1
		`,
		uuid.NullUUID{UUID: sid, Valid: true},
	)

	return err
}
