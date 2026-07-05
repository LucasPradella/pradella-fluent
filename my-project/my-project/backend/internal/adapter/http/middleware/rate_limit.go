package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"
)

// bucket is a simple token bucket.
type bucket struct {
	tokens   float64
	lastFill time.Time
}

// RateLimiter is an in-memory per-key token bucket (per-IP or per-user).
// Suitable for a single-instance MVP; swap for a shared store when scaling.
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64 // tokens per second
	capacity float64
	logger   *slog.Logger
}

// NewRateLimiter allows `capacity` bursts refilled at `perMinute` tokens/min.
func NewRateLimiter(perMinute float64, capacity int, logger *slog.Logger) *RateLimiter {
	rl := &RateLimiter{
		buckets:  make(map[string]*bucket),
		rate:     perMinute / 60.0,
		capacity: float64(capacity),
		logger:   logger,
	}
	go rl.gc()
	return rl
}

func (rl *RateLimiter) gc() {
	for range time.Tick(5 * time.Minute) {
		rl.mu.Lock()
		for k, b := range rl.buckets {
			if time.Since(b.lastFill) > 10*time.Minute {
				delete(rl.buckets, k)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow consumes one token for the key, reporting whether it fit the budget.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{tokens: rl.capacity, lastFill: now}
		rl.buckets[key] = b
	}
	b.tokens += now.Sub(b.lastFill).Seconds() * rl.rate
	if b.tokens > rl.capacity {
		b.tokens = rl.capacity
	}
	b.lastFill = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// ClientIP extracts the caller address (trusting the direct peer only).
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// LimitByIP guards an endpoint with a per-IP budget (auth endpoints —
// OWASP A07). Emits a rate-limit event log on rejection.
func (rl *RateLimiter) LimitByIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.Allow(ClientIP(r)) {
			rl.reject(w, r, "ip")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// LimitByKeyFunc guards an endpoint with a caller-derived key (e.g. user id
// for the cost-bearing speech endpoint — OWASP A04).
func (rl *RateLimiter) LimitByKeyFunc(key func(r *http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			k := key(r)
			if k == "" {
				k = ClientIP(r)
			}
			if !rl.Allow(k) {
				rl.reject(w, r, "user")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (rl *RateLimiter) reject(w http.ResponseWriter, r *http.Request, scope string) {
	if rl.logger != nil {
		rl.logger.WarnContext(r.Context(), "rate limit exceeded",
			slog.String("request_id", GetRequestID(r.Context())),
			slog.String("scope", scope),
			slog.String("path", r.URL.Path))
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.Header().Set("Retry-After", "60")
	w.WriteHeader(http.StatusTooManyRequests)
	_, _ = w.Write([]byte(`{"type":"about:blank","title":"Muitas requisições","status":429,"detail":"Tente novamente em instantes."}`))
}
