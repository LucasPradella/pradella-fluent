// Command api is the FluentDev API server: composition root wiring
// config → pgx pool → repositories → usecases → router, with optional
// migration and seed steps (-migrate / -seed).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"

	httpapi "github.com/pradella/fluentdev/backend/internal/adapter/http"
	"github.com/pradella/fluentdev/backend/internal/adapter/postgres"
	"github.com/pradella/fluentdev/backend/internal/infra/config"
	"github.com/pradella/fluentdev/backend/internal/infra/oauth"
	"github.com/pradella/fluentdev/backend/internal/infra/transcriber"
	"github.com/pradella/fluentdev/backend/internal/usecase/auth"
	"github.com/pradella/fluentdev/backend/internal/usecase/dashboard"
	"github.com/pradella/fluentdev/backend/internal/usecase/lessons"
	placementuc "github.com/pradella/fluentdev/backend/internal/usecase/placement"
	speechuc "github.com/pradella/fluentdev/backend/internal/usecase/speech"
	"github.com/pradella/fluentdev/backend/seed"
)

func main() {
	runMigrations := flag.Bool("migrate", false, "apply database migrations before serving")
	runSeed := flag.Bool("seed", false, "seed lessons and placement bank when empty")
	migrationsDir := flag.String("migrations", "migrations", "path to migration files")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := run(*runMigrations, *runSeed, *migrationsDir, logger); err != nil {
		logger.Error("fatal", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(runMigrations, runSeed bool, migrationsDir string, logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if runMigrations {
		m, err := migrate.New("file://"+migrationsDir, cfg.DatabaseURL)
		if err != nil {
			return fmt.Errorf("migrate init: %w", err)
		}
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("migrate up: %w", err)
		}
		logger.Info("migrations applied")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("db pool: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("db ping: %w", err)
	}

	if runSeed {
		if err := seed.Load(ctx, pool, logger); err != nil {
			return err
		}
	}

	// Repositories.
	store := postgres.NewStore(pool)
	userRepo := postgres.NewUserRepo(store)
	sessionRepo := postgres.NewSessionRepo(store)
	placementRepo := postgres.NewPlacementRepo(store)
	contentRepo := postgres.NewContentRepo(store)
	progressRepo := postgres.NewProgressRepo(store)
	dashboardRepo := postgres.NewDashboardRepo(store)

	// Transcriber chain (research R2).
	var primary, secondary transcriber.Transcriber
	groq := transcriber.NewGroq(cfg.GroqBaseURL, cfg.GroqAPIKey, logger)
	openai := transcriber.NewOpenAI(cfg.OpenAIBaseURL, cfg.OpenAIAPIKey, logger)
	if cfg.TranscribePrimary == "openai" {
		primary, secondary = openai, groq
	} else {
		primary, secondary = groq, openai
	}
	failover := transcriber.NewFailover(primary, secondary, logger)

	// Usecases.
	authSvc := auth.New(userRepo, sessionRepo, cfg.SessionTTL)
	placementSvc := placementuc.New(placementRepo)
	lessonsSvc := lessons.New(contentRepo, progressRepo, userRepo)
	speechSvc := speechuc.New(contentRepo, progressRepo, userRepo, failover)
	dashboardSvc := dashboard.New(userRepo, dashboardRepo)

	// OAuth providers (FR-001).
	providers := oauth.Providers{}
	if cfg.GitHubClientID != "" {
		providers["github"] = oauth.NewGitHub(cfg.GitHubClientID, cfg.GitHubClientSecret, cfg.OAuthRedirectBase)
	}
	if cfg.GoogleClientID != "" {
		providers["google"] = oauth.NewGoogle(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.OAuthRedirectBase)
	}

	router := httpapi.NewRouter(httpapi.Deps{
		Logger:        logger,
		Auth:          authSvc,
		Placement:     placementSvc,
		Lessons:       lessonsSvc,
		Speech:        speechSvc,
		Dashboard:     dashboardSvc,
		OAuth:         providers,
		CookieName:    cfg.SessionCookieName,
		SecureCookies: cfg.SecureCookies,
		AppBaseURL:    cfg.AppBaseURL,
		SessionTTL:    cfg.SessionTTL,
	})

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("api listening", slog.String("addr", cfg.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	logger.Info("shutting down")
	return srv.Shutdown(shutdownCtx)
}
