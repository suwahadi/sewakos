-- name: CreateAccessToken :one
INSERT INTO access_tokens (user_id, token_hash)
VALUES ($1, $2)
RETURNING id;

-- name: GetAccessTokenForVerification :one
SELECT at.id, at.token_hash, u.id AS user_id, u.role
FROM access_tokens at
JOIN users u ON u.id = at.user_id
WHERE at.id = $1;

-- name: TouchAccessToken :exec
UPDATE access_tokens
SET last_used_at = now(), updated_at = now()
WHERE id = $1;
