-- name: GetPlanByID :one
SELECT * FROM plans
WHERE id = $1 LIMIT 1;

-- name: GetPlanByName :one
SELECT * FROM plans
WHERE name = $1 LIMIT 1;

-- name: ListPlans :many
SELECT * FROM plans
ORDER BY price ASC;

-- name: CreatePlan :one
INSERT INTO plans (
    name, display_name, price, max_ram_mb, max_disk_mb, max_apps,
    always_on, auto_deploy, health_checks, logs, zero_downtime,
    workers, priority_builds, manual_deploy_only
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
) RETURNING *;

-- name: UpdatePlan :one
UPDATE plans
SET
    display_name = COALESCE($2, display_name),
    price = COALESCE($3, price),
    max_ram_mb = COALESCE($4, max_ram_mb),
    max_disk_mb = COALESCE($5, max_disk_mb),
    max_apps = COALESCE($6, max_apps),
    always_on = COALESCE($7, always_on),
    auto_deploy = COALESCE($8, auto_deploy),
    health_checks = COALESCE($9, health_checks),
    logs = COALESCE($10, logs),
    zero_downtime = COALESCE($11, zero_downtime),
    workers = COALESCE($12, workers),
    priority_builds = COALESCE($13, priority_builds),
    manual_deploy_only = COALESCE($14, manual_deploy_only),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeletePlan :exec
DELETE FROM plans
WHERE id = $1;

