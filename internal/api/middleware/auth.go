package middleware

import (
	"context"
	"github.com/google/uuid"
	"log"
	"net/http"
	"party-buddy/internal/api/base"
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
		val := r.Header.Get("Authorization")
		if val == "" {
			msg := "authentication required"
			base.WriteErrorResponse(w, http.StatusUnauthorized, api.ErrAuthRequired, msg)
			log.Printf("request: %v %s -> err: %v", r.Method, r.URL, msg)
			return
		}
		strUUID, found := strings.CutPrefix(val, "Bearer ")
		if !found {
			msg := "provided user id is not valid"
			base.WriteErrorResponse(w, http.StatusUnauthorized, api.ErrUserIDInvalid, msg)
			log.Printf("request: %v %s -> err: %v", r.Method, r.URL, msg)
			return
		}

		userID, err := uuid.Parse(strUUID)
		if err != nil {
			msg := "provided user id is not valid"
			base.WriteErrorResponse(w, http.StatusUnauthorized, api.ErrUserIDInvalid, msg)
			log.Printf("request: %v %s -> err: %v", r.Method, r.URL, msg)
			return
		}

		tx := TxFromContext(r.Context())

		entity, err := db.GetUserByID(r.Context(), tx, userID)
		if err != nil {
			msg := "internal server error while getting user"
			base.WriteErrorResponse(w, http.StatusUnauthorized, api.ErrInternal, msg)
			log.Printf("request: %v %s -> err: %v", r.Method, r.URL, err)
			return
		}
		authInfo := AuthInfo{ID: entity.ID.UUID, Role: entity.Role}

		ctx := context.WithValue(r.Context(), authKey, authInfo)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AuthInfoFromContext(ctx context.Context) AuthInfo {
	return ctx.Value(authKey).(AuthInfo)
}
