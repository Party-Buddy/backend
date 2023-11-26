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
	"party-buddy/internal/session"
	"time"
)

func toSessionGame(
	ctx context.Context,
	tx pgx.Tx,
	owner uuid.UUID,
	gameInfo schemas.FullGameInfo,
) (session.Game, []api.ImgReqResponse, error) {

	game := session.Game{}
	game.Name = *gameInfo.Name
	game.Description = *gameInfo.Description
	imgs := make(map[api.ImgRequest]uuid.UUID)
	if *gameInfo.ImgRequest >= 0 {
		imgID, err := db.CreateImageMetadata(tx, ctx, owner)
		if err != nil {
			return session.Game{}, nil, api.ErrorFromConverters{
				ApiError:   api.Errorf(api.ErrInternal, ""),
				StatusCode: http.StatusInternalServerError,
				LogMessage: fmt.Sprintf("failed to create img metadata: %s", err),
			}
		}
		game.ImageID = session.ImageID(imgID)
		imgs[*gameInfo.ImgRequest] = imgID.UUID
	}
	tasks := make([]session.Task, len(*gameInfo.Tasks))
	for i := 0; i < len(*gameInfo.Tasks); i++ {
		t, newImgs, err := toSessionTask(ctx, tx, owner, (*gameInfo.Tasks)[i], imgs)
		if err != nil {
			return session.Game{}, nil, err
		}
		imgs = newImgs
		tasks = append(tasks, t)
	}
	imgResps := make([]api.ImgReqResponse, 0)
	for k, v := range imgs {
		imgResps = append(imgResps, api.ImgReqResponse{ImgRequest: k, ImgURI: configuration.GenImgURI(v)})
	}
	return game, imgResps, nil
}

func toSessionTask(
	ctx context.Context,
	tx pgx.Tx,
	owner uuid.UUID,
	task schemas.BaseTaskWithImgRequest,
	imgs map[api.ImgRequest]uuid.UUID,
) (session.Task, map[api.ImgRequest]uuid.UUID, error) {
	sessionImgID, newImgs, err := genSessionImgID(ctx, tx, owner, *task.ImgRequest, imgs)
	if err != nil {
		return nil, imgs, err
	}
	baseTask := session.BaseTask{
		Name:         *task.Name,
		Description:  *task.Description,
		ImageID:      session.ImageID(sessionImgID),
		TaskDuration: time.Duration(task.Duration.Secs) * time.Second,
	}

	if task.Type == nil {
		panic("unexpected nil for task type while converting received task to session task")
	}

	switch *task.Type {
	case schemas.Photo:
		return session.PhotoTask{
			BaseTask:     baseTask,
			PollDuration: toSessionPollDuration(*task.PollDuration),
		}, newImgs, nil

	case schemas.Text:
		return session.TextTask{
			BaseTask:     baseTask,
			PollDuration: toSessionPollDuration(*task.PollDuration),
		}, newImgs, nil

	case schemas.CheckedText:
		return session.CheckedTextTask{
			BaseTask: baseTask,
			Answer:   *task.Answer,
		}, newImgs, nil

	case schemas.Choice:
		return session.ChoiceTask{
			BaseTask:  baseTask,
			Options:   *task.Options,
			AnswerIdx: int(*task.AnswerIndex),
		}, newImgs, nil

	default:
		return nil, imgs, api.ErrorFromConverters{
			ApiError:   api.Errorf(api.ErrTaskInvalid, "unknown task type: %s", *task.Type),
			StatusCode: http.StatusBadRequest,
			LogMessage: fmt.Sprintf("unknown task type: %s", *task.Type),
		}
	}
}

func toSessionPollDuration(duration schemas.PollDuration) session.PollDurationer {
	dur := time.Second * time.Duration(duration.Secs)
	switch duration.Kind {
	case schemas.Fixed:
		return session.FixedPollDuration(dur)

	case schemas.Dynamic:
		return session.DynamicPollDuration(dur)

	default:
		panic("Unknown poll duration type")
	}
}

func genSessionImgID(
	ctx context.Context,
	tx pgx.Tx,
	owner uuid.UUID,
	imgReq api.ImgRequest,
	imgs map[api.ImgRequest]uuid.UUID,
) (uuid.NullUUID, map[api.ImgRequest]uuid.UUID, error) {

	var sessionImgID uuid.NullUUID
	if imgReq >= 0 {
		val, ok := imgs[imgReq]
		if ok {
			sessionImgID = uuid.NullUUID{UUID: val, Valid: true}
		} else {
			imgID, err := db.CreateImageMetadata(tx, ctx, owner)
			if err != nil {
				return sessionImgID, imgs, api.ErrorFromConverters{
					ApiError:   api.Errorf(api.ErrInternal, ""),
					StatusCode: http.StatusInternalServerError,
					LogMessage: fmt.Sprintf("failed to create img metadata: %s", err),
				}
			}
			sessionImgID = imgID
			imgs[imgReq] = imgID.UUID
		}
	}
	return sessionImgID, imgs, nil
}
