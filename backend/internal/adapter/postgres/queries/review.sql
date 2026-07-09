-- name: GetReviewItem :one
SELECT * FROM review_queue_items WHERE user_id = $1 AND exercise_id = $2;

-- name: InsertReviewItem :exec
INSERT INTO review_queue_items (id, user_id, exercise_id, due_at, interval_days, streak_at_7d, failure_count, last_result)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: UpdateReviewItem :exec
UPDATE review_queue_items
SET due_at = $2, interval_days = $3, streak_at_7d = $4, failure_count = $5, last_result = $6
WHERE id = $1;

-- name: DeleteReviewItem :exec
DELETE FROM review_queue_items WHERE id = $1;

-- name: ListDueReviewItems :many
SELECT sqlc.embed(review_queue_items), sqlc.embed(exercises)
FROM review_queue_items
JOIN exercises ON exercises.id = review_queue_items.exercise_id
WHERE review_queue_items.user_id = $1 AND review_queue_items.due_at <= now()
ORDER BY review_queue_items.due_at;

-- name: CountDueReviewItems :one
SELECT count(*) FROM review_queue_items WHERE user_id = $1 AND due_at <= now();
