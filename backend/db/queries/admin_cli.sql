-- name: CountUsersByPhone :one
SELECT count(*) FROM users WHERE phone = $1;

-- name: CreateAdminUser :one
INSERT INTO users (name, role, phone, password, phone_verified_at)
VALUES ($1, 'admin', $2, $3, now())
RETURNING id;
