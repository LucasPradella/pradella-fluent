// Package config loads all runtime configuration from the environment.
// No other package reads environment variables directly.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config is the full runtime configuration of the API process.
type Config struct {
	// HTTP
	Addr       string // listen address, e.g. ":8080"
	AppBaseURL string // frontend origin used for OAuth redirects and CORS

	// Database
	DatabaseURL string

	// Sessions
	SessionCookieName string
	SessionTTL        time.Duration // sliding expiry window
	SecureCookies     bool          // false only for plain-HTTP local dev

	// Transcription (research R2)
	TranscribePrimary string // "groq" | "openai"
	GroqAPIKey        string
	OpenAIAPIKey      string
	GroqBaseURL       string // overridable for tests/fakes
	OpenAIBaseURL     string

	// OAuth (FR-001)
	GitHubClientID     string
	GitHubClientSecret string
	GoogleClientID     string
	GoogleClientSecret string
	OAuthRedirectBase  string // backend base URL for provider callbacks
}

// Load reads configuration from the environment, applying dev-safe defaults
// for everything except DATABASE_URL, which is required.
func Load() (Config, error) {
	cfg := Config{
		Addr:              getEnv("ADDR", ":8080"),
		AppBaseURL:        getEnv("APP_BASE_URL", "http://localhost:5173"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		SessionCookieName: getEnv("SESSION_COOKIE_NAME", "fluentdev_session"),
		SessionTTL:        getDuration("SESSION_TTL", 30*24*time.Hour),
		SecureCookies:     getBool("SECURE_COOKIES", false),
		TranscribePrimary: getEnv("TRANSCRIBE_PRIMARY", "groq"),
		GroqAPIKey:        os.Getenv("GROQ_API_KEY"),
		OpenAIAPIKey:      os.Getenv("OPENAI_API_KEY"),
		GroqBaseURL:       getEnv("GROQ_BASE_URL", "https://api.groq.com/openai/v1"),
		OpenAIBaseURL:     getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),

		GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		OAuthRedirectBase:  getEnv("OAUTH_REDIRECT_BASE", "http://localhost:8080"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("config: DATABASE_URL is required")
	}
	if cfg.TranscribePrimary != "groq" && cfg.TranscribePrimary != "openai" {
		return Config{}, fmt.Errorf("config: TRANSCRIBE_PRIMARY must be groq or openai, got %q", cfg.TranscribePrimary)
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
