-- name: CreateAlert :one
INSERT INTO alerts (user_id, type, title, body, data)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListAlertsByUser :many
SELECT * FROM alerts
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountUnreadAlerts :one
SELECT count(*) FROM alerts WHERE user_id = $1 AND is_read = FALSE;

-- name: MarkAlertRead :exec
UPDATE alerts SET is_read = TRUE WHERE id = $1;

-- name: MarkAllAlertsRead :exec
UPDATE alerts SET is_read = TRUE WHERE user_id = $1 AND is_read = FALSE;
