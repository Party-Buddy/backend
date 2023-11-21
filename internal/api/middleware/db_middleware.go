package middleware

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"log"
	"net/http"
	"party-buddy/internal/db"
	"party-buddy/internal/schemas/api"
)

type txKeyType int

var txKey txKeyType

// DBUsingMiddleware is a middleware for db usage
type DBUsingMiddleware struct {
	Pool *db.DBPool
}

// Middleware starts transaction and puts the tx (pgx.Tx) and ctx (context.Context) to request context
func (dbm DBUsingMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoder := json.NewEncoder(w)

		tx, err := dbm.Pool.Pool().Begin(r.Context())
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			dto := api.Errorf(api.ErrInternal, "")
			_ = encoder.Encode(dto)
			log.Printf("request: %v %v -> err: %v", r.Method, r.URL.String(),
				api.Errorf(api.ErrInternal, "failed to start transaction").Error())

			return
		}
		defer tx.Rollback(r.Context())

		ctx := context.WithValue(r.Context(), txKey, tx)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func TxFromContext(ctx context.Context) pgx.Tx {
	return ctx.Value(txKey).(pgx.Tx)
}
