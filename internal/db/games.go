package db

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func GameByID(ctx context.Context, tx pgx.Tx, gameID uuid.UUID) (GameEntity, error) {
	rows, err := tx.Query(ctx, `
		SELECT * FROM games WHERE id = $1
	`, uuid.NullUUID{UUID: gameID, Valid: true})

	if err != nil {
		return GameEntity{}, err
	}

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[GameEntity])
	if err != nil {
		return GameEntity{}, err
	}
	if len(entities) != 1 {
		return GameEntity{}, ErrToManyEntitiesWithID
	}
	return entities[0], nil
}
