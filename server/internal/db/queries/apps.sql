-- name: GetAppByID :one
SELECT * FROM apps
WHERE id = $1 LIMIT 1;

-- name: GetAppBySlug :one
SELECT * FROM apps
WHERE slug = $1 LIMIT 1;

-- name: ListAppsByUserID :many
SELECT * FROM apps
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: CreateApp :one
INSERT INTO apps (
    user_id, name, slug, status, url, repo_url, branch
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: UpdateApp :one
UPDATE apps
SET
    name = COALESCE($2, name),
    slug = COALESCE($3, slug),
    status = COALESCE($4, status),
    url = COALESCE($5, url),
    repo_url = COALESCE($6, repo_url),
    branch = COALESCE($7, branch),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteApp :exec
DELETE FROM apps
WHERE id = $1;

