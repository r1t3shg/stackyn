-- name: GetDeploymentByID :one
SELECT * FROM deployments
WHERE id = $1 LIMIT 1;

-- name: ListDeploymentsByAppID :many
SELECT * FROM deployments
WHERE app_id = $1
ORDER BY created_at DESC;

-- name: ListDeploymentsByStatus :many
SELECT * FROM deployments
WHERE status = $1
ORDER BY created_at ASC;

-- name: GetDeploymentBySubdomain :one
SELECT * FROM deployments
WHERE subdomain = $1 LIMIT 1;

-- name: CreateDeployment :one
INSERT INTO deployments (
    app_id, build_job_id, status, image_name, container_id, subdomain,
    build_log, runtime_log, error_message
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: UpdateDeployment :one
UPDATE deployments
SET
    build_job_id = COALESCE($2, build_job_id),
    status = COALESCE($3, status),
    image_name = COALESCE($4, image_name),
    container_id = COALESCE($5, container_id),
    subdomain = COALESCE($6, subdomain),
    build_log = COALESCE($7, build_log),
    runtime_log = COALESCE($8, runtime_log),
    error_message = COALESCE($9, error_message),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteDeployment :exec
DELETE FROM deployments
WHERE id = $1;

