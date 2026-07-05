-- name: ListModules :many
SELECT * FROM modules ORDER BY difficulty_level, theme_type, sequential_order;

-- name: ListLessons :many
SELECT * FROM lessons ORDER BY module_id, sequential_order;

-- name: GetLessonWithModule :one
SELECT sqlc.embed(lessons), sqlc.embed(modules)
FROM lessons JOIN modules ON modules.id = lessons.module_id
WHERE lessons.id = $1;

-- name: ListExercisesByLesson :many
SELECT * FROM exercises WHERE lesson_id = $1 ORDER BY sequential_order;

-- name: GetExerciseWithLesson :one
SELECT sqlc.embed(exercises), sqlc.embed(lessons), sqlc.embed(modules)
FROM exercises
JOIN lessons ON lessons.id = exercises.lesson_id
JOIN modules ON modules.id = lessons.module_id
WHERE exercises.id = $1;

-- name: ListCompletedLessonIDs :many
SELECT l.id FROM lessons l
JOIN exercises e ON e.lesson_id = l.id
LEFT JOIN (
    SELECT DISTINCT exercise_id FROM progress_logs
    WHERE user_id = $1 AND accuracy_score >= 0.8
) p ON p.exercise_id = e.id
GROUP BY l.id
HAVING count(*) = count(p.exercise_id);

-- name: CountModules :one
SELECT count(*) FROM modules;

-- name: InsertModule :exec
INSERT INTO modules (id, title, description, theme_type, difficulty_level, sequential_order)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: InsertLesson :exec
INSERT INTO lessons (id, module_id, title, pedagogical_focus, xp_reward, sequential_order)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: InsertExercise :exec
INSERT INTO exercises (id, lesson_id, exercise_type, prompt_context, target_answer_text, options, audio_asset_url, sequential_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);
