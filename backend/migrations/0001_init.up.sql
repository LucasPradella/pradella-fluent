CREATE EXTENSION IF NOT EXISTS citext;

CREATE TYPE proficiency_level AS ENUM ('basic', 'intermediate', 'advanced');
CREATE TYPE auth_provider AS ENUM ('github', 'google', 'email');
CREATE TYPE theme_type AS ENUM ('travel', 'tech');
CREATE TYPE exercise_type AS ENUM ('translate', 'fill_blank', 'listening_choice', 'listening_order', 'speaking');
CREATE TYPE cefr_band AS ENUM ('A1', 'A2', 'B1', 'B2', 'C1');
CREATE TYPE placement_question_type AS ENUM ('choice', 'listening_choice', 'order');
CREATE TYPE placement_status AS ENUM ('active', 'completed', 'abandoned');
CREATE TYPE review_result AS ENUM ('failed', 'passed');

CREATE TABLE users (
    id                uuid PRIMARY KEY,
    email             citext NOT NULL UNIQUE,
    password_hash     text,
    display_name      text NOT NULL CHECK (char_length(display_name) BETWEEN 1 AND 60),
    proficiency_level proficiency_level,
    current_streak    int NOT NULL DEFAULT 0 CHECK (current_streak >= 0),
    longest_streak    int NOT NULL DEFAULT 0 CHECK (longest_streak >= 0),
    timezone          text NOT NULL DEFAULT 'America/Sao_Paulo',
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE auth_identities (
    id               uuid PRIMARY KEY,
    user_id          uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider         auth_provider NOT NULL,
    provider_subject text NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_subject)
);

CREATE TABLE sessions (
    id           uuid PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash   bytea NOT NULL UNIQUE,
    expires_at   timestamptz NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

CREATE TABLE modules (
    id               uuid PRIMARY KEY,
    title            text NOT NULL,
    description      text NOT NULL DEFAULT '',
    theme_type       theme_type NOT NULL,
    difficulty_level proficiency_level NOT NULL,
    sequential_order int NOT NULL,
    UNIQUE (theme_type, difficulty_level, sequential_order)
);

CREATE TABLE lessons (
    id                uuid PRIMARY KEY,
    module_id         uuid NOT NULL REFERENCES modules(id) ON DELETE CASCADE,
    title             text NOT NULL,
    pedagogical_focus text NOT NULL DEFAULT '',
    xp_reward         int NOT NULL CHECK (xp_reward > 0),
    sequential_order  int NOT NULL,
    UNIQUE (module_id, sequential_order)
);

CREATE TABLE exercises (
    id                 uuid PRIMARY KEY,
    lesson_id          uuid NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    exercise_type      exercise_type NOT NULL,
    prompt_context     text NOT NULL,
    target_answer_text text NOT NULL,
    options            jsonb,
    audio_asset_url    text,
    sequential_order   int NOT NULL,
    UNIQUE (lesson_id, sequential_order),
    CHECK (
        (exercise_type IN ('listening_choice', 'listening_order') AND audio_asset_url IS NOT NULL)
        OR (exercise_type NOT IN ('listening_choice', 'listening_order'))
    )
);

CREATE TABLE placement_questions (
    id             uuid PRIMARY KEY,
    cefr_band      cefr_band NOT NULL,
    question_type  placement_question_type NOT NULL,
    prompt         text NOT NULL,
    options        jsonb NOT NULL,
    correct_option text NOT NULL,
    audio_asset_url text
);
CREATE INDEX placement_questions_band_idx ON placement_questions (cefr_band);

CREATE TABLE placement_sessions (
    id               uuid PRIMARY KEY,
    user_id          uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status           placement_status NOT NULL DEFAULT 'active',
    current_band     cefr_band NOT NULL DEFAULT 'B1',
    questions_served int NOT NULL DEFAULT 0 CHECK (questions_served BETWEEN 0 AND 12),
    assigned_level   proficiency_level,
    created_at       timestamptz NOT NULL DEFAULT now(),
    completed_at     timestamptz
);
-- one active session per user (data-model rule)
CREATE UNIQUE INDEX placement_sessions_one_active_idx
    ON placement_sessions (user_id) WHERE status = 'active';

CREATE TABLE placement_answers (
    id                   uuid PRIMARY KEY,
    placement_session_id uuid NOT NULL REFERENCES placement_sessions(id) ON DELETE CASCADE,
    question_id          uuid NOT NULL REFERENCES placement_questions(id),
    testlet_index        int NOT NULL CHECK (testlet_index BETWEEN 0 AND 3),
    is_correct           boolean NOT NULL,
    answered_at          timestamptz NOT NULL DEFAULT now(),
    UNIQUE (placement_session_id, question_id)
);

-- Immutable activity log (FR-011): INSERT-only, enforced by trigger so the
-- guarantee holds for any role, including the table owner in dev.
CREATE TABLE progress_logs (
    id             uuid PRIMARY KEY,
    user_id        uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exercise_id    uuid NOT NULL REFERENCES exercises(id),
    completed_at   timestamptz NOT NULL,
    accuracy_score numeric(5,4) NOT NULL CHECK (accuracy_score BETWEEN 0 AND 1),
    is_review      boolean NOT NULL DEFAULT false,
    detail         jsonb NOT NULL DEFAULT '{}'::jsonb
);
CREATE INDEX progress_logs_user_completed_idx ON progress_logs (user_id, completed_at);

CREATE FUNCTION progress_logs_immutable() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'progress_logs is INSERT-only (FR-011)';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER progress_logs_no_update_delete
    BEFORE UPDATE OR DELETE ON progress_logs
    FOR EACH ROW EXECUTE FUNCTION progress_logs_immutable();

CREATE TABLE review_queue_items (
    id            uuid PRIMARY KEY,
    user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exercise_id   uuid NOT NULL REFERENCES exercises(id),
    due_at        timestamptz NOT NULL,
    interval_days int NOT NULL DEFAULT 1 CHECK (interval_days >= 1),
    streak_at_7d  int NOT NULL DEFAULT 0,
    failure_count int NOT NULL CHECK (failure_count >= 1),
    last_result   review_result NOT NULL,
    UNIQUE (user_id, exercise_id)
);
CREATE INDEX review_queue_due_idx ON review_queue_items (user_id, due_at);
