# Implementation Plan: FluentDev — English Learning PWA for Tech Professionals

**Branch**: `001-fluentdev-pwa` | **Date**: 2026-07-04 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/001-fluentdev-pwa/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Build the FluentDev MVP: an installable, offline-resilient PWA that teaches communicative
and technical English to Brazilian Portuguese speakers. Core flows: OAuth/e-mail onboarding
into an adaptive placement test (CAMST testlets), task-based lessons (writing/listening),
speech evaluation via audio transcription with similarity scoring, and retention mechanics
(streak, 90-day heatmap, spaced review). Technical approach mandated by the user: **Go
backend with clean architecture**, browser-first frontend with full PWA support, OWASP-
aligned security, strong test coverage, and transcription via **Groq-hosted Whisper**
(cheapest per minute) behind a provider abstraction with **OpenAI as second provider**.

## Technical Context

**Language/Version**: Go 1.24 (backend API); TypeScript 5.x (frontend)

**Primary Dependencies**:
- Backend: `chi` (HTTP router), `pgx` + `sqlc` (PostgreSQL access), `golang.org/x/oauth2`
  (GitHub/Google SSO), `argon2id` password hashing, `slog` structured logging
- Frontend: React 19 + Vite, `vite-plugin-pwa` (Workbox service worker), TanStack Query
  (server state + stale-while-revalidate), Dexie (IndexedDB), Zustand (UI state)
- Speech: Groq API (`whisper-large-v3-turbo`) primary; OpenAI (`gpt-4o-mini-transcribe`)
  fallback — both behind a `Transcriber` port (see research.md R2)

**Storage**: PostgreSQL 16 (source of truth); IndexedDB on-device (offline cache of
progress, lessons, review queue — cache only, never source of truth per FR-023)

**Testing**: Backend — `go test` + `testify` + `testcontainers-go` (real Postgres in
integration tests), `httptest` for handlers, ≥80% coverage on domain/usecase layers.
Frontend — Vitest + React Testing Library (unit/component), Playwright (E2E incl.
offline-mode and PWA install checks). Contract tests validate handlers against
`contracts/openapi.yaml`.

**Target Platform**: Modern evergreen browsers; mobile Safari iOS 16.4+ and Android
Chrome for the installed PWA experience; backend as a Linux container

**Project Type**: Web application (frontend PWA + backend API)

**Performance Goals**: Speech loop (record → transcribe → render) p95 ≤ 3.5 s on 4G
(SC-004); app shell load < 1.5 s offline (SC-005); non-speech API endpoints p95 < 200 ms;
Lighthouse PWA installability = pass, performance ≥ 90 on mid-range mobile

**Constraints**: OWASP Top 10 / ASVS L1 controls (see research.md R6); WCAG 2.1 AA
contrast ≥ 4.5:1 in dark theme (FR-024); offline-first with optimistic cloud sync
(Safari 50 MB cache / 7-day eviction — FR-023); audio uploads capped at 30 s;
pt-BR UI, en-US content

**Scale/Scope**: MVP, free tier only; low thousands of users; 5 user stories, 25 FRs,
7 entities, ~20 seed lessons + calibrated placement bank

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Gate | Status |
|-----------|------|--------|
| I. Spec-Driven Development | Plan derives from approved `spec.md`; no behavior invented outside it | ✅ PASS — all design elements trace to FR/SC identifiers |
| II. Test-First Quality | Spec/user input request test coverage → test strategy defined per layer; acceptance scenarios map to automated tests | ✅ PASS — coverage targets and test types fixed in Technical Context |
| III. Simplicity & YAGNI | Structure beyond a single project must be justified | ✅ PASS with justification — 2 projects (frontend/backend) inherent to a web app; clean-architecture layering is a user-mandated constraint (see Complexity Tracking) |
| IV. Code Review & Quality Gates | Lint/format/test gates defined for both stacks | ✅ PASS — `golangci-lint`, `go vet`, `govulncheck`; ESLint + Prettier + `npm audit`; CI gates in quickstart.md |
| V. Observability & Versioning | Structured logging; versioned public contracts | ✅ PASS — `slog` JSON logs with request IDs; API versioned under `/api/v1` in openapi.yaml |

