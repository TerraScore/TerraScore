-- name: CreateAgent :one
INSERT INTO agents (full_name, phone, email, date_of_birth, home_location, state_code, district_code, keycloak_id)
VALUES ($1, $2, $3, $4, ST_SetSRID(ST_MakePoint($5, $6), 4326), $7, $8, $9)
RETURNING *;

-- name: GetAgentByID :one
SELECT * FROM agents WHERE id = $1;

-- name: GetAgentByPhone :one
SELECT * FROM agents WHERE phone = $1;

-- name: GetAgentByKeycloakID :one
SELECT * FROM agents WHERE keycloak_id = $1;

-- name: FindNearbyAgents :many
SELECT *,
    ST_Distance(last_known_location::geography, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) / 1000 AS distance_km
FROM agents
WHERE status = 'active'
    AND is_online = TRUE
    AND ST_DWithin(
        last_known_location::geography,
        ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
        $3
    )
ORDER BY distance_km ASC
LIMIT $4;

-- name: UpdateAgentLocation :exec
UPDATE agents SET
    last_known_location = ST_SetSRID(ST_MakePoint($2, $3), 4326),
    last_location_at = NOW(),
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateAgentStatus :exec
UPDATE agents SET status = $2, updated_at = NOW() WHERE id = $1;

-- name: UpdateAgentOnlineStatus :exec
UPDATE agents SET is_online = $2, updated_at = NOW() WHERE id = $1;

-- name: UpdateAgentStats :exec
UPDATE agents SET
    total_jobs_completed = $2,
    avg_rating = $3,
    completion_rate = $4,
    qa_pass_rate = $5,
    last_job_completed_at = NOW(),
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateAgentFCMToken :exec
UPDATE agents SET fcm_token = $2, device_id = $3, app_version = $4, updated_at = NOW() WHERE id = $1;

-- name: ListAgents :many
SELECT * FROM agents
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountAgents :one
SELECT count(*) FROM agents;
