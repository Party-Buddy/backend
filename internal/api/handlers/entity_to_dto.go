package handlers

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"net/http"
	"party-buddy/internal/configuration"
	"party-buddy/internal/db"
	"party-buddy/internal/schemas"
	"party-buddy/internal/schemas/api"
)

func gameIDToIDGameInfo(ctx context.Context, tx pgx.Tx, gameID uuid.UUID) (schemas.IDGameInfo, error) {
	gameEntity, err := db.GameByID(ctx, tx, gameID)
	if err != nil {
		return schemas.IDGameInfo{}, api.ErrorFromConverters{
			ApiError:   api.Errorf(api.ErrNotFound, "game not found"),
			StatusCode: http.StatusNotFound,
			LogMessage: fmt.Sprintf("game with id %v not found", gameID),
		}
	}
	gameInfo := schemas.IDGameInfo{
		ID: gameID,
	}
	gameInfo.Name = gameEntity.Name
	gameInfo.Description = gameEntity.Description
	gameInfo.DateChanged = gameEntity.UpdatedAt
	if gameEntity.ImageID.Valid {
		gameInfo.ImgURI = configuration.GenImgURI(gameEntity.ImageID.UUID)
	}

	taskEntities, err := db.GetGameTasksByID(ctx, tx, gameID)
	if err != nil {
		return schemas.IDGameInfo{}, api.ErrorFromConverters{
			ApiError:   api.Errorf(api.ErrInternal, "internal error"),
			StatusCode: http.StatusInternalServerError,
			LogMessage: fmt.Sprintf("failed to get tasks for game with id %v with err: %v", gameID, err),
		}
	}

	tasks := make([]schemas.BaseTaskWithImgAndID, 0, len(taskEntities))
	for _, e := range taskEntities {
		t, err := entityToSchemaTask(e)
		if err != nil {
			return schemas.IDGameInfo{}, err
		}
		tasks = append(tasks, t)
	}
	gameInfo.Tasks = tasks
	return gameInfo, nil
}

func entityToSchemaTask(entity db.TaskEntity) (schemas.BaseTaskWithImgAndID, error) {
	baseTask := schemas.BaseTaskWithImgAndID{}
	baseTask.Name = entity.Name
	baseTask.Description = entity.Description
	baseTask.Duration = schemas.PollDuration{
		Kind: schemas.Fixed,
		Secs: uint16(entity.DurationSeconds),
	}
	baseTask.ID = entity.ID.UUID
	if entity.ImageID.Valid {
		baseTask.ImgURI = configuration.GenImgURI(entity.ImageID.UUID)
	}
	// TODO baseTask.LastUpdated =
	switch entity.TaskKind {
	case db.Text:
		baseTask.Type = schemas.Text
		baseTask.PollDuration = dbToSchemasPollDuration(entity.PollDurationType, entity.PollDurationSeconds)

	case db.Photo:
		baseTask.Type = schemas.Photo
		baseTask.PollDuration = dbToSchemasPollDuration(entity.PollDurationType, entity.PollDurationSeconds)

	case db.CheckedText:
		baseTask.Type = schemas.CheckedText

	case db.Choice:
		baseTask.Type = schemas.Choice

	default:
		return schemas.BaseTaskWithImgAndID{}, api.ErrorFromConverters{
			ApiError:   api.Errorf(api.ErrInternal, ""),
			StatusCode: http.StatusInternalServerError,
			LogMessage: fmt.Sprintf("unknown task kind in database: %s", entity.TaskKind),
		}
	}
	return baseTask, nil
}

func dbToSchemasPollDuration(durationType db.PollDurationType, secs int) schemas.PollDuration {
	switch durationType {
	case db.Fixed:
		return schemas.PollDuration{
			Kind: schemas.Fixed,
			Secs: uint16(secs),
		}
	case db.Dynamic:
		return schemas.PollDuration{
			Kind: schemas.Dynamic,
			Secs: uint16(secs),
		}
	default:
		panic(fmt.Sprintf("unknown poll duration in db %v", durationType))
	}
}
