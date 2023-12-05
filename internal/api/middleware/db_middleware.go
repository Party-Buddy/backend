package middleware

import (
	"context"
	"github.com/jackc/pgx/v5"
	"log"
	"net/http"
	"party-buddy/internal/api/base"
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
		tx, err := dbm.Pool.Pool().Begin(r.Context())
		if err != nil {
			base.WriteErrorResponse(w, http.StatusInternalServerError, api.ErrInternal, "internal server error")
			log.Printf("request: %v %s -> falided to start transaction with err: %v", r.Method, r.URL, err)
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
