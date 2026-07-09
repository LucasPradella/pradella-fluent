# Data Model: FluentDev — English Learning PWA

**Feature**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md) | **Date**: 2026-07-04

PostgreSQL 16 relational model. All primary keys are UUID v7 (time-ordered, generated
server-side; client-generated for offline-outbox rows, deduplicated on sync). All
timestamps are `timestamptz` stored in UTC; streak/heatmap day-bucketing applies the
user's timezone at read time (edge case: midnight boundary).

## Entity Overview

```text
users 1──n auth_identities
users 1──n sessions
users 1──1 placement_sessions (active) ── n placement_answers ── placement_questions
modules 1──n lessons 1──n exercises
users 1──n progress_logs n──1 exercises
users 1──n review_queue_items n──1 exercises
```

## users

| Field | Type | Constraints / Notes |
|-------|------|---------------------|
| id | uuid | PK |
| email | citext | UNIQUE, NOT NULL, RFC-format validated at adapter |
| password_hash | text | NULL for OAuth-only accounts; argon2id encoded string |
| display_name | text | NOT NULL, 1–60 chars, sanitized (no control chars) |
| proficiency_level | enum `basic \| intermediate \| advanced` | NULL until placement completes (FR-005) |
| current_streak | int | ≥ 0, derived-but-materialized for dashboard reads (FR-018) |
| longest_streak | int | ≥ 0, monotonically non-decreasing |
| timezone | text | IANA name, default `America/Sao_Paulo`; drives day bucketing |
| created_at / updated_at | timestamptz | NOT NULL |

**Rules**: streak fields are recomputed by the progress usecase inside the same
transaction that inserts a `progress_logs` row — never trusted from the client.

## auth_identities

| Field | Type | Constraints / Notes |
|-------|------|---------------------|
| id | uuid | PK |
| user_id | uuid | FK → users, ON DELETE CASCADE |
| provider | enum `github \| google \| email` | |
| provider_subject | text | provider's stable user id; UNIQUE(provider, provider_subject) |
| created_at | timestamptz | |

**Rules**: an e-mail signup creates `provider='email'`; linking the same verified e-mail
from OAuth attaches a new identity to the existing user rather than creating a duplicate.

## sessions

| Field | Type | Constraints / Notes |
|-------|------|---------------------|
| id | uuid | PK |
| user_id | uuid | FK → users, ON DELETE CASCADE |
| token_hash | bytea | SHA-256 of the opaque cookie token; UNIQUE; raw token never stored |
| expires_at | timestamptz | sliding expiry, max 30 days |
| created_at / last_seen_at | timestamptz | |

## modules

| Field | Type | Constraints / Notes |
|-------|------|---------------------|
| id | uuid | PK |
| title | text | NOT NULL |
| description | text | |
| theme_type | enum `travel \| tech` | FR-007 |
| difficulty_level | enum `basic \| intermediate \| advanced` | gates access vs `users.proficiency_level` (FR-006) |
| sequential_order | int | UNIQUE within (theme_type, difficulty_level) |

## lessons

| Field | Type | Constraints / Notes |
|-------|------|---------------------|
| id | uuid | PK |
| module_id | uuid | FK → modules |
| title | text | NOT NULL |
| pedagogical_focus | text | task/scenario framing (FR-008) |
| xp_reward | int | > 0 (FR-011) |
| sequential_order | int | UNIQUE within module |

## exercises

| Field | Type | Constraints / Notes |
|-------|------|---------------------|
| id | uuid | PK |
| lesson_id | uuid | FK → lessons |
| exercise_type | enum `translate \| fill_blank \| listening_choice \| listening_order \| speaking` | FR-009/010/013 |
| prompt_context | text | the task scenario shown to the learner |
| target_answer_text | text | canonical answer / sentence to speak |
| options | jsonb | choices or word blocks for listening types; NULL otherwise |
| audio_asset_url | text | required for listening types; NULL otherwise (CHECK) |
| sequential_order | int | UNIQUE within lesson |

## placement_questions

| Field | Type | Constraints / Notes |
|-------|------|---------------------|
| id | uuid | PK |
| cefr_band | enum `A1 \| A2 \| B1 \| B2 \| C1` | FR-003 |
| question_type | enum `choice \| listening_choice \| order` | placement is text/listening only (spec assumption) |
| prompt / options / correct_option | text / jsonb / text | correct answer never sent to the client |

## placement_sessions

| Field | Type | Constraints / Notes |
|-------|------|---------------------|
| id | uuid | PK |
| user_id | uuid | FK → users; partial UNIQUE index on (user_id) WHERE status='active' — one active session |
| status | enum `active \| completed \| abandoned` | |
| current_band | enum CEFR | starts `B1` (research R10) |
| questions_served | int | 0–12; hard stop at 12 (FR-005) |
| assigned_level | enum `basic \| intermediate \| advanced` | NULL until completed |
| created_at / completed_at | timestamptz | |

**State transitions**: `active → completed` (12th answer scored) — sets
`assigned_level` and `users.proficiency_level` atomically; `active → abandoned` only by
explicit user restart (resume is supported, US1 scenario 6).

## placement_answers

| Field | Type | Constraints / Notes |
|-------|------|---------------------|
| id | uuid | PK |
| placement_session_id | uuid | FK → placement_sessions |
| question_id | uuid | FK → placement_questions; UNIQUE(session, question) — no repeats |
| testlet_index | int | 0–3 |
| is_correct | boolean | scored server-side only |
| answered_at | timestamptz | |

## progress_logs (immutable — FR-011)

| Field | Type | Constraints / Notes |
|-------|------|---------------------|
| id | uuid | PK — client-generated for offline outbox rows; PK collision on sync = duplicate replay, ignored (idempotent) |
| user_id | uuid | FK → users |
| exercise_id | uuid | FK → exercises |
| completed_at | timestamptz | client-reported for offline completions, clamped to ≤ server now |
| accuracy_score | numeric(5,4) | 0–1 |
| is_review | boolean | spaced-repetition review flag (FR-020) |
| detail | jsonb | e.g., speaking: similarity, missed words, provider used |

**Rules**: INSERT-only (no UPDATE/DELETE grants for the app role). Feeds streak,
heatmap (90-day aggregation by user-timezone day), and XP.

## review_queue_items

| Field | Type | Constraints / Notes |
|-------|------|---------------------|
| id | uuid | PK |
| user_id | uuid | FK → users; UNIQUE(user_id, exercise_id) |
| exercise_id | uuid | FK → exercises |
| due_at | timestamptz | next appearance (FR-020) |
| failure_count | int | ≥ 1; repeated failures shorten the next interval (edge case) |
| last_result | enum `failed \| passed` | |

**Spacing rule (domain)**: intervals 1 d → 3 d → 7 d → 21 d on success; reset to 1 d on
failure. Item leaves the queue after two consecutive passes at ≥ 7 d spacing.

## Client-side (IndexedDB via Dexie — cache only, FR-023)

- `cachedProfile`, `cachedTracks`, `cachedLessons` — SWR mirrors of API reads.
- `outbox` — pending `progress_logs` rows created offline `{id (uuid), payload,
  createdAt}`; replayed FIFO on reconnect; server dedupes by PK (idempotent).
- Never authoritative; full wipe (Safari 7-day eviction) is recoverable from the API
  (US5 scenario 4).
