package middleware

import (
	"github.com/cohesivestack/valgo"
	"net/http"
	"party-buddy/internal/validate"
)

type ValidateMiddleware struct {
	Factory *valgo.ValidationFactory
}

func (v *ValidateMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := validate.NewContext(r.Context(), v.Factory)
		rWithValidate := r.WithContext(ctx)
		next.ServeHTTP(w, rWithValidate)
	})
}
