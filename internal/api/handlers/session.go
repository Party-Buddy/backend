package handlers

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"party-buddy/internal/api/base"
	"party-buddy/internal/api/middleware"
	"party-buddy/internal/schemas"
	"party-buddy/internal/schemas/api"
	"party-buddy/internal/session"
	"party-buddy/internal/validate"
	"party-buddy/internal/ws"
)

type SessionConnectHandler struct{}

func (sch SessionConnectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	manager := middleware.ManagerFromContext(r.Context())

	var sid session.SessionID
	strID := r.URL.Query().Get("session-id")
	if strID == "" {
		code := r.URL.Query().Get("invite-code")
		if code == "" {
			msg := "no query params provided"
			base.WriteErrorResponse(w, http.StatusBadRequest, api.ErrParamMissing, msg)
			log.Printf("request: %v %s -> err: %v", r.Method, r.URL, msg)
			return
		}

		var ok bool
		manager.Storage().Atomically(func(s *session.UnsafeStorage) {
			sid, ok = s.SidByInviteCode(session.InviteCode(code))
		})
		if !ok {
			msg := "invalid invite code or session identifier"
			base.WriteErrorResponse(w, http.StatusNotFound, api.ErrNotFound, msg)
			log.Printf("request: %v %s -> err: %v", r.Method, r.URL, msg)
			return
		}
	} else {
		id, err := uuid.Parse(strID)
		if err != nil {
			msg := "invalid invite code or session identifier"
			base.WriteErrorResponse(w, http.StatusNotFound, api.ErrNotFound, msg)
			log.Printf("request: %v %s -> err: %v", r.Method, r.URL, msg)
			return
		}
		sid = session.SessionID(id)

		var exists bool
		manager.Storage().Atomically(func(s *session.UnsafeStorage) {
			exists = s.SessionExists(sid)
		})
		if !exists {
			msg := "invalid invite code or session identifier"
			base.WriteErrorResponse(w, http.StatusNotFound, api.ErrNotFound, msg)
			log.Printf("request: %v %s -> err: %v", r.Method, r.URL, msg)
			return
		}
	}

	authInfo := middleware.AuthInfoFromContext(r.Context())

	if !websocket.IsWebSocketUpgrade(r) {
		msg := "bad Upgrade Header"
		base.WriteErrorResponse(w, http.StatusUpgradeRequired, api.ErrInvalidUpgrade, msg)
		log.Printf("request: %v %s -> err: %v", r.Method, r.URL, msg)
		return
	}

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		Error: func(w http.ResponseWriter, r *http.Request, status int, cause error) {
			base.WriteErrorResponse(w, status, api.ErrUpgradeFailed, cause.Error())
		},
	}

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("request: %v %v -> err after upgrade: %v", r.Method, r.URL.String(), err)
		return
	}

	info := ws.NewConn(log.Default(), manager, wsConn, session.ClientID(authInfo.ID), sid)
	f, _ := validate.FromContext(r.Context())
	info.StartReadAndWriteConn(f)
	log.Printf("request: %v %v -> OK", r.Method, r.URL.String())
}

type SessionCreateHandler struct{}

func (sch SessionCreateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		base.WriteErrorResponse(w, http.StatusInternalServerError, api.ErrInternal, "failed to read request body")
		log.Printf("request: %v %s -> failed to read request body with err: %v", r.Method, r.URL, err)
		return
	}

	var baseReq schemas.BaseCreateSessionRequest
	err = api.Parse(r.Context(), &baseReq, bytes, true)
	if err != nil {
		var dto api.Error
		errors.As(err, &dto)
		base.WriteErrorResponse(w, http.StatusBadRequest, dto.Kind, dto.Message)
		log.Printf("request: %v %s -> err: %v", r.Method, r.URL, dto)
		return
	}

	if baseReq.GameType == nil {
		panic("unexpected nil for game type")
	}

	switch *baseReq.GameType {
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

	var dto api.Error
	errors.As(err, &dto)
	base.WriteErrorResponse(w, http.StatusBadRequest, dto.Kind, dto.Message)
	log.Printf("request: %v %s -> err: %v", r.Method, r.URL, dto)
}

func handlePublicReq(w http.ResponseWriter, r *http.Request, publicReq schemas.PublicCreateSessionRequest) {
	encoder := json.NewEncoder(w)
	tx := middleware.TxFromContext(r.Context())

	game, err := gameIDToSessionGame(r.Context(), tx, *publicReq.GameID)
	if err != nil {
		var errConv api.ErrorFromConverters
		errors.As(err, &errConv)
		log.Printf("request: %v %s -> err: %v", r.Method, r.URL, errConv)
		base.WriteErrorResponse(w, errConv.StatusCode, errConv.ApiError.Kind, errConv.ApiError.Message)
		return
	}

	authInfo := middleware.AuthInfoFromContext(r.Context())
	manager := middleware.ManagerFromContext(r.Context())
	_, code, err := manager.NewSession(
		r.Context(),
		tx,
		&game,
		session.ClientID(authInfo.ID),
		*publicReq.RequireReady,
		int(*publicReq.PlayerCount))
	if err != nil {
		base.WriteErrorResponse(w, http.StatusInternalServerError, api.ErrInternal, "failed to create session")
		log.Printf("request: %v %s -> err: %v", r.Method, r.URL, err)
		return
	}
	err = tx.Commit(r.Context())
	if err != nil {
		base.WriteErrorResponse(w, http.StatusInternalServerError, api.ErrInternal, "failed to create session")
		log.Printf("request: %v %s -> err: %v", r.Method, r.URL, err)
		return
	}

	req := api.SessionCreateResponse{InviteCode: string(code), ImgRequests: []api.ImgReqResponse{}}
	log.Printf("request: %v %s -> OK", r.Method, r.URL)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = encoder.Encode(req)
}

func handlePrivateReq(w http.ResponseWriter, r *http.Request, privateReq schemas.PrivateCreateSessionRequest) {
	encoder := json.NewEncoder(w)
	tx := middleware.TxFromContext(r.Context())
	authInfo := middleware.AuthInfoFromContext(r.Context())

	game, imgResps, err := toSessionGame(r.Context(), tx, authInfo.ID, *privateReq.Game)
	if err != nil {
		var errConv api.ErrorFromConverters
		errors.As(err, &errConv)
		log.Printf("request: %v %s -> err: %v", r.Method, r.URL, errConv)
		base.WriteErrorResponse(w, errConv.StatusCode, errConv.ApiError.Kind, errConv.ApiError.Message)
		return
	}

	manager := middleware.ManagerFromContext(r.Context())
	_, code, err := manager.NewSession(
		r.Context(),
		tx,
		&game,
		session.ClientID(authInfo.ID),
		*privateReq.RequireReady,
		int(*privateReq.PlayerCount))
	if err != nil {
		base.WriteErrorResponse(w, http.StatusInternalServerError, api.ErrInternal, "failed to create session")
		log.Printf("request: %v %s -> err: %v", r.Method, r.URL, err)
		return
	}
	err = tx.Commit(r.Context())
	if err != nil {
		base.WriteErrorResponse(w, http.StatusInternalServerError, api.ErrInternal, "failed to create session")
		log.Printf("request: %v %s -> err: %v", r.Method, r.URL, err)
		return
	}

	req := api.SessionCreateResponse{InviteCode: string(code), ImgRequests: imgResps}
	log.Printf("request: %v %s -> OK", r.Method, r.URL)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = encoder.Encode(req)
}
