-- name: GetUserByPhone :one
SELECT id, name, role, password, phone_verified_at, suspended_at
FROM users
WHERE phone = $1;
