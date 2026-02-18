-- name: CreateUser :one
INSERT INTO users (phone, email, full_name, role, state_code, district_code, city, keycloak_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByPhone :one
SELECT * FROM users WHERE phone = $1;

-- name: GetUserByKeycloakID :one
SELECT * FROM users WHERE keycloak_id = $1;

-- name: UpdateUser :one
UPDATE users SET
    email = COALESCE(sqlc.narg('email'), email),
    full_name = COALESCE(sqlc.narg('full_name'), full_name),
    avatar_url = COALESCE(sqlc.narg('avatar_url'), avatar_url),
    state_code = COALESCE(sqlc.narg('state_code'), state_code),
    district_code = COALESCE(sqlc.narg('district_code'), district_code),
    city = COALESCE(sqlc.narg('city'), city),
    language = COALESCE(sqlc.narg('language'), language),
    notification_prefs = COALESCE(sqlc.narg('notification_prefs'), notification_prefs),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateUserStatus :exec
UPDATE users SET status = $2, updated_at = NOW() WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT count(*) FROM users;
