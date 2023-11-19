package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"party-buddy/internal/api"
	"party-buddy/internal/api/handlers"
	"party-buddy/internal/db"
)

type DBUsingMiddleware struct {
	ctx  context.Context
	pool *db.DBPool
}

func (dbm *DBUsingMiddleware) Middleware(next handlers.DBUsingHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoder := json.NewEncoder(w)

		tx, err := dbm.pool.Pool().Begin(dbm.ctx)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			dto := api.Errorf(api.ErrInternal, "failed to start transaction")
			_ = encoder.Encode(dto)
			return
		}

		next.SetContext(dbm.ctx)
		next.SetTx(tx)

		next.ServeHTTP(w, r)

		_ = tx.Rollback(dbm.ctx)
	})
}
