-- name: GetUserByPhone :one
SELECT id, name, role, password, phone_verified_at, suspended_at
FROM users
WHERE phone = $1;

-- name: GetUserByID :one
SELECT id, name, role, phone, email, phone_verified_at, suspended_at
FROM users
WHERE id = $1;

-- name: GetUserPasswordByID :one
SELECT password FROM users WHERE id = $1;

-- name: ExistsUserByPhone :one
SELECT EXISTS(SELECT 1 FROM users WHERE phone = $1);

-- name: ExistsUserByEmail :one
SELECT EXISTS(SELECT 1 FROM users WHERE email = $1);

-- name: CreateUser :one
INSERT INTO users (name, phone, email, password, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: VerifyUserPhone :exec
UPDATE users
SET phone_verified_at = now(), updated_at = now()
WHERE id = $1;

-- name: UpdateUserProfile :exec
UPDATE users
SET name = $2, email = $3, updated_at = now()
WHERE id = $1;

-- name: UpdateUserPassword :exec
UPDATE users
SET password = $2, updated_at = now()
WHERE id = $1;

-- name: UpdateUserPasswordAndVerify :exec
UPDATE users
SET password = $2, phone_verified_at = now(), updated_at = now()
WHERE id = $1;

-- name: UpdateUserPhone :exec
UPDATE users
SET phone = $2, updated_at = now()
WHERE id = $1;
