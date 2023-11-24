package db

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func GetGameTasksByID(ctx context.Context, tx pgx.Tx, gameID uuid.UUID) ([]TaskEntity, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, name, owner_id, description, image_id, duration_secs, poll_duration_secs, poll_duration_type, task_kind  
		FROM tasks t 
		INNER JOIN game_tasks gt
		ON gt.game_id = $1
		WHERE t.id = gt.task_id
	`, uuid.NullUUID{UUID: gameID, Valid: true})

	if err != nil {
		return []TaskEntity{}, err
	}

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[TaskEntity])
	if err != nil {
		return []TaskEntity{}, err
	}
	return entities, nil
}

func GetTextAnswerForTaskByID(ctx context.Context, tx pgx.Tx, taskID uuid.UUID) (CheckedTextTaskEntity, error) {
	rows, err := tx.Query(ctx, `
		SELECT * FROM checked_text_tasks WHERE task_id = $1
	`, uuid.NullUUID{UUID: taskID, Valid: true})

	if err != nil {
		return CheckedTextTaskEntity{}, err
	}

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[CheckedTextTaskEntity])
	if err != nil {
		return CheckedTextTaskEntity{}, err
	}
	if len(entities) != 1 {
		return CheckedTextTaskEntity{}, ErrToManyEntitiesWithID
	}
	return entities[0], nil
}

func GetChoicesForTaskByID(ctx context.Context, tx pgx.Tx, taskID uuid.UUID) ([]ChoiceTaskOptionsEntity, error) {
	rows, err := tx.Query(ctx, `
		SELECT * FROM choice_task_options WHERE task_id = $1
	`, uuid.NullUUID{UUID: taskID, Valid: true})

	if err != nil {
		return []ChoiceTaskOptionsEntity{}, err
	}

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[ChoiceTaskOptionsEntity])
	if err != nil {
		return []ChoiceTaskOptionsEntity{}, err
	}
	return entities, nil
}
