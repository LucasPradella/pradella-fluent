# Tasks: FluentDev — English Learning PWA for Tech Professionals

**Input**: Design documents from `/specs/001-fluentdev-pwa/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/openapi.yaml, quickstart.md

**Tests**: INCLUDED — test coverage is an explicit user constraint and constitution Principle II requirement (≥80% on backend domain/usecase; Vitest/Playwright on frontend).

**Organization**: Tasks are grouped by user story (US1–US5, priority P1–P5) so each story is independently implementable and testable.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US5)
- Include exact file paths in descriptions

## Path Conventions

Web app per plan.md: `backend/` (Go clean architecture) + `frontend/` (React PWA).

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and tooling for both stacks

- [X] T001 Create root structure per plan.md: `backend/`, `frontend/`, `docker-compose.yml` with PostgreSQL 16 service (db `fluentdev`, healthcheck)
- [X] T002 Initialize Go module in `backend/go.mod` (Go 1.24) with chi, pgx/v5, sqlc config `backend/sqlc.yaml`, golang-migrate, testify; create `backend/cmd/api/main.go` skeleton with `-migrate`/`-seed` flags
- [X] T003 [P] Initialize frontend in `frontend/package.json`: Vite + React 19 + TypeScript 5, vite-plugin-pwa, TanStack Query, Zustand, Dexie, Vitest + React Testing Library, Playwright; `frontend/vite.config.ts` with `/api` proxy to :8080
- [X] T004 [P] Configure backend linting: `backend/.golangci.yml` (govet, errcheck, gosec, staticcheck) and `backend/Makefile` targets `lint`, `test`, `test-integration`, `coverage` (fails under 80% for `internal/domain`, `internal/usecase`)
- [X] T005 [P] Configure frontend linting: `frontend/eslint.config.js`, `frontend/.prettierrc`, npm scripts `lint`, `test`, `test:e2e`, `typecheck` in `frontend/package.json`
- [X] T006 [P] Create CI pipeline in `.github/workflows/ci.yml`: backend (golangci-lint, go vet, govulncheck, unit + integration tests, coverage gate) and frontend (eslint, tsc --noEmit, vitest, npm audit, Playwright smoke) jobs
- [X] T007 [P] Create `backend/.env.example` and config loader in `backend/internal/infra/config/config.go` (env-based: DATABASE_URL, session, TRANSCRIBE_PRIMARY, provider keys, OAuth creds, APP_BASE_URL per quickstart.md)

**Checkpoint**: `docker compose up -d postgres` works; `go build ./...` and `npm run dev` start clean

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Database schema, HTTP/middleware backbone, app shell — required by every story

- [X] T008 Write SQL migrations in `backend/migrations/0001_init.up.sql`/`.down.sql` for all data-model.md tables: users, auth_identities, sessions, modules, lessons, exercises, placement_questions, placement_sessions, placement_answers, progress_logs (INSERT-only grants), review_queue_items — with enums, FKs, unique/partial indexes
- [X] T009 Define sqlc queries in `backend/internal/adapter/postgres/queries/*.sql` (users, sessions, content, placement, progress, review) and generate type-safe code via `sqlc generate` into `backend/internal/adapter/postgres/gen/`
- [X] T010 [P] Create domain packages with entities and shared errors (`ErrNotFound`, `ErrForbidden`, `ErrConflict`) in `backend/internal/domain/{user,content,placement,progress,review,speech}/` (types only; behavior arrives per story)
- [X] T011 Build HTTP backbone in `backend/internal/adapter/http/`: chi router (`router.go`), middleware chain (`middleware/request_id.go`, `middleware/logging.go` slog JSON, `middleware/security_headers.go` CSP/HSTS/nosniff/Permissions-Policy, `middleware/rate_limit.go` per-IP+per-user token bucket, `middleware/recover.go`), RFC 9457 error writer (`problem.go`) — per research R6
- [X] T012 Implement session management: `backend/internal/usecase/auth/session.go` (opaque 256-bit token, SHA-256 hash storage, sliding expiry) + auth middleware `backend/internal/adapter/http/middleware/auth.go` (cookie `fluentdev_session`, HttpOnly/Secure/SameSite=Lax) + CSRF double-submit middleware `middleware/csrf.go`
- [X] T013 Wire composition root in `backend/cmd/api/main.go`: config → pgx pool → repositories → usecases → router → graceful shutdown; migrations runner and seed loader hook (`backend/seed/loader.go`)
- [X] T014 [P] Integration-test harness in `backend/internal/adapter/postgres/testutil/db.go` using testcontainers-go (spin Postgres, run migrations) + first smoke test `backend/internal/adapter/postgres/users_test.go`
- [X] T015 [P] Create frontend app shell in `frontend/src/app/`: router (`routes.tsx`), providers (`providers.tsx` TanStack Query + persistence), layout shell (`shell.tsx`), dark-theme design tokens `frontend/src/shared/ui/tokens.css` (#121212-class backgrounds, #E0E0E0 text, desaturated accents — FR-024)
- [X] T016 [P] Generate typed API client from `specs/001-fluentdev-pwa/contracts/openapi.yaml` into `frontend/src/shared/api/` (openapi-typescript + thin fetch wrapper `client.ts` sending X-CSRF-Token, mapping problem+json errors)
- [X] T017 [P] Configure base PWA in `frontend/vite.config.ts` + `frontend/src/pwa/`: manifest (name, pt-BR lang, standalone display, 192/512 icons in `frontend/public/icons/`), Workbox precache of built shell (Cache-First per research R7)
- [X] T018 [P] Create Dexie database schema in `frontend/src/shared/offline/db.ts` (cachedProfile, cachedTracks, cachedLessons, outbox stores per data-model.md client-side section)

**Checkpoint**: API serves 404 problem+json with security headers; frontend shell renders in dark theme; Lighthouse recognizes installable manifest — no user story implemented yet

---

## Phase 3: User Story 1 — Onboarding and Adaptive Placement Test (Priority: P1) 🎯 MVP

**Goal**: Sign up (GitHub/Google/e-mail) → immediate adaptive placement test → assigned level + unlocked track (FR-001…FR-006)

**Independent Test**: quickstart.md V1/V2 — register, complete placement in ≤12 questions with band adaptation, see level and locked tracks; resume after abandoning

### Tests for User Story 1

- [X] T019 [P] [US1] Unit tests for placement band-walk in `backend/internal/domain/placement/session_test.go`: >70% up, <40% down, else stay; B1 start; 12-question hard stop; final level from last two testlets; no question repeats (research R10)
- [X] T020 [P] [US1] Unit tests for password hashing and session token lifecycle in `backend/internal/usecase/auth/auth_test.go` (argon2id verify, constant-time, expiry)
- [X] T021 [P] [US1] Contract tests in `backend/internal/adapter/http/auth_handlers_test.go` + `placement_handlers_test.go` (httptest): register/login/logout/me and placement session/answers against openapi.yaml shapes, incl. 401/409/429 paths

### Implementation for User Story 1

- [X] T022 [P] [US1] Implement placement domain logic in `backend/internal/domain/placement/session.go`: band transitions, testlet scoring, completion rule, level mapping (pure functions)
- [X] T023 [P] [US1] Implement auth usecases in `backend/internal/usecase/auth/`: `register.go` (argon2id, dup e-mail → 409), `login.go` (rate-limit hooks, generic errors), `logout.go`, `me.go`
- [X] T024 [US1] Implement OAuth in `backend/internal/infra/oauth/{github,google}.go` + `backend/internal/usecase/auth/oauth.go`: authorization-code + state + PKCE, identity linking by verified e-mail (data-model auth_identities rule)
- [X] T025 [US1] Implement placement usecase in `backend/internal/usecase/placement/service.go`: start/resume session, serve next question (never leaking correct answer), score answer server-side, persist after every answer, atomic level assignment to users
- [X] T026 [US1] Implement postgres repositories in `backend/internal/adapter/postgres/{users,sessions,placement}.go` over sqlc gen; integration tests in `placement_repo_test.go` (active-session partial unique, no-repeat constraint)
- [X] T027 [US1] Register HTTP handlers in `backend/internal/adapter/http/{auth_handlers,placement_handlers}.go` and routes in `router.go`: POST /auth/register, /auth/login, /auth/logout, GET /me, OAuth start/callback, GET+POST /placement/session, POST /placement/session/answers — with input validation (length/enum/uuid)
- [X] T028 [US1] Author placement question bank seed (≥15 calibrated questions per band A1–C1, choice/listening_choice/order types) in `backend/seed/placement_questions.json` + loader wiring
- [X] T029 [P] [US1] Build onboarding UI in `frontend/src/features/onboarding/`: sign-in/register screens (pt-BR copy, password ≥10 chars), OAuth buttons + callback route, session-aware redirect (new user → placement)
- [X] T030 [US1] Build placement UI in `frontend/src/features/placement/`: question renderer per type, progress indicator (n/12), resume-on-return, result screen showing level + locked/unlocked tracks (FR-006)
- [X] T031 [P] [US1] Component tests in `frontend/src/features/placement/placement.test.tsx` (question type rendering, result states) and Playwright journey `frontend/tests/e2e/us1-placement.spec.ts` covering quickstart V1 + V2

**Checkpoint**: Full US1 journey works end-to-end — this alone is a deployable MVP (diagnosis product)

---

## Phase 4: User Story 2 — Task-Based Lessons (Writing and Listening) (Priority: P2)

**Goal**: Placed learner completes task-framed lessons with typo-tolerant writing and listening exercises; XP + immutable progress logging (FR-007…FR-012)

**Independent Test**: quickstart.md V3 — complete a lesson with a small typo accepted, wrong answer rejected with expected answer shown, XP awarded

### Tests for User Story 2

- [X] T032 [P] [US2] Unit tests for writing validator in `backend/internal/domain/speech/typo_test.go`: per-word Levenshtein tolerance (≤1 for ≤5 chars, ≤2 longer), semantic miss rejection, normalization (research R8)
- [X] T033 [P] [US2] Unit tests for XP award + lesson completion detection in `backend/internal/usecase/lessons/attempts_test.go` incl. idempotent replay by attemptId
- [X] T034 [P] [US2] Contract tests in `backend/internal/adapter/http/content_handlers_test.go`: GET /tracks lock state (403 on locked lesson — FR-006), GET /lessons/{id} excludes correct answers, POST attempts 201/200-duplicate

### Implementation for User Story 2

- [X] T035 [P] [US2] Implement writing validator in `backend/internal/domain/speech/typo.go` (pure: normalize, per-word Levenshtein, tolerated-typo list output)
- [X] T036 [US2] Implement content + attempts usecases in `backend/internal/usecase/lessons/{tracks.go,lesson.go,attempts.go}`: lock enforcement vs proficiency_level, server-side scoring, immutable progress_logs insert + streak recompute in one transaction (data-model users rule), listening answer checking (choice + word order)
- [X] T037 [US2] Implement postgres repositories in `backend/internal/adapter/postgres/{content,progress}.go` + integration tests `progress_repo_test.go` (INSERT-only enforcement, PK-idempotent replay)
- [X] T038 [US2] Register HTTP handlers/routes in `backend/internal/adapter/http/content_handlers.go`: GET /tracks, GET /lessons/{lessonId}, POST /exercises/{exerciseId}/attempts per openapi.yaml
- [X] T039 [US2] Author MVP content seed in `backend/seed/lessons.json`: ≥20 lessons across travel/tech themes and levels with task framing (FR-008, FR-012), incl. listening audio asset URLs under `frontend/public/audio/`
- [X] T040 [P] [US2] Build track/module UI in `frontend/src/features/lessons/{tracks.tsx,module-list.tsx}` (theme + level grouping, lock badges)
- [X] T041 [US2] Build lesson player in `frontend/src/features/lessons/{lesson-player.tsx,writing-exercise.tsx,listening-choice.tsx,listening-order.tsx,completion.tsx}`: typo highlight, expected-answer feedback, replayable audio, word-block ordering, XP celebration
- [X] T042 [P] [US2] Component tests in `frontend/src/features/lessons/lesson-player.test.tsx` + Playwright journey `frontend/tests/e2e/us2-lessons.spec.ts` covering quickstart V3

**Checkpoint**: US1 + US2 = usable learning product without speech

---

## Phase 5: User Story 3 — Speaking Practice with Automatic Feedback (Priority: P3)

**Goal**: Record ≤30 s speech → Groq-first transcription with OpenAI failover → 1−WER similarity, ≥80% pass, missed-word highlights (FR-013…FR-017)

**Independent Test**: quickstart.md V4/V5/V6 — scored feedback ≤3.5 s on throttled 4G, permission-denied skip path, provider failover

### Tests for User Story 3

- [X] T043 [P] [US3] Unit tests for similarity scoring in `backend/internal/domain/speech/wer_test.go`: normalization (case/punct/contractions), 1−WER math, alignment-derived missed words, 0.80 threshold (research R8)
- [X] T044 [P] [US3] Adapter tests in `backend/internal/infra/transcriber/transcriber_test.go` with fake HTTP servers: Groq success, Groq 429/timeout → OpenAI failover, both-down → typed `ErrProvidersUnavailable`, 2.5 s soft deadline (research R2)
- [X] T045 [P] [US3] Contract tests in `backend/internal/adapter/http/speech_handlers_test.go`: multipart validation (size 413, MIME sniff, duration), 201 result shape, 422 unintelligible (not scored as failure), 429 rate limit, 503 providers down

### Implementation for User Story 3

- [X] T046 [P] [US3] Implement scoring domain in `backend/internal/domain/speech/wer.go` (pure: normalize, word-Levenshtein alignment, similarity + missedWords)
- [X] T047 [US3] Implement `Transcriber` port + adapters in `backend/internal/infra/transcriber/{transcriber.go,groq.go,openai.go,failover.go}`: interface per research R2, env-selected primary, single failover retry, per-provider latency/error slog metrics, outbound HTTP restricted to configured hosts (OWASP A10)
- [X] T048 [US3] Implement speech usecase in `backend/internal/usecase/speech/attempt.go`: validate audio (≤1.5 MB, ≤30 s, webm/mp4 MIME sniffing), transcribe, score, log progress with provider detail, feed review queue on fail, idempotent by attemptId
- [X] T049 [US3] Register handler/route POST /exercises/{exerciseId}/speech-attempts in `backend/internal/adapter/http/speech_handlers.go` with per-user rate limit (cost-bearing endpoint — research R6 A04)
- [X] T050 [P] [US3] Build recorder in `frontend/src/features/speaking/{recorder.ts,use-recorder.ts}`: MediaRecorder with `isTypeSupported` fallback webm→mp4 (Safari), 30 s auto-stop, permission request UX with explanation, denied → skip path that never blocks lesson completion (FR-016)
- [X] T051 [US3] Build speaking exercise UI in `frontend/src/features/speaking/{speaking-exercise.tsx,result.tsx}`: target sentence display, record button states, similarity %, missed words in desaturated red (FR-015), retry, offline/provider-down messaging
- [X] T052 [P] [US3] Recorder state-machine tests in `frontend/src/features/speaking/recorder.test.ts` + Playwright journey `frontend/tests/e2e/us3-speaking.spec.ts` (fake media stream, permission-denied, mocked provider failover) covering quickstart V4–V6

**Checkpoint**: Core differentiator live — speaking evaluated within latency budget

---

## Phase 6: User Story 4 — Streak, Activity Heatmap, and Spaced Review (Priority: P4)

**Goal**: Dashboard with streak + 90-day heatmap; failed items auto-queued for spaced review counting toward activity (FR-018…FR-020)

**Independent Test**: quickstart.md V7 — multi-day simulated activity produces correct streak/heatmap; failed item resurfaces and logs `is_review=true`

### Tests for User Story 4

- [X] T053 [P] [US4] Unit tests for streak rules in `backend/internal/domain/progress/streak_test.go`: consecutive-day increment, reset after gap, longest preserved, user-timezone day bucketing incl. 23:59/00:01 edge case
- [X] T054 [P] [US4] Unit tests for spacing intervals in `backend/internal/domain/review/spacing_test.go`: 1d→3d→7d→21d on pass, reset to 1d on fail, exit after two ≥7d passes, repeated-failure shortening
- [X] T055 [P] [US4] Contract tests in `backend/internal/adapter/http/dashboard_handlers_test.go`: GET /dashboard heatmap buckets (90 items max, saturation level 0–4), GET /review-queue due-only ordering

### Implementation for User Story 4

- [X] T056 [P] [US4] Implement streak + heatmap domain in `backend/internal/domain/progress/{streak.go,heatmap.go}` (pure functions over progress logs, IANA timezone aware)
- [X] T057 [P] [US4] Implement spacing domain in `backend/internal/domain/review/spacing.go`
- [X] T058 [US4] Implement dashboard + review usecases in `backend/internal/usecase/dashboard/{dashboard.go,review.go}`; wire failed writing/speaking attempts into review_queue_items inside the attempts usecases from T036/T048
- [X] T059 [US4] Implement postgres repository in `backend/internal/adapter/postgres/review.go` + 90-day aggregation query in `queries/progress.sql`; integration test `review_repo_test.go` (UNIQUE user+exercise upsert)
- [X] T060 [US4] Register handlers/routes GET /dashboard, GET /review-queue in `backend/internal/adapter/http/dashboard_handlers.go`
- [X] T061 [P] [US4] Build dashboard UI in `frontend/src/features/dashboard/{dashboard.tsx,streak.tsx,heatmap.tsx}`: streak counter, 90-day CSS-grid heatmap with 5 saturation buckets (accessible labels), due-reviews entry point
- [X] T062 [US4] Build review session flow in `frontend/src/features/review/review-session.tsx`: quick-exercise runner reusing lesson exercise components with `isReview` flag
- [X] T063 [P] [US4] Heatmap component tests in `frontend/src/features/dashboard/heatmap.test.tsx` + Playwright journey `frontend/tests/e2e/us4-retention.spec.ts` (clock-mocked multi-day) covering quickstart V7

**Checkpoint**: Retention loop complete

---

## Phase 7: User Story 5 — Installable, Offline-Resilient App Experience (Priority: P5)

**Goal**: Home-screen standalone install; offline shell <1.5 s with cached data; offline writes replay via outbox; eviction-proof cloud sync (FR-021…FR-023)

**Independent Test**: quickstart.md V8/V9/V10 — install on Android/iOS, offline reload under budget, offline completion syncs idempotently

### Tests for User Story 5

- [X] T064 [P] [US5] Unit tests for outbox replay in `frontend/src/shared/offline/outbox.test.ts`: FIFO replay on reconnect, duplicate-replay idempotence (same attemptId), clamped timestamps
- [X] T065 [P] [US5] Playwright offline suite `frontend/tests/e2e/us5-offline.spec.ts`: context.setOffline shell render <1.5 s with cached profile, network-required messaging on speaking/auth, offline attempt → reconnect → single progress log (quickstart V9/V10)

### Implementation for User Story 5

- [X] T066 [US5] Complete Workbox runtime strategies in `frontend/src/pwa/sw-strategies.ts` per research R7 cache matrix: SWR for /me, /tracks, /dashboard; Cache-First for audio/images/fonts; Network-Only for speech/auth with problem+json offline fallback
- [X] T067 [US5] Implement offline sync in `frontend/src/shared/offline/{outbox.ts,sync.ts,connectivity.ts}`: enqueue attempts when offline, replay on `online`/app-open, TanStack Query persistence to Dexie, cache mirrors refresh
- [X] T068 [P] [US5] Offline-aware UX in `frontend/src/shared/ui/offline-banner.tsx` + gating in speaking/auth features: clear pt-BR messaging for connectivity-required features (US5 scenario 3)
- [X] T069 [P] [US5] Finalize installability: real 192/512 maskable icons in `frontend/public/icons/`, iOS meta tags, install prompt hint component `frontend/src/pwa/install-hint.tsx` (Safari manual instructions); verify Lighthouse PWA pass (quickstart V8)

**Checkpoint**: All five user stories complete

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: NFR verification, security hardening, performance budgets, docs

- [X] T070 [P] Accessibility audit and fixes: axe-core checks wired into Playwright (`frontend/tests/e2e/a11y.spec.ts`) for dashboard, lesson, placement screens in dark theme — 0 contrast violations (quickstart V11, FR-024)
- [X] T071 [P] Security verification: ZAP baseline scan script `scripts/zap-baseline.sh` against local API; cross-user object access tests in `backend/internal/adapter/http/authz_test.go` (OWASP A01); confirm rate-limit 429s on auth/speech burst (quickstart V12)
- [X] T072 [P] Performance budget verification: Lighthouse CI config `frontend/lighthouserc.json` (PWA pass, perf ≥90 mobile, shell <1.5 s), Playwright 4G-throttled speech-loop timing assertion (SC-004), backend p95 <200 ms check via `backend/internal/adapter/http/bench_test.go`
- [X] T073 [P] Structured-logging & observability pass: request-ID propagation to usecases, provider latency metrics, auth/rate-limit event logs, assert no PII/audio in logs (`backend/internal/adapter/http/logging_test.go`)
- [X] T074 Enforce coverage gates in CI (fail <80% domain/usecase) and run `govulncheck` + `npm audit` clean; fix findings
- [X] T075 [P] Write `README.md` (project overview, quickstart link) and validate every command in `specs/001-fluentdev-pwa/quickstart.md` on a clean checkout; run full V1–V12 validation matrix and record results in `specs/001-fluentdev-pwa/quickstart.md` notes

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Setup — **BLOCKS all user stories**
- **US1 (Phase 3)**: Depends on Foundational only
- **US2 (Phase 4)**: Depends on Foundational; needs auth/session from US1 (T023/T027) to exercise lock rules — implementable in parallel after T023 lands
- **US3 (Phase 5)**: Depends on Foundational + lesson player host from US2 (T041) for full integration; backend speech stack (T043–T049) independent of US2
- **US4 (Phase 6)**: Depends on progress logging from US2 (T036/T037); review-queue feeding touches T036/T048
- **US5 (Phase 7)**: Depends on Foundational PWA base (T017/T018); outbox replay needs attempts endpoint from US2
- **Polish (Phase 8)**: Depends on all desired stories being complete

### Story completion order

`Setup → Foundational → US1 (MVP) → US2 → US3 → US4 → US5 → Polish`

### Parallel opportunities (examples)

- **Phase 1**: T003, T004, T005, T006, T007 in parallel after T001/T002
- **Phase 2**: T010, T014–T018 in parallel once T008/T009 land (backend dev + frontend dev split cleanly)
- **US1**: T019/T020/T021 (tests) together; then T022/T023 in parallel; T029 parallel with all backend tasks
- **US3 backend vs frontend**: T046–T049 (Go) fully parallel with T050 (recorder)
- **Cross-story**: after US2's T036, one developer can run Phase 6 (US4) while another does Phase 5 (US3) — they touch disjoint files except the noted T058 hook
- **Phase 8**: T070–T073, T075 all parallel

## Implementation Strategy

**MVP first**: Phases 1–3 only (T001–T031) ship a deployable product: sign-up + adaptive English diagnosis. Stop, validate quickstart V1/V2, demo.

**Incremental delivery**: each subsequent phase is an independently testable increment — US2 makes it a learning product, US3 adds the differentiator, US4 retention, US5 the installed offline experience. Run the story's Playwright journey at each checkpoint before moving on; keep CI gates green throughout (constitution Principle IV).
