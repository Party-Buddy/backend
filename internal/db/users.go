package db

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func GetUserByID(ctx context.Context, tx pgx.Tx, userID uuid.NullUUID) (UserEntity, error) {
	rows, err := tx.Query(ctx, `
		SELECT * FROM users WHERE id = $1
		`, userID)
	if err != nil {
		return UserEntity{}, err
	}

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[UserEntity])
	if err != nil {
		return UserEntity{}, err
	}
	if len(entities) == 0 {
		return UserEntity{ID: userID, Role: Base}, nil
	}
	if len(entities) > 1 {
		return UserEntity{}, ErrToManyEntitiesWithID
	}
	return entities[0], nil
}
