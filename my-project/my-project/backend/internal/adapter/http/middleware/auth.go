package middleware

import (
	"context"
	"net/http"

	"github.com/pradella/fluentdev/backend/internal/domain/user"
	"github.com/pradella/fluentdev/backend/internal/usecase/auth"
)

// userKey carries the authenticated user in the request context.
const userKey ctxKey = "user"

// Auth resolves the session cookie into a user (deny by default) and
// stores it in the request context. 401 problem+json when unauthenticated.
func Auth(svc *auth.Service, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(cookieName)
			if err != nil {
				unauthorized(w)
				return
			}
			u, err := svc.Authenticate(r.Context(), c.Value)
			if err != nil {
				unauthorized(w)
				return
			}
			ctx := context.WithValue(r.Context(), userKey, u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"type":"about:blank","title":"Não autenticado","status":401}`))
}

// CurrentUser returns the authenticated user placed by Auth.
func CurrentUser(ctx context.Context) (user.User, bool) {
	u, ok := ctx.Value(userKey).(user.User)
	return u, ok
}
