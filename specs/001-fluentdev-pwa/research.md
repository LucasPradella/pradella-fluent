# Phase 0 Research: FluentDev — English Learning PWA

**Feature**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md) | **Date**: 2026-07-04

All Technical Context unknowns are resolved below. User-supplied constraints (Go backend,
clean architecture, PWA, OWASP security, test coverage, Groq-first transcription with a
second provider) are treated as fixed inputs, not open questions.

## R1 — Backend language & HTTP stack

- **Decision**: Go 1.24, `chi` router, standard `net/http`, `slog` for structured JSON
  logging, graceful shutdown, request-ID middleware.
- **Rationale**: Go mandated by user. `chi` is idiomatic (stdlib-compatible handlers,
  context-based middleware), tiny dependency surface, and battle-tested — a good fit for
  clean architecture where the router is an adapter detail.
- **Alternatives considered**: Echo/Gin (heavier abstractions, custom context types leak
  into handlers); Fiber (fasthttp — incompatible with `net/http` ecosystem and `httptest`).

## R2 — Speech transcription providers (user: "prefira GROG… segundo provedor")

- **Decision**: Interpret "GROG" as **Groq** (LPU inference cloud hosting Whisper models —
  matches the "cheaper" rationale; "Grok"/xAI offers no transcription API). Primary:
  **Groq `whisper-large-v3-turbo`** (~US$0.04 per audio-hour ≈ US$0.0007/min, ~5× cheaper
  than the PRD's original pick, with very low latency — helps the 3.5 s p95 budget).
  Secondary: **OpenAI `gpt-4o-mini-transcribe`** (US$0.003/min, the PRD's original choice).
  Both implemented as adapters of a domain-owned port:

  ```go
  type Transcriber interface {
      Transcribe(ctx context.Context, audio io.Reader, opts TranscribeOpts) (Transcript, error)
  }
  ```

  Failover policy: try primary with a 2.5 s soft deadline (fits RNF-01 budget); on error,
  timeout, or HTTP 429, retry once on the secondary. Provider order configurable by env
  (`TRANSCRIBE_PRIMARY=groq|openai`). Emit per-provider latency/error metrics in logs.
- **Rationale**: Meets the cost preference, satisfies the explicit second-provider
  requirement, and shields the domain from vendor lock (FR-017 also allows future
  on-device processing as just another adapter).
- **Alternatives considered**: OpenAI-only (PRD's original: rejected — user cost
  preference + single point of failure); Deepgram/AssemblyAI as secondary (fine products,
  but OpenAI keeps parity with the PRD and the widest Whisper compatibility); on-device
  WASM Whisper in MVP (rejected: 40 s CPU transcription on mobile per PRD research —
  deferred to Phase 2 roadmap as a third adapter).

## R3 — Frontend framework & PWA toolchain

- **Decision**: React 19 + TypeScript + Vite; `vite-plugin-pwa` (Workbox under the hood)
  for manifest + service worker; TanStack Query for server state (native
  stale-while-revalidate semantics matching the PRD cache table); Zustand for small UI
  state; Dexie for IndexedDB.
- **Rationale**: PRD names React/Vue; React has the deepest ecosystem for the pieces this
  product needs (audio recording hooks, virtualized heatmaps, a11y tooling). Vite gives
  the performance budget headroom (code-splitting per feature folder, precache manifest
  generation). TanStack Query maps 1:1 to the *Stale-While-Revalidate* strategy required
  for profile/progress data.
- **Alternatives considered**: Vue/Nuxt (equally viable; React chosen for team-market
  availability in BR and richer PWA examples); Next.js (SSR unneeded for an installed
  app-shell PWA and complicates the offline story); SvelteKit (smaller bundles but
  thinner library ecosystem for heatmap/recorder needs).

## R4 — Database & data access

- **Decision**: PostgreSQL 16. Access via `pgx/v5` + `sqlc` (compile-time-checked,
  generated, parameterized queries). Migrations with `golang-migrate`, plain SQL files.
- **Rationale**: PRD prescribes relational modeling; Postgres is the market default and
  the entities (immutable progress log, review queue with due dates) are naturally
  relational. `sqlc` guarantees parameterized SQL (OWASP A03 injection) and keeps the
  repository layer thin — aligned with clean architecture.
- **Alternatives considered**: GORM (runtime reflection, weaker injection guarantees,
  hides SQL — worse for the performance budget); SQLite (no managed-hosting story for
  multi-device sync); Supabase client-direct access (bypasses the Go domain layer where
  streak/spacing rules must live).

## R5 — Authentication & session model

- **Decision**: OAuth 2.0 / OIDC with GitHub and Google via `golang.org/x/oauth2`
  (authorization-code flow with `state` + PKCE), plus e-mail/password with **argon2id**
  hashing. Sessions: opaque 256-bit token in an `HttpOnly; Secure; SameSite=Lax` cookie,
  server-side session record in Postgres with sliding expiry. CSRF: SameSite plus
  double-submit token on state-changing requests.
- **Rationale**: FR-001 requires exactly these providers. Cookie sessions beat
  JWT-in-localStorage on OWASP grounds (no XSS-readable credentials, instant revocation).
  Argon2id is the current OWASP password-storage recommendation.
- **Alternatives considered**: JWT access/refresh tokens (revocation complexity, XSS
  exposure if stored in JS-readable storage — unnecessary for a single first-party API);
  hosted auth (Auth0/Clerk — recurring cost against a free-tier MVP and an external
  dependency the spec doesn't need).

## R6 — OWASP-aligned security controls (user: "se proteja de qualquer ataque baseado no OWASP")

- **Decision**: Adopt OWASP ASVS Level 1 as the MVP bar, mapped to Top 10 (2021):
  - **A01 Broken Access Control**: every handler resolves the acting user from the
    session server-side; object-level checks in usecases (a user can only read/write
    their own progress, sessions, queue). Deny-by-default router.
  - **A02 Cryptographic Failures**: TLS-only (HSTS), argon2id passwords, secrets from
    env/secret manager, no PII in logs.
  - **A03 Injection**: sqlc parameterized queries only; strict input validation
    (length/enum/UUID) at the HTTP adapter; audio uploads validated by size (≤ ~1 MB /
    30 s), MIME sniffing, and duration before leaving the adapter.
  - **A04 Insecure Design**: rate limiting per IP + per user on auth and speech endpoints
    (speech is the cost-bearing endpoint — abuse = direct spend); placement/scoring rules
    enforced server-side, never trusted from the client.
  - **A05 Security Misconfiguration**: security-header middleware — CSP (self + API
    origin, no inline script), `X-Content-Type-Options`, `Referrer-Policy`,
    `Permissions-Policy` (microphone only where needed); minimal container image
    (distroless), non-root.
  - **A06 Vulnerable Components**: `govulncheck` + `npm audit` + Dependabot/Renovate in CI;
    lockfiles committed.
  - **A07 Auth Failures**: rate-limited login, constant-time compares, generic error
    messages, session invalidation on logout/password change.
  - **A08 Integrity Failures**: service worker only precaches build-hashed assets; CI
    builds from lockfiles.
  - **A09 Logging Failures**: structured `slog` logs with request ID, user ID (not
    e-mail), auth and rate-limit events; no audio content or credentials logged.
  - **A10 SSRF**: outbound HTTP restricted to the two configured transcription hosts.
- **Rationale**: Direct user mandate; ASVS L1 is the standard verifiable baseline for an
  MVP of this risk profile.
- **Alternatives considered**: ASVS L2 (adds threat-modeling/2FA burdens not justified
  for a free-tier learning MVP; revisit for B2B phase).

## R7 — Offline & caching architecture (PWA)

- **Decision**: Implement exactly the PRD cache matrix with Workbox strategies:
  app shell (HTML/CSS/JS) → *Cache-First* with build-versioned precache; images/icons/
  fonts → *Cache-First*; profile/progress reads → *Stale-While-Revalidate* (TanStack
  Query persisted to IndexedDB via Dexie); speech upload + auth → *Network-Only* with
  clear offline messaging. Writes made offline (lesson completions, reviews) enter a
  Dexie-backed outbox replayed on reconnect (optimistic sync, server = source of truth,
  last-write-wins keyed by client-generated UUID + timestamp per FR-023 / edge cases).
- **Rationale**: Matches PRD architecture guidance and survives Safari's 50 MB /
  7-day eviction because Postgres holds the truth and the outbox is short-lived.
- **Alternatives considered**: Background Sync API (unsupported on iOS Safari — outbox
  replay on app-open/`online` event chosen instead); full offline lesson packs in MVP
  (deferred: only the current track's lesson metadata + text assets are cached; audio
  packs are a Phase 2 item per PRD).

## R8 — Similarity scoring & typo tolerance

- **Decision**: Speaking (FR-014/15): normalize transcript and target (lowercase, strip
  punctuation, collapse whitespace, expand common contractions), then score
  `similarity = 1 − WER` (word-level Levenshtein / word error rate); pass at ≥ 0.80;
  the alignment's deletions/substitutions give the highlighted words. Writing (FR-009):
  per-word character-level Levenshtein — distance ≤ 1 for words ≤ 5 chars, ≤ 2 for longer
  words counts as a tolerated typo; any non-tolerated word difference fails the answer.
  All scoring lives in `internal/domain/speech` as pure functions.
- **Rationale**: WER is the standard ASR-evaluation metric, cheap to compute in Go, fully
  deterministic (testable), and produces word alignment for the red-highlight UX for free.
- **Alternatives considered**: LLM-judged semantic similarity (non-deterministic, adds
  cost + latency to a 3.5 s budget); phoneme-level scoring (better pedagogy, but requires
  forced alignment tooling — Phase 2+ candidate); embedding cosine similarity (opaque
  thresholds, hard to explain word-level feedback).

## R9 — Testing strategy & coverage gates

- **Decision**:
  - Backend: unit tests for `domain` + `usecase` (≥ 80% line coverage, enforced in CI);
    integration tests for `adapter/postgres` with `testcontainers-go` (real Postgres);
    `httptest`-based handler tests validating requests/responses against
    `contracts/openapi.yaml` (contract tests); transcription adapters tested against
    recorded fixtures with a fake HTTP server.
  - Frontend: Vitest + React Testing Library for exercise validators, heatmap rendering,
    recorder state machine; Playwright E2E for each user story's primary journey,
    including an offline scenario (context.setOffline) and PWA manifest/installability
    assertions; axe-core a11y checks on key screens (contrast — FR-024).
  - CI gates: `golangci-lint`, `go vet`, `govulncheck`, backend coverage ≥ 80%
    (domain/usecase), ESLint, `tsc --noEmit`, Vitest, Playwright smoke suite.
- **Rationale**: User mandated coverage; constitution Principle II requires tests mapped
  to acceptance scenarios; the split keeps fast unit feedback while integration tests
  cover the OWASP-sensitive edges (auth, uploads).
- **Alternatives considered**: Mock-based repository tests only (rejected — SQL and
  migrations are where regressions live); 100% coverage mandate (rejected — YAGNI,
  drives test theater in adapters).

## R10 — Adaptive placement (CAMST) mechanics

- **Decision**: Five difficulty bands (A1, A2, B1, B2, C1). Session starts at band B1.
  Serve testlets of 3 calibrated questions; score > 70% → move up one band, < 40% → move
  down one band, else stay. Hard stop after 12 questions (4 testlets); final level =
  band function over the last two testlets (Basic = A1/A2, Intermediate = B1/B2,
  Advanced = C1). Session state persisted server-side after every answer (resume
  support, RF/US1 scenario 6); questions drawn without repetition per session.
- **Rationale**: Implements FR-003/004/005 exactly; server-side state prevents client
  tampering (OWASP A04) and enables resume.
- **Alternatives considered**: IRT-based ability estimation (statistically superior but
  needs calibration data the MVP doesn't have; the band walk is the PRD's stated model).

## R11 — Audio capture format & upload

- **Decision**: `MediaDevices.getUserMedia` + `MediaRecorder`; prefer
  `audio/webm;codecs=opus`, fall back to `audio/mp4` (AAC) on Safari — detect via
  `MediaRecorder.isTypeSupported`. Hard 30 s client cap (FR-013) + server-side duration
  re-check. Upload as `multipart/form-data` to the speech endpoint; both Groq and OpenAI
  accept webm/opus and mp4 uploads directly, so no server-side transcoding in MVP.
- **Rationale**: Avoids the Web Speech API entirely (PRD's core risk mitigation);
  no ffmpeg dependency keeps the backend container minimal and the latency budget intact.
- **Alternatives considered**: WAV PCM capture (3–10× larger uploads on 4G — hurts
  RNF-01); server transcoding pipeline (unneeded while both providers accept the
  native container formats).
