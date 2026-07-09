package httpapi

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/pradella/fluentdev/backend/internal/adapter/http/middleware"
	"github.com/pradella/fluentdev/backend/internal/infra/oauth"
	"github.com/pradella/fluentdev/backend/internal/usecase/auth"
	"github.com/pradella/fluentdev/backend/internal/usecase/dashboard"
	"github.com/pradella/fluentdev/backend/internal/usecase/lessons"
	placementuc "github.com/pradella/fluentdev/backend/internal/usecase/placement"
	speechuc "github.com/pradella/fluentdev/backend/internal/usecase/speech"
)

// Deps is everything the router needs, injected by the composition root.
type Deps struct {
	Logger        *slog.Logger
	Auth          *auth.Service
	Placement     *placementuc.Service
	Lessons       *lessons.Service
	Speech        *speechuc.Service
	Dashboard     *dashboard.Service
	OAuth         oauth.Providers
	CookieName    string
	SecureCookies bool
	AppBaseURL    string
	SessionTTL    time.Duration
	// StaticDir, when non-empty, enables SPA serving: static assets are served
	// from this directory and unmatched non-/api/ paths return index.html.
	StaticDir string
}

// NewRouter assembles the /api/v1 surface with the full middleware chain.
// Deny by default: everything outside /auth/* requires a session.
func NewRouter(d Deps) http.Handler {
	authH := &authHandlers{
		svc:           d.Auth,
		providers:     d.OAuth,
		cookieName:    d.CookieName,
		secureCookies: d.SecureCookies,
		appBaseURL:    d.AppBaseURL,
		sessionTTL:    d.SessionTTL,
	}
	placementH := &placementHandlers{svc: d.Placement}
	contentH := &contentHandlers{svc: d.Lessons}
	speechH := &speechHandlers{svc: d.Speech}
	dashboardH := &dashboardHandlers{svc: d.Dashboard}

	// Budgets: auth is brute-force sensitive; speech is cost-bearing (R6).
	authLimiter := middleware.NewRateLimiter(10, 10, d.Logger)   // 10/min per IP
	speechLimiter := middleware.NewRateLimiter(20, 10, d.Logger) // 20/min per user

	requireAuth := middleware.Auth(d.Auth, d.CookieName)

	r := chi.NewRouter()
	r.Use(middleware.Recover(d.Logger))
	r.Use(middleware.RequestID)
	r.Use(middleware.Logging(d.Logger))
	r.Use(middleware.SecurityHeaders)

	r.NotFound(makeNotFoundHandler(d.StaticDir))
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		writeProblem(w, http.StatusMethodNotAllowed, "Método não permitido", "")
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Public auth endpoints (no session, no CSRF — no ambient credential).
		r.Group(func(r chi.Router) {
			r.Use(authLimiter.LimitByIP)
			r.Post("/auth/register", authH.register)
			r.Post("/auth/login", authH.login)
			r.Get("/auth/oauth/{provider}/start", authH.oauthStart)
			r.Get("/auth/oauth/{provider}/callback", authH.oauthCallback)
		})

		// Authenticated endpoints (session + CSRF on state changes).
		r.Group(func(r chi.Router) {
			r.Use(requireAuth)
			r.Use(middleware.CSRF)

			r.Post("/auth/logout", authH.logout)
			r.Get("/me", authH.me)

			r.Get("/placement/session", placementH.current)
			r.Post("/placement/session", placementH.start)
			r.Post("/placement/session/answers", placementH.answer)

			r.Get("/tracks", contentH.tracks)
			r.Get("/lessons/{lessonId}", contentH.lesson)
			r.Post("/exercises/{exerciseId}/attempts", contentH.attempt)

			r.Group(func(r chi.Router) {
				r.Use(speechLimiter.LimitByKeyFunc(func(req *http.Request) string {
					if u, ok := middleware.CurrentUser(req.Context()); ok {
						return "user:" + u.ID.String()
					}
					return ""
				}))
				r.Post("/exercises/{exerciseId}/speech-attempts", speechH.attempt)
			})

			r.Get("/dashboard", dashboardH.dashboard)
			r.Get("/review-queue", dashboardH.reviewQueue)
		})
	})

	return r
}

// makeNotFoundHandler returns a JSON 404 for /api/ paths and a SPA handler for
// everything else when staticDir is set; pure JSON 404 otherwise.
func makeNotFoundHandler(staticDir string) http.HandlerFunc {
	if staticDir == "" {
		return func(w http.ResponseWriter, r *http.Request) {
			writeProblem(w, http.StatusNotFound, "Não encontrado", "")
		}
	}
	spaH := newSPAHandler(staticDir)
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeProblem(w, http.StatusNotFound, "Não encontrado", "")
			return
		}
		spaH.ServeHTTP(w, r)
	}
}

// newSPAHandler serves static files from root; paths that don't resolve to a
// real file fall back to index.html so React Router can handle them.
func newSPAHandler(root string) http.Handler {
	fileServer := http.FileServer(http.Dir(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := filepath.Join(root, filepath.Clean("/"+r.URL.Path))
		if _, err := os.Stat(target); os.IsNotExist(err) {
			http.ServeFile(w, r, filepath.Join(root, "index.html"))
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}