**Post-Phase-1 re-check (2026-07-04)**: ✅ PASS — design artifacts introduce no new
projects or speculative abstractions; the only pattern beyond minimum (transcription
provider port with two adapters) is explicitly required by the user ("opções de um
segundo provedor") and by FR-017.

## Project Structure

### Documentation (this feature)

```text
specs/001-fluentdev-pwa/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/
│   └── openapi.yaml     # Phase 1 output (/speckit-plan command)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
backend/
├── cmd/
│   └── api/
│       └── main.go              # composition root: config, DI wiring, server start
├── internal/
│   ├── domain/                  # entities + business rules, zero external deps
│   │   ├── user/                # User, streak rules
│   │   ├── content/             # Module, Lesson, Exercise
│   │   ├── placement/           # PlacementSession, testlet band transitions
│   │   ├── progress/            # ProgressLogEntry, heatmap aggregation
│   │   ├── review/              # ReviewQueueItem, spacing intervals
│   │   └── speech/              # similarity scoring, word-diff highlighting
│   ├── usecase/                 # application services (one package per story area)
│   │   ├── auth/
│   │   ├── placement/
│   │   ├── lessons/
│   │   ├── speech/
│   │   └── dashboard/
│   ├── adapter/
│   │   ├── http/                # chi router, handlers, middleware (auth, rate limit,
│   │   │                        #   security headers, request logging)
│   │   └── postgres/            # sqlc-generated queries + repository implementations
│   └── infra/
│       ├── config/              # env-based config loading
│       ├── transcriber/         # Transcriber port + groq/ and openai/ adapters + failover
│       └── oauth/               # GitHub/Google provider setup
├── migrations/                  # SQL migrations (golang-migrate)
├── seed/                        # 20 MVP lessons + placement question bank (JSON/SQL)
└── go.mod

frontend/
├── src/
│   ├── app/                     # app shell, routing, providers, dark theme tokens
│   ├── features/
│   │   ├── onboarding/          # sign-in, OAuth callback
│   │   ├── placement/           # adaptive test flow
│   │   ├── lessons/             # module list, lesson player, writing/listening exercises
│   │   ├── speaking/            # recorder (MediaDevices), result rendering
│   │   ├── dashboard/           # streak, heatmap, review queue entry
│   │   └── review/              # spaced-review quick sessions
│   ├── shared/
│   │   ├── api/                 # typed client generated from openapi.yaml
│   │   ├── ui/                  # design-system components (WCAG AA dark theme)
│   │   └── offline/             # Dexie schemas, sync queue, connectivity signals
│   └── pwa/                     # manifest, service-worker strategies (Workbox)
├── public/                      # icons 192/512, static assets
├── tests/
│   └── e2e/                     # Playwright: per-user-story journeys + offline/PWA
└── package.json

docker-compose.yml               # local Postgres (+ optional API) for development
```

**Structure Decision**: Option 2 (web application). `backend/` follows clean architecture
with dependency direction infra/adapter → usecase → domain; `frontend/` is feature-foldered
with the PWA/offline concerns isolated in `shared/offline` and `pwa/`. Two projects only —
no separate workers, gateways, or shared libs in the MVP.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Clean-architecture layering (domain/usecase/adapter/infra) instead of flat handler→db code | Explicit user requirement ("arquitetura deve ser clean"); also isolates the transcription provider swap (FR-017) and keeps domain rules (testlet bands, streaks, spacing) unit-testable without infrastructure | Flat structure rejected: violates the stated constraint and would couple OWASP-sensitive I/O code with business rules, hurting the mandated test coverage |
| Second transcription provider adapter (OpenAI) behind the `Transcriber` port | Explicit user requirement ("opções de um segundo provedor") and resilience for the platform's core differentiator (US3) — Groq outage would otherwise disable speaking exercises entirely | Single provider rejected: user constraint; also single point of failure on the highest-value flow |
