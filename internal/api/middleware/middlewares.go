package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"party-buddy/internal/api"
	"party-buddy/internal/db"
)

// DBUsingMiddleware is a middleware for db usage
type DBUsingMiddleware struct {
	Ctx  context.Context
	Pool *db.DBPool
}

// Middleware starts transaction and puts the tx (pgx.Tx) and ctx (context.Context) to request context
func (dbm DBUsingMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoder := json.NewEncoder(w)

		tx, err := dbm.Pool.Pool().Begin(dbm.Ctx)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			dto := api.Errorf(api.ErrInternal, "failed to start transaction")
			_ = encoder.Encode(dto)
			return
		}

		ctx := context.WithValue(r.Context(), "tx", tx)
		ctx = context.WithValue(ctx, "ctx", dbm.Ctx)

		rWithDb := r.WithContext(ctx)

		next.ServeHTTP(w, rWithDb)

		_ = tx.Rollback(dbm.Ctx)
	})
}
