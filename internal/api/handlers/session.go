package handlers

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"party-buddy/internal/api/middleware"
	"party-buddy/internal/schemas"
	"party-buddy/internal/schemas/api"
	"party-buddy/internal/session"
	"party-buddy/internal/ws"
)

type SessionConnectHandler struct{}

func (sch SessionConnectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	encoder := json.NewEncoder(w)

	manager := middleware.ManagerFromContext(r.Context())

	var sid session.SessionID
	strID := r.URL.Query().Get("session-id")
	if strID == "" {
		code := r.URL.Query().Get("invite-code")
		if code == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			dto := api.Errorf(api.ErrParamMissing, "no query params provided")
			_ = encoder.Encode(dto)
			log.Printf("request: %v %s -> err: %v", r.Method, r.URL, dto)
			return
		}

		var ok bool
		manager.Storage().Atomically(func(s *session.UnsafeStorage) {
			sid, ok = s.SidByInviteCode(session.InviteCode(code))
		})
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			dto := api.Errorf(api.ErrNotFound, "invalid invite code or session identifier")
			_ = encoder.Encode(dto)
			log.Printf("request: %v %s -> err: %v", r.Method, r.URL, dto)
			return
		}
	} else {
		id, err := uuid.Parse(strID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			dto := api.Errorf(api.ErrNotFound, "invalid invite code or session identifier")
			_ = encoder.Encode(dto)
			log.Printf("request: %v %s -> err: %v", r.Method, r.URL, dto)
			return
		}
		sid = session.SessionID(id)

		var exists bool
		manager.Storage().Atomically(func(s *session.UnsafeStorage) {
			exists = s.SessionExists(sid)
		})
		if !exists {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			dto := api.Errorf(api.ErrNotFound, "invalid invite code or session identifier")
			_ = encoder.Encode(dto)
			log.Printf("request: %v %s -> err: %v", r.Method, r.URL, dto)
			return
		}
	}

	authInfo := middleware.AuthInfoFromContext(r.Context())

	if !websocket.IsWebSocketUpgrade(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUpgradeRequired)
		dto := api.Errorf(api.ErrInvalidUpgrade, "bad Upgrade Header")
		_ = encoder.Encode(dto)
		log.Printf("request: %v %s -> err: %v", r.Method, r.URL, dto)
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
		log.Printf("request: %v %v -> err after upgrade: %v", r.Method, r.URL.String(), err)
		return
	}

	info := ws.NewConnInfo(manager, wsConn, session.ClientID(authInfo.ID), sid)
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
		log.Printf("request: %v %s -> err: %v", r.Method, r.URL, dto)
		return
	}

	var baseReq schemas.BaseCreateSessionRequest
	err = api.Parse(r.Context(), &baseReq, bytes, true)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		var dto api.Error
		errors.As(err, &dto)
		_ = encoder.Encode(dto)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	var dto api.Error
	errors.As(err, &dto)
	_ = encoder.Encode(dto)
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(errConv.StatusCode)
		_ = encoder.Encode(errConv.ApiError)
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
		log.Printf("request: %v %s -> err creating session: %v", r.Method, r.URL, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		dto := api.Errorf(api.ErrInternal, "failed to create session")
		_ = encoder.Encode(dto)
		return
	}
	err = tx.Commit(r.Context())
	if err != nil {
		log.Printf("request: %v %s -> err creating session: %v", r.Method, r.URL, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		dto := api.Errorf(api.ErrInternal, "failed to create session")
		_ = encoder.Encode(dto)
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(errConv.StatusCode)
		_ = encoder.Encode(errConv.ApiError)
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
		log.Printf("request: %v %s -> err creating session: %v", r.Method, r.URL, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		dto := api.Errorf(api.ErrInternal, "failed to create session")
		_ = encoder.Encode(dto)
		return
	}
	err = tx.Commit(r.Context())
	if err != nil {
		log.Printf("request: %v %s -> err creating session: %v", r.Method, r.URL, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		dto := api.Errorf(api.ErrInternal, "failed to create session")
		_ = encoder.Encode(dto)
		return
	}

	req := api.SessionCreateResponse{InviteCode: string(code), ImgRequests: imgResps}
	log.Printf("request: %v %s -> OK", r.Method, r.URL)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = encoder.Encode(req)
}
