package handlers

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"party-buddy/internal/db"
	"party-buddy/internal/schemas/api"
	"party-buddy/internal/session"
	"time"
)

func gameIDToSessionGame(ctx context.Context, tx pgx.Tx, gameID uuid.UUID) (session.Game, error) {
	game := session.Game{}
	gameEntity, err := db.GameByID(ctx, tx, gameID)
	if err != nil {
		return session.Game{}, api.Errorf(api.ErrNotFound, err.Error())
	}
	game.Name = gameEntity.Name
	game.Description = gameEntity.Description
	game.DateChanged = gameEntity.UpdatedAt
	game.ImageId = session.ImageId(gameEntity.ImageID)

	taskEntities, err := db.GetGameTasksByID(ctx, tx, gameID)
	if err != nil {
		return session.Game{}, api.Errorf(api.ErrInternal, err.Error())
	}
	tasks := make([]session.Task, len(taskEntities))
	for i := 0; i < len(taskEntities); i++ {
		t, err := entityToSessionTask(ctx, tx, taskEntities[i])
		if err != nil {
			return session.Game{}, err
		}
		tasks = append(tasks, t)
	}
	game.Tasks = tasks
	return game, nil
}

func entityToSessionTask(ctx context.Context, tx pgx.Tx, entity db.TaskEntity) (session.Task, error) {
	baseTask := session.BaseTask{
		Name:         entity.Name,
		Description:  entity.Description,
		TaskDuration: time.Duration(entity.DurationSeconds) * time.Second,
		ImageId:      session.ImageId(entity.ImageID),
	}
	switch entity.TaskKind {
	case db.Text:
		return session.TextTask{
			BaseTask: baseTask,
			PollDuration: dbToSessionPollDuration(
				entity.PollDurationType,
				entity.PollDurationSeconds),
		}, nil

	case db.Photo:
		return session.PhotoTask{
			BaseTask: baseTask,
			PollDuration: dbToSessionPollDuration(
				entity.PollDurationType,
				entity.PollDurationSeconds),
		}, nil

	case db.CheckedText:
		answerEntity, err := db.GetTextAnswerForTaskByID(ctx, tx, entity.ID.UUID)
		if err != nil {
			return nil, api.Errorf(api.ErrInternal, err.Error())
		}
		return session.CheckedTextTask{
			BaseTask: baseTask,
			Answer:   answerEntity.Answer,
		}, nil

	case db.Choice:
		choiceEntities, err := db.GetChoicesForTaskByID(ctx, tx, entity.ID.UUID)
		if err != nil {
			return nil, api.Errorf(api.ErrInternal, err.Error())
		}
		var answerIdx int
		options := make([]string, len(choiceEntities))
		for i := 0; i < len(choiceEntities); i++ {
			if choiceEntities[i].Correct {
				answerIdx = i
			}
			options[i] = choiceEntities[i].Alternative
		}
		return session.ChoiceTask{
			BaseTask:  baseTask,
			Options:   options,
			AnswerIdx: answerIdx,
		}, nil

	default:
		return nil, api.Errorf(api.ErrInternal, "unknown task kind in database")
	}
}

func dbToSessionPollDuration(kind db.PollDurationType, secs int) session.PollDurationer {
	switch kind {
	case db.Fixed:
		return session.FixedPollDuration(time.Duration(secs) * time.Second)

	case db.Dynamic:
		return session.DynamicPollDuration(time.Duration(secs) * time.Second)

	default:
		panic("unknown poll duration type in database")
	}
}
