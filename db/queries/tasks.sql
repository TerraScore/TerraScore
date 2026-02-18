-- name: EnqueueTask :one
INSERT INTO task_queue (task_type, payload, priority, scheduled_at)
VALUES ($1, $2, $3, COALESCE(sqlc.narg('scheduled_at'), NOW()))
RETURNING *;

-- name: ClaimTask :one
UPDATE task_queue
SET status = 'processing', started_at = NOW(), attempts = attempts + 1
WHERE id = (
    SELECT id FROM task_queue
    WHERE status = 'pending' AND scheduled_at <= NOW()
    ORDER BY priority DESC, scheduled_at ASC
    LIMIT 1
    FOR UPDATE SKIP LOCKED
)
RETURNING *;

-- name: CompleteTask :exec
UPDATE task_queue SET status = 'completed', completed_at = NOW() WHERE id = $1;

-- name: FailTask :exec
UPDATE task_queue SET
    status = CASE WHEN attempts >= max_attempts THEN 'dead' ELSE 'pending' END,
    last_error = $2,
    error_message = $2,
    completed_at = CASE WHEN attempts >= max_attempts THEN NOW() ELSE NULL END
WHERE id = $1;

-- name: CountPendingTasks :one
SELECT count(*) FROM task_queue WHERE status = 'pending';
