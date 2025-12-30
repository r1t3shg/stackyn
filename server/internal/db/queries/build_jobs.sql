-- name: GetBuildJobByID :one
SELECT * FROM build_jobs
WHERE id = $1 LIMIT 1;

-- name: ListBuildJobsByAppID :many
SELECT * FROM build_jobs
WHERE app_id = $1
ORDER BY created_at DESC;

-- name: ListBuildJobsByStatus :many
SELECT * FROM build_jobs
WHERE status = $1
ORDER BY created_at ASC;

-- name: CreateBuildJob :one
INSERT INTO build_jobs (
    app_id, status, build_log, error_message
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: UpdateBuildJob :one
UPDATE build_jobs
SET
    status = COALESCE($2, status),
    build_log = COALESCE($3, build_log),
    error_message = COALESCE($4, error_message),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteBuildJob :exec
DELETE FROM build_jobs
WHERE id = $1;

