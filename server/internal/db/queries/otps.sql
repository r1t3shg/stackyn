-- name: CreateOTP :one
INSERT INTO otps (
    email, otp_hash, expires_at
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: GetOTPByEmail :many
SELECT * FROM otps
WHERE email = $1 AND used = false AND expires_at > NOW()
ORDER BY created_at DESC
LIMIT 1;

-- name: MarkOTPAsUsed :exec
UPDATE otps
SET used = true
WHERE id = $1;

-- name: CleanupExpiredOTPs :exec
DELETE FROM otps
WHERE expires_at < NOW() OR used = true;

