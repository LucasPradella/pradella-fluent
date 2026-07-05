package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Recover converts panics into 500 problem+json responses and logs them
// with the request id (fail loudly, never crash the server).
func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.ErrorContext(r.Context(), "panic recovered",
						slog.String("request_id", GetRequestID(r.Context())),
						slog.Any("panic", rec))
					w.Header().Set("Content-Type", "application/problem+json")
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"type": "about:blank", "title": "Erro interno", "status": 500,
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
