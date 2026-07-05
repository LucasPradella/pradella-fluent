// Package httpapi is the chi-based HTTP adapter: router, middleware and
// handlers speaking the /api/v1 contract (contracts/openapi.yaml).
package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/pradella/fluentdev/backend/internal/domain"
)

// Problem is an RFC 9457 problem+json body.
type Problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// writeProblem emits a problem+json response.
func writeProblem(w http.ResponseWriter, status int, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Problem{
		Type:   "about:blank",
		Title:  title,
		Status: status,
		Detail: detail,
	})
}

// writeError maps domain sentinels to HTTP problems. Internal errors are
// logged with context but never leak details to the client.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeProblem(w, http.StatusNotFound, "Não encontrado", "")
	case errors.Is(err, domain.ErrForbidden):
		writeProblem(w, http.StatusForbidden, "Acesso negado", "Este conteúdo está bloqueado para o seu nível atual.")
	case errors.Is(err, domain.ErrConflict):
		writeProblem(w, http.StatusConflict, "Conflito", err.Error())
	case errors.Is(err, domain.ErrInvalid):
		writeProblem(w, http.StatusBadRequest, "Requisição inválida", err.Error())
	case errors.Is(err, domain.ErrUnauthorized):
		writeProblem(w, http.StatusUnauthorized, "Não autenticado", "")
	default:
		slog.ErrorContext(r.Context(), "internal error",
			slog.String("path", r.URL.Path),
			slog.String("error", err.Error()))
		writeProblem(w, http.StatusInternalServerError, "Erro interno", "")
	}
}

// writeJSON emits a JSON response body.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// decodeJSON parses a bounded JSON request body.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeProblem(w, http.StatusBadRequest, "Requisição inválida", "corpo JSON malformado")
		return false
	}
	return true
}
