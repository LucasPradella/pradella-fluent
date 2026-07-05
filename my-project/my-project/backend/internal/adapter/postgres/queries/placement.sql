-- name: GetActivePlacementSession :one
SELECT * FROM placement_sessions WHERE user_id = $1 AND status = 'active';

-- name: CreatePlacementSession :one
INSERT INTO placement_sessions (id, user_id) VALUES ($1, $2) RETURNING *;

-- name: AbandonActivePlacementSessions :exec
UPDATE placement_sessions SET status = 'abandoned' WHERE user_id = $1 AND status = 'active';

-- name: UpdatePlacementProgress :exec
UPDATE placement_sessions
SET current_band = $2, questions_served = $3
WHERE id = $1;

-- name: CompletePlacementSession :exec
UPDATE placement_sessions
SET status = 'completed', assigned_level = $2, completed_at = now()
WHERE id = $1;

-- name: InsertPlacementAnswer :exec
INSERT INTO placement_answers (id, placement_session_id, question_id, testlet_index, is_correct)
VALUES ($1, $2, $3, $4, $5);

-- name: ListPlacementAnswers :many
SELECT * FROM placement_answers
WHERE placement_session_id = $1
ORDER BY answered_at;

-- name: GetPlacementQuestion :one
SELECT * FROM placement_questions WHERE id = $1;

-- name: PickUnservedQuestionForBand :one
SELECT * FROM placement_questions q
WHERE q.cefr_band = $1
  AND NOT EXISTS (
      SELECT 1 FROM placement_answers a
      WHERE a.placement_session_id = $2 AND a.question_id = q.id
  )
ORDER BY random()
LIMIT 1;

-- name: CountPlacementQuestions :one
SELECT count(*) FROM placement_questions;

-- name: InsertPlacementQuestion :exec
INSERT INTO placement_questions (id, cefr_band, question_type, prompt, options, correct_option, audio_asset_url)
VALUES ($1, $2, $3, $4, $5, $6, $7);
