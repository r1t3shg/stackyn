-- name: GetEnvVarByID :one
SELECT * FROM env_vars
WHERE id = $1 LIMIT 1;

-- name: GetEnvVarByAppIDAndKey :one
SELECT * FROM env_vars
WHERE app_id = $1 AND key = $2 LIMIT 1;

-- name: ListEnvVarsByAppID :many
SELECT * FROM env_vars
WHERE app_id = $1
ORDER BY key ASC;

-- name: CreateEnvVar :one
INSERT INTO env_vars (
    app_id, key, value
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: UpdateEnvVar :one
UPDATE env_vars
SET
    value = COALESCE($3, value),
    updated_at = NOW()
WHERE app_id = $1 AND key = $2
RETURNING *;

-- name: DeleteEnvVar :exec
DELETE FROM env_vars
WHERE app_id = $1 AND key = $2;

