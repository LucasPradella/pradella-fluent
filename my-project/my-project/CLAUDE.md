# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Auto-updated by spec-kit commands; manual edits between features are fine.

## Project

Installable, offline-resilient PWA teaching communicative/technical English to Brazilian
Portuguese speakers (personas: dev júnior, viajante executivo). Spec-driven repo — see
`.specify/memory/constitution.md` (v1.0.0) and `specs/001-fluentdev-pwa/`.

## Active Feature: 001-fluentdev-pwa

- Spec: `specs/001-fluentdev-pwa/spec.md` (5 user stories P1–P5, FR-001…FR-025)
- Plan: `specs/001-fluentdev-pwa/plan.md` | Research: `research.md` |
  Data model: `data-model.md` | Contract: `contracts/openapi.yaml` |
  Validation: `quickstart.md`

## Tech Stack

- **Backend** (`backend/`): Go 1.24, chi, pgx/v5 + sqlc, golang-migrate, slog (JSON),
  argon2id, golang.org/x/oauth2 (GitHub/Google, PKCE). Clean architecture:
  `internal/domain` → `internal/usecase` → `internal/adapter/{http,postgres}` →
  `internal/infra/{config,transcriber,oauth}`. Dependency direction points inward; domain
  has zero external deps.
- **Frontend** (`frontend/`): React 19 + TypeScript 5 + Vite, vite-plugin-pwa (Workbox),
  TanStack Query (SWR), Dexie (IndexedDB cache + offline outbox), Zustand.
- **DB**: PostgreSQL 16 (source of truth; IndexedDB is cache only).
- **Transcription**: `Transcriber` port in `internal/infra/transcriber` — primary Groq
  `whisper-large-v3-turbo`, failover to OpenAI `gpt-4o-mini-transcribe` (env
  `TRANSCRIBE_PRIMARY`). Never call providers outside this port.

## Development

### Prerequisites

- Go 1.24+, Node.js 20+, Docker
- Copy `backend/.env.example` to `backend/.env` and fill in keys (never commit `.env`)

### Run locally

```bash
# Start database
docker compose up -d postgres

# Backend — applies migrations, seeds 20 lessons + placement bank, serves :8080
cd backend && go run ./cmd/api -migrate -seed

# Frontend — Vite dev server on :5173, proxying /api → :8080
cd frontend && npm install && npm run dev

# Production-like PWA build (service worker only activates on a real build)
cd frontend && npm run build && npm run preview
```

### Backend tests

```bash
cd backend

# Lint + vet + vuln scan
golangci-lint run && go vet ./... && govulncheck ./...

# All unit tests
go test ./...

# Single test or package
go test ./internal/domain/speech/... -run TestSimilarityScore
go test ./internal/usecase/placement/... -run TestTestletBand

# Integration tests (requires Docker; testcontainers spins up real Postgres)
go test -tags=integration ./...

# Coverage report (gate: ≥80% on domain and usecase layers)
go test ./... -coverprofile=cover.out
go tool cover -func=cover.out
```

### Frontend tests

```bash
cd frontend

# Lint + type-check
npm run lint && npx tsc --noEmit

# Unit / component tests (Vitest + React Testing Library)
npm run test

# Single test file
npx vitest run src/features/placement/PlacementFlow.test.tsx

# E2E (Playwright — starts API + built PWA)
npm run test:e2e
```

## Conventions & Gates

- API under `/api/v1`, contract-first against `contracts/openapi.yaml`; errors as RFC 9457
  problem+json. Cookie sessions (HttpOnly/Secure/SameSite=Lax) + CSRF double-submit.
- Security: OWASP ASVS L1 baseline (see research.md R6) — parameterized SQL only (sqlc),
  rate limits on auth/speech, security-header middleware, no PII/audio in logs.
- Tests: `go test` + testify + testcontainers (integration), coverage ≥80% on
  domain/usecase; Vitest + RTL, Playwright E2E (incl. offline + PWA); axe-core a11y.
- Lint: golangci-lint, go vet, govulncheck | ESLint, tsc --noEmit, npm audit.
- Scoring rules live in `internal/domain` as pure functions (WER-based similarity,
  Levenshtein typo tolerance — research R8). Server never trusts client scores.
- UI: pt-BR; content en-US. Dark theme, WCAG 2.1 AA (≥4.5:1), no pure black (#121212-class).
- All PKs are UUID v7 (server-generated; client-generated for offline-outbox rows, deduped on sync).
- `progress_logs` is INSERT-only (immutable activity log — no UPDATE/DELETE).
- Streak and heatmap day-bucketing uses `users.timezone` (IANA name); never compute in UTC.
