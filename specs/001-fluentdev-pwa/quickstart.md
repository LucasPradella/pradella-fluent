# Quickstart & Validation Guide: FluentDev — English Learning PWA

**Feature**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md) |
**Contract**: [contracts/openapi.yaml](./contracts/openapi.yaml) |
**Data model**: [data-model.md](./data-model.md)

## Prerequisites

- Go 1.24+, Node.js 20+, Docker (for PostgreSQL via docker-compose)
- API keys: `GROQ_API_KEY` (primary transcriber), `OPENAI_API_KEY` (secondary — optional
  in dev; failover tests use fakes)
- OAuth apps (dev): GitHub and Google client ID/secret with callback
  `http://localhost:8080/api/v1/auth/oauth/{provider}/callback`

## Environment

`backend/.env.example` (copy to `.env`; never commit `.env`):

```env
DATABASE_URL=postgres://fluentdev:fluentdev@localhost:5432/fluentdev?sslmode=disable
SESSION_COOKIE_NAME=fluentdev_session
TRANSCRIBE_PRIMARY=groq            # groq | openai (research R2)
GROQ_API_KEY=...
OPENAI_API_KEY=...
GITHUB_CLIENT_ID=... / GITHUB_CLIENT_SECRET=...
GOOGLE_CLIENT_ID=... / GOOGLE_CLIENT_SECRET=...
APP_BASE_URL=http://localhost:5173
```

## Run locally

```bash
# 1. Database
docker compose up -d postgres

# 2. Backend (applies migrations, seeds 20 lessons + placement bank, serves :8080)
cd backend
go run ./cmd/api -migrate -seed

# 3. Frontend (Vite dev server on :5173, proxying /api → :8080)
cd frontend
npm install && npm run dev

# Production-like PWA check (service worker only activates on a real build)
npm run build && npm run preview
```

## Test commands (CI gates — plan Constitution Check IV)

```bash
# Backend
cd backend
golangci-lint run && go vet ./... && govulncheck ./...
go test ./... -coverprofile=cover.out            # unit + httptest contract tests
go test -tags=integration ./...                  # testcontainers: real Postgres
go tool cover -func=cover.out                    # gate: >=80% on internal/domain, internal/usecase

# Frontend
cd frontend
npm run lint && npx tsc --noEmit
npm run test                                     # Vitest + Testing Library
npm run test:e2e                                 # Playwright (starts API + built PWA)
```

## End-to-end validation scenarios (map to user stories)

| # | Story | Steps | Expected outcome |
|---|-------|-------|------------------|
| V1 | US1 P1 | Register with e-mail → answer placement testlets (force >70% correct on the first) | Test starts immediately; difficulty band rises; stops at ≤12 questions; level + unlocked track shown; locked tracks visible |
| V2 | US1 | Quit placement after 5 answers, log back in, `GET /placement/session` | Session resumes at question 6 |
| V3 | US2 P2 | Open unlocked track → complete a lesson with a deliberate small typo in one writing answer | Typo tolerated and highlighted; wrong semantic answer rejected with expected answer shown; lesson completion awards XP and logs progress |
| V4 | US3 P3 | In a speaking exercise: record the target sentence (normal read) | Feedback ≤3.5 s on throttled "Fast 4G" (Playwright/devtools); similarity ≥80% passes; omit a word → it renders highlighted in desaturated red |
| V5 | US3 | Deny microphone permission | Explanatory prompt + skip option; lesson still completable |
| V6 | US3 | Stop Groq fake server mid-test | Attempt transparently served by secondary provider; `detail.provider` in progress log = openai |
| V7 | US4 P4 | Complete activity on two simulated consecutive days (adjust clock/fixtures) | Streak = 2; heatmap squares saturate with volume; failed item from day 1 appears in review queue on day 2 and, when completed, logs `is_review=true` |
| V8 | US5 P5 | `npm run build && npm run preview`, install PWA on Android + iOS device/emulator | Standalone full-screen launch, correct 192/512 icons (Lighthouse PWA pass) |
| V9 | US5 | Load app once online, then reload with network disabled | App shell renders <1.5 s with cached profile/progress; speaking exercise shows "requires connection" message |
| V10 | US5 | Complete a written exercise offline, reconnect | Outbox replays; progress visible via `GET /dashboard` from a second browser (idempotent — replaying twice creates one log) |
| V11 | NFR | Run axe-core/Lighthouse a11y audit on dashboard, lesson, and placement screens in dark theme | 0 contrast violations (≥4.5:1, FR-024) |
| V12 | Security | `zap-baseline` (or equivalent) scan against the running API; attempt cross-user access (`GET /lessons/{id}` attempt logs of another user) | Security headers present; cross-user object access denied (OWASP A01); auth endpoints rate limited (429 after burst) |

