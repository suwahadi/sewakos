-- name: AdminListAreas :many
SELECT id, name, slug, type, is_top, latitude, longitude, place_id
FROM areas
WHERE (sqlc.narg(type)::text IS NULL OR type = sqlc.narg(type))
ORDER BY id
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: AdminLockArea :one
SELECT id, name, slug, type, is_top, latitude, longitude, place_id FROM areas WHERE id = sqlc.arg(id) FOR UPDATE;

-- name: AreaSlugTaken :one
SELECT EXISTS (SELECT 1 FROM areas WHERE slug = sqlc.arg(slug) AND id <> sqlc.arg(except_id));

-- name: AdminInsertArea :one
INSERT INTO areas (name, slug, type, is_top, latitude, longitude, place_id)
VALUES (sqlc.arg(name), sqlc.arg(slug), sqlc.arg(type), sqlc.arg(is_top), sqlc.arg(latitude), sqlc.arg(longitude), sqlc.narg(place_id))
RETURNING id;

-- name: AdminUpdateArea :exec
UPDATE areas SET
  name = sqlc.arg(name), slug = sqlc.arg(slug), type = sqlc.arg(type), is_top = sqlc.arg(is_top),
  latitude = sqlc.arg(latitude), longitude = sqlc.arg(longitude), place_id = sqlc.narg(place_id), updated_at = now()
WHERE id = sqlc.arg(id);

-- name: AreaInUse :one
SELECT EXISTS (SELECT 1 FROM properties WHERE area_id = sqlc.arg(area_id));

-- name: AdminDeleteArea :exec
DELETE FROM areas WHERE id = sqlc.arg(id);

-- name: AdminListUsers :many
SELECT id, name, role, phone, email, phone_verified_at, suspended_at, created_at
FROM users
WHERE (sqlc.narg(role)::text IS NULL OR role = sqlc.narg(role))
  AND (sqlc.narg(q)::text IS NULL OR name ILIKE '%' || sqlc.narg(q) || '%' OR phone LIKE '%' || sqlc.narg(q) || '%')
ORDER BY id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: AdminLockUser :one
SELECT id, name, role, phone, email, phone_verified_at, suspended_at, created_at FROM users WHERE id = sqlc.arg(id) FOR UPDATE;

-- name: SetUserSuspended :exec
UPDATE users SET
  suspended_at = CASE WHEN sqlc.arg(suspended)::boolean THEN COALESCE(suspended_at, now()) END,
  updated_at = now()
WHERE id = sqlc.arg(id);
