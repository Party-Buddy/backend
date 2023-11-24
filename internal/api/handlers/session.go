package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"io"
	"log"
	"net/http"
	"party-buddy/internal/api/middleware"
	"party-buddy/internal/configuration"
	"party-buddy/internal/db"
	"party-buddy/internal/schemas"
	"party-buddy/internal/schemas/api"
	"party-buddy/internal/session"
	"party-buddy/internal/ws"
	"time"
)

type SessionConnectHandler struct{}

func (sch SessionConnectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	encoder := json.NewEncoder(w)

	manager := middleware.ManagerFromContext(r.Context())

	var sid session.SessionId
	strID := r.URL.Query().Get("session-id")
	if strID == "" {
		code := r.URL.Query().Get("invite-code")
		if code == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			dto := api.Errorf(api.ErrParamMissing, "no query params provided")
			_ = encoder.Encode(dto)
			log.Printf("request: %v %v -> err: %v", r.Method, r.URL.String(), dto.Error())
			return
		}
		id, ok := manager.SidByInviteCode(code)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			dto := api.Errorf(api.ErrNotFound, "invalid invite code or session identifier")
			_ = encoder.Encode(dto)
			log.Printf("request: %v %v -> err: %v", r.Method, r.URL.String(), dto.Error())
			return
		}
		sid = id
	} else {
		id, err := uuid.Parse(strID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			dto := api.Errorf(api.ErrNotFound, "invalid invite code or session identifier")
			_ = encoder.Encode(dto)
			log.Printf("request: %v %v -> err: %v", r.Method, r.URL.String(), dto.Error())
			return
		}
		sid = session.SessionId(id)
		if !manager.SessionExists(sid) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			dto := api.Errorf(api.ErrNotFound, "invalid invite code or session identifier")
			_ = encoder.Encode(dto)
			log.Printf("request: %v %v -> err: %v", r.Method, r.URL.String(), dto.Error())
			return
		}
	}

	authInfo := middleware.AuthInfoFromContext(r.Context())

	if !websocket.IsWebSocketUpgrade(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUpgradeRequired)
		dto := api.Errorf(api.ErrInvalidUpgrade, "bad Upgrade Header")
		_ = encoder.Encode(dto)
		log.Printf("request: %v %v -> err: %v", r.Method, r.URL.String(), dto.Error())
		return
	}

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		Error: func(w http.ResponseWriter, r *http.Request, status int, cause error) {
			encoder := json.NewEncoder(w)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			dto := api.Errorf(api.ErrUpgradeFailed, "upgrade failed, err: %v", cause.Error())
			_ = encoder.Encode(dto)
		},
	}

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("request: %v %v -> err after upgrade: %v", r.Method, r.URL.String(), err.Error())
		return
	}

	info := ws.NewConnInfo(manager, wsConn, session.ClientId(authInfo.ID), sid)
	info.StartReadAndWriteConn(r.Context())
	log.Printf("request: %v %v -> OK", r.Method, r.URL.String())
}

type SessionCreateHandler struct{}

