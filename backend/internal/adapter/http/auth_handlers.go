package httpapi

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/oauth2"

	"github.com/pradella/fluentdev/backend/internal/adapter/http/middleware"
	"github.com/pradella/fluentdev/backend/internal/domain/user"
	"github.com/pradella/fluentdev/backend/internal/infra/oauth"
	"github.com/pradella/fluentdev/backend/internal/usecase/auth"
)

// authHandlers serves /auth/* and /me.
type authHandlers struct {
	svc           *auth.Service
	providers     oauth.Providers
	cookieName    string
	secureCookies bool
	appBaseURL    string
	sessionTTL    time.Duration
}

// meDTO is the contract User schema.
type meDTO struct {
	ID               string  `json:"id"`
	Email            string  `json:"email"`
	DisplayName      string  `json:"displayName"`
	ProficiencyLevel *string `json:"proficiencyLevel"`
	CurrentStreak    int     `json:"currentStreak"`
	LongestStreak    int     `json:"longestStreak"`
}

func toMeDTO(u user.User) meDTO {
	dto := meDTO{
		ID:            u.ID.String(),
		Email:         u.Email,
		DisplayName:   u.DisplayName,
		CurrentStreak: u.CurrentStreak,
		LongestStreak: u.LongestStreak,
	}
	if u.Level != "" {
		lvl := string(u.Level)
		dto.ProficiencyLevel = &lvl
	}
	return dto
}

func (h *authHandlers) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(h.sessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
	middleware.SetCSRFCookie(w, h.secureCookies)
}

func (h *authHandlers) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: h.cookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: h.secureCookies, SameSite: http.SameSiteLaxMode,
	})
}

// POST /auth/register
func (h *authHandlers) register(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"displayName"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if len(in.Email) > 254 {
		writeProblem(w, http.StatusBadRequest, "Requisição inválida", "e-mail longo demais")
		return
	}
	u, token, err := h.svc.Register(r.Context(), in.Email, in.Password, in.DisplayName)
	if err != nil {
		writeError(w, r, err)
		return
	}
	h.setSessionCookie(w, token)
	writeJSON(w, http.StatusCreated, toMeDTO(u))
}

// POST /auth/login
func (h *authHandlers) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	u, token, err := h.svc.Login(r.Context(), in.Email, in.Password)
	if err != nil {
		// Generic message — OWASP A07, no enumeration.
		writeProblem(w, http.StatusUnauthorized, "Credenciais inválidas", "E-mail ou senha incorretos.")
		return
	}
	h.setSessionCookie(w, token)
	writeJSON(w, http.StatusOK, toMeDTO(u))
}

// POST /auth/logout
func (h *authHandlers) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(h.cookieName); err == nil {
		_ = h.svc.Logout(r.Context(), c.Value)
	}
	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// GET /me
func (h *authHandlers) me(w http.ResponseWriter, r *http.Request) {
	u, ok := middleware.CurrentUser(r.Context())
	if !ok {
		writeProblem(w, http.StatusUnauthorized, "Não autenticado", "")
		return
	}
	writeJSON(w, http.StatusOK, toMeDTO(u))
}

const (
	oauthStateCookie    = "fluentdev_oauth_state"
	oauthVerifierCookie = "fluentdev_oauth_verifier"
)

// GET /auth/oauth/{provider}/start
func (h *authHandlers) oauthStart(w http.ResponseWriter, r *http.Request) {
	p, ok := h.providers[chi.URLParam(r, "provider")]
	if !ok {
		writeProblem(w, http.StatusBadRequest, "Requisição inválida", "provedor desconhecido")
		return
	}
	state := middleware.NewCSRFToken()
	verifier := oauth2.GenerateVerifier()

	http.SetCookie(w, &http.Cookie{
		Name: oauthStateCookie, Value: state, Path: "/api/v1/auth/oauth",
		MaxAge: 600, HttpOnly: true, Secure: h.secureCookies, SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name: oauthVerifierCookie, Value: verifier, Path: "/api/v1/auth/oauth",
		MaxAge: 600, HttpOnly: true, Secure: h.secureCookies, SameSite: http.SameSiteLaxMode,
	})

	url := p.Config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
	http.Redirect(w, r, url, http.StatusFound)
}

// GET /auth/oauth/{provider}/callback
func (h *authHandlers) oauthCallback(w http.ResponseWriter, r *http.Request) {
	p, ok := h.providers[chi.URLParam(r, "provider")]
	if !ok {
		writeProblem(w, http.StatusBadRequest, "Requisição inválida", "provedor desconhecido")
		return
	}
	stateCookie, err := r.Cookie(oauthStateCookie)
	if err != nil || stateCookie.Value == "" || r.URL.Query().Get("state") != stateCookie.Value {
		writeProblem(w, http.StatusBadRequest, "Requisição inválida", "state inválido")
		return
	}
	verifierCookie, err := r.Cookie(oauthVerifierCookie)
	if err != nil || verifierCookie.Value == "" {
		writeProblem(w, http.StatusBadRequest, "Requisição inválida", "verificador PKCE ausente")
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		writeProblem(w, http.StatusBadRequest, "Requisição inválida", "código de autorização ausente")
		return
	}

	tok, err := p.Config.Exchange(r.Context(), code, oauth2.VerifierOption(verifierCookie.Value))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "Requisição inválida", "falha na troca do código OAuth")
		return
	}
	profile, err := p.FetchProfile(r.Context(), p.Config.TokenSource(r.Context(), tok))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "Requisição inválida", "não foi possível obter o perfil do provedor")
		return
	}

	u, token, _, err := h.svc.HandleOAuthUser(r.Context(), profile)
	if err != nil {
		writeError(w, r, err)
		return
	}
	h.setSessionCookie(w, token)

	// New users (no level yet) land on placement; the rest on the dashboard.
	dest := h.appBaseURL + "/dashboard"
	if u.Level == "" {
		dest = h.appBaseURL + "/placement"
	}
	http.Redirect(w, r, dest, http.StatusFound)
}
