-- name: CountUsersByPhone :one
SELECT count(*) FROM users WHERE phone = $1;

-- name: CreateAdminUser :one
INSERT INTO users (name, role, phone, email, password, phone_verified_at)
VALUES (sqlc.arg(name), 'admin', sqlc.arg(phone), sqlc.narg(email), sqlc.arg(password), now())
RETURNING id;
