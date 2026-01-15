-- name: CreateLogChunk :one
INSERT INTO app_logs (app_id, build_job_id, deployment_id, log_type, chunk_index, content, size_bytes)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetLogsByAppID :many
SELECT * FROM app_logs
WHERE app_id = $1
  AND log_type = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetLogsByBuildJobID :many
SELECT * FROM app_logs
WHERE app_id = $1
  AND build_job_id = $2
  AND log_type = 'build'
ORDER BY chunk_index ASC, created_at ASC;

-- name: GetLogsByDeploymentID :many
SELECT * FROM app_logs
WHERE app_id = $1
  AND deployment_id = $2
  AND log_type = 'runtime'
ORDER BY chunk_index ASC, created_at ASC;

-- name: GetAppLogStorageSize :one
SELECT get_app_log_storage_size($1) as total_size;

-- name: DeleteOldLogs :exec
DELETE FROM app_logs
WHERE app_id = $1
  AND created_at < $2;

-- name: DeleteLogsByAppID :exec
DELETE FROM app_logs
WHERE app_id = $1;

