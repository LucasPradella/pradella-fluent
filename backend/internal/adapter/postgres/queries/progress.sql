-- name: InsertProgressLog :execrows
INSERT INTO progress_logs (id, user_id, exercise_id, completed_at, accuracy_score, is_review, detail)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (id) DO NOTHING;

-- name: GetProgressLog :one
SELECT * FROM progress_logs WHERE id = $1;

-- name: ListActivityTimestamps :many
SELECT completed_at FROM progress_logs
WHERE user_id = $1 AND completed_at >= now() - interval '400 days'
ORDER BY completed_at;

-- name: HeatmapBuckets :many
SELECT (completed_at AT TIME ZONE sqlc.arg(tz)::text)::date AS day, count(*) AS interactions
FROM progress_logs
WHERE user_id = $1 AND completed_at >= now() - interval '91 days'
GROUP BY 1
ORDER BY 1;

-- name: TotalXP :one
SELECT COALESCE(SUM((detail->>'xpAwarded')::int), 0)::int AS total_xp
FROM progress_logs
WHERE user_id = $1 AND detail ? 'xpAwarded';

-- name: CountUnpassedExercisesInLesson :one
SELECT count(*) FROM exercises e
WHERE e.lesson_id = $1
  AND NOT EXISTS (
      SELECT 1 FROM progress_logs p
      WHERE p.user_id = $2 AND p.exercise_id = e.id AND p.accuracy_score >= 0.8
  );

-- name: HasLessonXPAward :one
SELECT EXISTS (
    SELECT 1 FROM progress_logs
    WHERE user_id = $1 AND detail->>'xpLessonId' = sqlc.arg(lesson_id)::text
) AS awarded;
