package middleware

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"log"
	"net/http"
	"party-buddy/internal/db"
	"party-buddy/internal/schemas/api"
	"strings"
)

type authKeyType int

var authKey authKeyType

type AuthInfo struct {
	ID   uuid.UUID
	Role db.UserRole
}

// AuthMiddleware must be applied after DBUsingMiddleware
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoder := json.NewEncoder(w)
		val := r.Header.Get("Authorization")
		if val == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			dto := api.Errorf(api.ErrAuthRequired, "authentication required")
			log.Printf("request: %v %v -> err: %v", r.Method, r.URL.String(), dto.Error())
			_ = encoder.Encode(dto)
			return
		}
		strUUID, found := strings.CutPrefix(val, "Bearer ")
		if !found {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			dto := api.Errorf(api.ErrUserIdInvalid, "provided user id is not valid")
			log.Printf("request: %v %v -> err: %v", r.Method, r.URL.String(), dto.Error())
			_ = encoder.Encode(dto)
			return
		}

		userID, err := uuid.Parse(strUUID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			dto := api.Errorf(api.ErrUserIdInvalid, "provided user id is not valid")
			log.Printf("request: %v %v -> err: %v", r.Method, r.URL.String(), dto.Error())
			_ = encoder.Encode(dto)
			return
		}

		tx := TxFromContext(r.Context())

		entity, err := db.GetUserByID(r.Context(), tx, userID)
		authInfo := AuthInfo{ID: entity.ID.UUID, Role: entity.Role}

		ctx := context.WithValue(r.Context(), authKey, authInfo)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AuthInfoFromContext(ctx context.Context) AuthInfo {
	return ctx.Value(authKey).(AuthInfo)
}
