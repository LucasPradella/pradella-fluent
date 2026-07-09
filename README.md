# FluentDev — English Learning PWA for Tech Professionals

Installable, offline-resilient PWA that teaches communicative and technical
English to Brazilian Portuguese speakers (personas: dev júnior, viajante
executivo). UI in pt-BR, learning content in en-US.

**Feature spec**: [`specs/001-fluentdev-pwa/spec.md`](specs/001-fluentdev-pwa/spec.md) ·
**Quickstart & validation**: [`specs/001-fluentdev-pwa/quickstart.md`](specs/001-fluentdev-pwa/quickstart.md)

## What it does

- **Adaptive placement test** (CAMST band walk A1–C1, ≤12 questions) assigns
  Basic/Intermediate/Advanced and unlocks the matching tracks.
- **Task-based lessons** (travel + tech themes): writing with typo tolerance,
  listening comprehension (choice + word ordering), XP and immutable
  progress logging.
- **Speaking practice**: record ≤30 s, transcription via Groq Whisper with
  OpenAI failover, similarity scoring (1−WER, pass ≥80%), missed-word
  highlights.
- **Retention loop**: daily streak, 90-day activity heatmap, spaced review
  (1→3→7→21 days).
- **PWA**: home-screen install (Android/iOS), offline shell with cached
  progress, offline writes replayed idempotently via an outbox.

## Stack

| Layer | Tech |
|-------|------|
| Backend | Go 1.24, chi, pgx/v5 + sqlc, golang-migrate, argon2id, slog |
| Frontend | React 19, TypeScript 5, Vite, vite-plugin-pwa (Workbox), TanStack Query, Dexie, Zustand |
| Database | PostgreSQL 16 (source of truth; IndexedDB is cache only) |
| Speech | Groq `whisper-large-v3-turbo` → OpenAI `gpt-4o-mini-transcribe` failover |

Backend follows clean architecture: `domain` → `usecase` → `adapter` → `infra`,
dependencies pointing inward; scoring rules are pure functions in `internal/domain`.

## Run locally

```bash
# 1. Database (set POSTGRES_PORT if 5432 is taken)
docker compose up -d postgres

# 2. Backend — migrations + seed (20 lessons, 75-question placement bank), :8080
cd backend
cp .env.example .env   # fill in keys; never commit .env
go run ./cmd/api -migrate -seed

# 3. Frontend — dev server :5173 proxying /api → :8080
cd frontend
npm install && npm run dev

# Production-like PWA (service worker active)
npm run build && npm run preview
```

## Tests & gates

```bash
# Backend
cd backend
make lint            # golangci-lint + go vet + govulncheck
make test            # unit + httptest contract tests
make test-integration# real Postgres via testcontainers (Docker required)
make coverage        # gate: >=80% on internal/domain + internal/usecase

# Frontend
cd frontend
npm run lint && npm run typecheck
npm run test         # Vitest + Testing Library
npm run test:e2e     # Playwright: US1–US5 journeys, offline, axe-core a11y
```

Security: OWASP ASVS L1 baseline — cookie sessions + CSRF double-submit,
argon2id, parameterized SQL only, rate limits on auth/speech, security-header
middleware, no PII in logs. Baseline scan: `scripts/zap-baseline.sh`.
