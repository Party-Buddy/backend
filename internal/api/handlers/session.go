package handlers

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"party-buddy/internal/api"
	"party-buddy/internal/api/middleware"
	"party-buddy/internal/session"
	"party-buddy/internal/ws"
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

	if websocket.IsWebSocketUpgrade(r) {
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
	}

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		dto := api.Errorf(api.ErrInternal, "failed to upgrade connection to web socket")
		_ = encoder.Encode(dto)
		log.Printf("request: %v %v -> err: %v", r.Method, r.URL.String(), dto.Error())
		return
	}

	info := ws.NewConnInfo(manager, wsConn, session.ClientId(authInfo.ID), sid)
	info.StartReadAndWriteConn(r.Context())
	log.Printf("request: %v %v -> OK", r.Method, r.URL.String())
}
