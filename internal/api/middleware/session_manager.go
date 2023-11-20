package middleware

import (
	"context"
	"net/http"
	"party-buddy/internal/session"
)

type managerKeyType int

var managerKey txKeyType

// ManagerUsingMiddleware is a middleware for db usage
type ManagerUsingMiddleware struct {
	Manager *session.Manager
}

// Middleware starts transaction and puts the tx (pgx.Tx) and ctx (context.Context) to request context
func (mm ManagerUsingMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), managerKey, mm.Manager)
		rWithDb := r.WithContext(ctx)

		next.ServeHTTP(w, rWithDb)
	})
}

func ManagerFromContext(ctx context.Context) *session.Manager {
	return ctx.Value(managerKey).(*session.Manager)
}
