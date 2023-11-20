package middleware

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"net/http"
	"party-buddy/internal/api"
	"party-buddy/internal/db"
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
			dto := api.Errorf(api.ErrInternal, "failed to start transaction")
			_ = encoder.Encode(dto)
			return
		}
		defer tx.Rollback(r.Context())

		ctx := context.WithValue(r.Context(), txKey, tx)
		rWithDb := r.WithContext(ctx)

		next.ServeHTTP(w, rWithDb)
	})
}

func TxFromContext(ctx context.Context) pgx.Tx {
	return ctx.Value(txKey).(pgx.Tx)
}
