-- name: CreateUser :one
INSERT INTO users (id, email, password_hash, display_name, timezone)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: SetProficiencyLevel :exec
UPDATE users SET proficiency_level = $2, updated_at = now() WHERE id = $1;

-- name: UpdateStreaks :exec
UPDATE users SET current_streak = $2, longest_streak = $3, updated_at = now() WHERE id = $1;

-- name: CreateAuthIdentity :exec
INSERT INTO auth_identities (id, user_id, provider, provider_subject)
VALUES ($1, $2, $3, $4);

-- name: GetAuthIdentity :one
SELECT * FROM auth_identities WHERE provider = $1 AND provider_subject = $2;
