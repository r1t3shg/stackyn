-- name: GetRuntimeInstanceByID :one
SELECT * FROM runtime_instances
WHERE id = $1 LIMIT 1;

-- name: GetRuntimeInstanceByContainerID :one
SELECT * FROM runtime_instances
WHERE container_id = $1 LIMIT 1;

-- name: ListRuntimeInstancesByDeploymentID :many
SELECT * FROM runtime_instances
WHERE deployment_id = $1
ORDER BY created_at DESC;

-- name: ListRuntimeInstancesByStatus :many
SELECT * FROM runtime_instances
WHERE status = $1
ORDER BY created_at ASC;

-- name: CreateRuntimeInstance :one
INSERT INTO runtime_instances (
    deployment_id, container_id, status, memory_mb, cpu, disk_gb,
    memory_usage_mb, memory_usage_percent, disk_usage_gb,
    disk_usage_percent, restart_count
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: UpdateRuntimeInstance :one
UPDATE runtime_instances
SET
    status = COALESCE($2, status),
    memory_mb = COALESCE($3, memory_mb),
    cpu = COALESCE($4, cpu),
    disk_gb = COALESCE($5, disk_gb),
    memory_usage_mb = COALESCE($6, memory_usage_mb),
    memory_usage_percent = COALESCE($7, memory_usage_percent),
    disk_usage_gb = COALESCE($8, disk_usage_gb),
    disk_usage_percent = COALESCE($9, disk_usage_percent),
    restart_count = COALESCE($10, restart_count),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteRuntimeInstance :exec
DELETE FROM runtime_instances
WHERE id = $1;

