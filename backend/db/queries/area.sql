-- name: ListAreasByType :many
SELECT id, name, slug, latitude, longitude
FROM areas
WHERE type = $1 AND (sqlc.narg(is_top)::bool IS NULL OR is_top = sqlc.narg(is_top))
ORDER BY id ASC;

-- name: GetAreaByID :one
SELECT id, name, type, latitude, longitude
FROM areas
WHERE id = $1;