## Definition of ready-to-ship (feature level)

All V1–V12 pass; CI gates green (lint, vuln scan, coverage ≥80% domain/usecase,
Playwright suite); success criteria SC-001…SC-010 verified per spec.

## Validation notes — 2026-07-05 (implementation run)

| # | Status | Evidence |
|---|--------|----------|
| V1 | ✅ automated | `frontend/tests/e2e/us1-placement.spec.ts` (register → 12-question walk → level + lock overview) — passing |
| V2 | ✅ automated | same spec: abandon after 5, re-login, resumes at question 6 — passing |
| V3 | ✅ automated | `us2-lessons.spec.ts`: typo "chek" tolerated + highlighted; wrong answer shows expected; completion screen — passing |
| V4 | ✅ automated (UI) | `us3-speaking.spec.ts` V4: similarity %, missed word in `.missed-word` (desaturated red). Latency budget: provider call mocked in E2E; backend soft deadline (2.5 s) covered by `transcriber_test.go`; end-to-end 4G timing requires real provider keys — verify manually before launch |
| V5 | ✅ automated | mic-denied path renders explanation + skip; lesson still completable |
| V6 | ✅ automated (backend) | `transcriber_test.go`: Groq 429/timeout → OpenAI failover; both-down → 503. Progress-log `detail.provider` persisted by `speechuc` |
| V7 | ✅ automated | streak/day-bucketing unit tests (`streak_test.go`, incl. 23:59/00:01 São Paulo edge); `us4-retention.spec.ts`: heatmap saturation, review round-trip sends `isReview=true` |
| V8 | ⚠️ manual pending | Manifest + 192/512 maskable icons + iOS meta tags shipped; `vite build` generates SW (9 precache entries). Install on physical Android/iOS devices still to be done by a human |
| V9 | ✅ automated | `us5-offline.spec.ts`: offline reload renders shell + cached data (<3 s in CI incl. navigation overhead); speaking shows connection-required message |
| V10 | ✅ automated | offline attempt → outbox → reconnect → exactly 1 interaction in dashboard (server dedupes by attemptId) |
| V11 | ✅ automated | `a11y.spec.ts` (axe-core, WCAG 2.1 A/AA): 0 contrast violations on login, placement, dashboard, lesson |
| V12 | ✅ automated + script | cross-user access denied (`authz_test.go`), 429 on auth/speech bursts, security headers asserted (`auth_handlers_test.go`); `scripts/zap-baseline.sh` ready to run against a live instance |

Gates at time of writing: backend `go vet` ✅, `go test ./...` ✅ (8 pkgs),
`go test -tags=integration` ✅ (testcontainers), coverage domain+usecase
**83.2%** (≥80 gate), `govulncheck` 0 called vulnerabilities; frontend ESLint ✅,
`tsc --noEmit` ✅, Vitest 24/24 ✅, Playwright 13/13 ✅, `npm audit` 0
vulnerabilities. Local Postgres note: use `POSTGRES_PORT=5433 docker compose up -d postgres`
when 5432 is occupied, and export the matching `DATABASE_URL`.