func (sch SessionCreateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	encoder := json.NewEncoder(w)

	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		dto := api.Errorf(api.ErrInternal, "failed to read request body")
		_ = encoder.Encode(dto)
		log.Printf("request: %v %v -> err: %v", r.Method, r.URL.String(), dto.Error())
		return
	}

	var baseReq schemas.BaseCreateSessionRequest
	err = api.Parse(r.Context(), &baseReq, bytes, true)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		var dto *api.Error
		errors.As(err, &dto)
		_ = encoder.Encode(dto)
		log.Printf("request: %v %v -> err: %v", r.Method, r.URL.String(), dto.Error())
		return
	}

	switch baseReq.GameType {
	case schemas.Public:
		var publicReq schemas.PublicCreateSessionRequest
		err = api.Parse(r.Context(), &publicReq, bytes, false)
		if err == nil {
			handlePublicReq(w, r, publicReq)
			return
		}

	case schemas.Private:
		var privateReq schemas.PrivateCreateSessionRequest
		err = api.Parse(r.Context(), &privateReq, bytes, false)
		if err == nil {
			handlePrivateReq(w, r, privateReq)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	var dto *api.Error
	errors.As(err, &dto)
	_ = encoder.Encode(dto)
	log.Printf("request: %v %v -> err: %v", r.Method, r.URL.String(), dto.Error())
}

func handlePublicReq(w http.ResponseWriter, r *http.Request, publicReq schemas.PublicCreateSessionRequest) {
	_, _ = fmt.Fprint(w, "Hello, world from public!")
}

func handlePrivateReq(w http.ResponseWriter, r *http.Request, privateReq schemas.PrivateCreateSessionRequest) {
	encoder := json.NewEncoder(w)
	tx := middleware.TxFromContext(r.Context())
	authInfo := middleware.AuthInfoFromContext(r.Context())

	game, imgResps, err := toSessionGame(r.Context(), tx, authInfo.ID, privateReq.Game)
	if err != nil {
		var dto *api.Error
		errors.As(err, &dto)
		log.Printf("request: %v %v -> err: %v", r.Method, r.URL.String(), dto.Error())
		w.Header().Set("Content-Type", "application/json")
		switch dto.Kind {
		case api.ErrTaskInvalid:
			w.WriteHeader(http.StatusBadRequest)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		_ = encoder.Encode(dto)
		return
	}

	manager := middleware.ManagerFromContext(r.Context())
	_, code, _, err := manager.NewSession(
		r.Context(),
		tx,
		&game,
		session.ClientId(authInfo.ID),
		"remove",
		privateReq.RequireReady, int(privateReq.PlayerCount))
	if err != nil {
		log.Printf("request: %v %v -> err creating session: %v", r.Method, r.URL.String(), err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		dto := api.Errorf(api.ErrInternal, "failed to create session")
		_ = encoder.Encode(dto)
		return
	}
	defer tx.Commit(r.Context())

	req := schemas.SessionCreateResponse{InviteCode: string(code), ImgRequests: imgResps}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = encoder.Encode(req)
}

func toSessionGame(
	ctx context.Context,
	tx pgx.Tx,
	owner uuid.UUID,
	gameInfo schemas.FullGameInfo,
) (session.Game, []schemas.ImgReqResponse, error) {

	game := session.Game{}
	game.Name = gameInfo.Name
	game.Description = gameInfo.Description
	imgs := make(map[schemas.ImgRequest]uuid.UUID)
	if gameInfo.ImgRequest >= 0 {
		imgID, err := db.CreateImageMetadata(tx, ctx, owner)
		if err != nil {
			return session.Game{}, nil, api.Errorf(api.ErrInternal, "failed to create img metadata: %v", err.Error())
		}
		game.ImageId = session.ImageId(imgID)
		imgs[gameInfo.ImgRequest] = imgID.UUID
	}
	tasks := make([]session.Task, len(gameInfo.Tasks))
	for i := 0; i < len(gameInfo.Tasks); i++ {
		t, newImgs, err := toSessionTask(ctx, tx, owner, gameInfo.Tasks[i], imgs)
		if err != nil {
			return session.Game{}, nil, err
		}
		imgs = newImgs
		tasks = append(tasks, t)
	}
	imgResps := make([]schemas.ImgReqResponse, 0)
	for k, v := range imgs {
		imgResps = append(imgResps, schemas.ImgReqResponse{ImgRequest: k, ImgURI: configuration.GenImgURI(v)})
	}
	return game, imgResps, nil
}

func toSessionTask(
	ctx context.Context,
	tx pgx.Tx,
	owner uuid.UUID,
	task schemas.BaseTaskWithImgRequest,
	imgs map[schemas.ImgRequest]uuid.UUID,
) (session.Task, map[schemas.ImgRequest]uuid.UUID, error) {
	sessionImgID, newImgs, err := genSessionImgID(ctx, tx, owner, task.ImgRequest, imgs)
	if err != nil {
		return nil, imgs, err
	}
	baseTask := session.BaseTask{
		Name:         task.Name,
		Description:  task.Description,
		ImageId:      session.ImageId(sessionImgID),
		TaskDuration: time.Duration(task.Duration.Secs) * time.Second,
	}

	switch task.Type {
	case schemas.Photo:
		return session.PhotoTask{
			BaseTask:     baseTask,
			PollDuration: toSessionPollDuration(task.PollDuration),
		}, newImgs, nil

	case schemas.Text:
		return session.TextTask{
			BaseTask:     baseTask,
			PollDuration: toSessionPollDuration(task.PollDuration),
		}, newImgs, nil

	case schemas.CheckedText:
		return session.CheckedTextTask{
			BaseTask: baseTask,
			Answer:   task.Answer,
		}, newImgs, nil

	case schemas.Choice:
		return session.ChoiceTask{
			BaseTask:  baseTask,
			Options:   task.Options,
			AnswerIdx: int(task.AnswerIndex),
		}, newImgs, nil

	default:
		return nil, imgs, api.Errorf(api.ErrTaskInvalid, "unknown task type")
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
	imgReq schemas.ImgRequest,
	imgs map[schemas.ImgRequest]uuid.UUID,
) (uuid.NullUUID, map[schemas.ImgRequest]uuid.UUID, error) {

	var sessionImgID uuid.NullUUID
	if imgReq >= 0 {
		val, ok := imgs[imgReq]
		if ok {
			sessionImgID = uuid.NullUUID{UUID: val, Valid: true}
		} else {
			imgID, err := db.CreateImageMetadata(tx, ctx, owner)
			if err != nil {
				return sessionImgID, imgs, api.Errorf(api.ErrInternal, "failed to create img metadata: %v", err.Error())
			}
			sessionImgID = imgID
			imgs[imgReq] = imgID.UUID
		}
	}
	return sessionImgID, imgs, nil
}
