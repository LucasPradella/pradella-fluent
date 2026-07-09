package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
)

// CSRFCookieName is the JS-readable cookie holding the double-submit token.
const CSRFCookieName = "fluentdev_csrf"

// NewCSRFToken mints a random double-submit token.
func NewCSRFToken() string {
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	return base64.RawURLEncoding.EncodeToString(buf)
}

// SetCSRFCookie issues the double-submit cookie (not HttpOnly by design —
// the SPA reads it to echo the X-CSRF-Token header; SameSite=Lax).
func SetCSRFCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    NewCSRFToken(),
		Path:     "/",
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// CSRF enforces the double-submit pattern on state-changing requests:
// the X-CSRF-Token header must equal the csrf cookie (research R5).
func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		c, err := r.Cookie(CSRFCookieName)
		header := r.Header.Get("X-CSRF-Token")
		if err != nil || header == "" ||
			subtle.ConstantTimeCompare([]byte(c.Value), []byte(header)) != 1 {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"type":"about:blank","title":"Falha de CSRF","status":403,"detail":"Token CSRF ausente ou inválido."}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
