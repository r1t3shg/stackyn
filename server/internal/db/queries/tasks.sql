-- name: CreateTaskState :one
INSERT INTO task_states (
    task_id, task_type, queue_name, payload, status, max_retries
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: UpdateTaskState :one
UPDATE task_states
SET
    status = COALESCE($2, status),
    retry_count = COALESCE($3, retry_count),
    error_message = COALESCE($4, error_message),
    completed_at = CASE WHEN $2 = 'completed' THEN NOW() ELSE completed_at END,
    failed_at = CASE WHEN $2 = 'failed' THEN NOW() ELSE failed_at END,
    updated_at = NOW()
WHERE task_id = $1
RETURNING *;

-- name: GetTaskStateByID :one
SELECT * FROM task_states
WHERE task_id = $1 LIMIT 1;

-- name: ListTaskStatesByStatus :many
SELECT * FROM task_states
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListTaskStatesByType :many
SELECT * FROM task_states
WHERE task_type = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

