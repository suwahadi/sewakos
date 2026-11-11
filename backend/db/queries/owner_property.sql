-- name: ListOwnerProperties :many
SELECT
  p.id, p.name, p.slug, p.gender, p.description, p.address, p.latitude, p.longitude,
  p.area_id, a.name AS area_name, p.is_active, p.moderation_status, p.moderation_note,
  p.is_flash_sale, p.flash_sale_discount, p.views,
  (SELECT count(*) FROM rooms r WHERE r.property_id = p.id) AS rooms_count,
  coalesce(rv.rating_average, 0)::float8 AS rating_average,
  coalesce(rv.rating_count, 0)::bigint AS rating_count,
  p.created_at
FROM properties p
JOIN areas a ON a.id = p.area_id
LEFT JOIN (
  SELECT property_id, round(avg(rating)::numeric, 1)::float8 AS rating_average, count(*) AS rating_count
  FROM reviews WHERE NOT is_hidden GROUP BY property_id
) rv ON rv.property_id = p.id
WHERE p.owner_id = sqlc.arg(owner_id)
  AND (sqlc.narg(property_id)::bigint IS NULL OR p.id = sqlc.narg(property_id))
  AND (sqlc.narg(moderation_status)::text IS NULL OR p.moderation_status = sqlc.narg(moderation_status))
ORDER BY p.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: ListPhotosForProperties :many
SELECT id, property_id, room_id, object_key, thumb_key, caption, sort_order
FROM property_photos
WHERE property_id = ANY(sqlc.arg(property_ids)::bigint[])
ORDER BY property_id, sort_order, id;

-- name: InsertProperty :one
INSERT INTO properties (owner_id, area_id, name, slug, gender, description, address, latitude, longitude)
VALUES (
  sqlc.arg(owner_id), sqlc.arg(area_id), sqlc.arg(name), sqlc.arg(slug), sqlc.arg(gender),
  sqlc.arg(description), sqlc.arg(address), sqlc.arg(latitude), sqlc.arg(longitude)
)
RETURNING id;

-- name: SetPropertySlug :exec
UPDATE properties SET slug = sqlc.arg(slug) WHERE id = sqlc.arg(id);

-- name: LockOwnerPropertyForEdit :one
SELECT id, area_id, name, gender, description, address, latitude, longitude, is_active,
  moderation_status, moderation_note, is_flash_sale, flash_sale_discount
FROM properties
WHERE id = sqlc.arg(id) AND owner_id = sqlc.arg(owner_id)
FOR UPDATE;

-- name: UpdateProperty :exec
UPDATE properties SET
  area_id = sqlc.arg(area_id), name = sqlc.arg(name), gender = sqlc.arg(gender),
  description = sqlc.arg(description), address = sqlc.arg(address),
  latitude = sqlc.arg(latitude), longitude = sqlc.arg(longitude), is_active = sqlc.arg(is_active),
  is_flash_sale = sqlc.arg(is_flash_sale), flash_sale_discount = sqlc.arg(flash_sale_discount),
  updated_at = now()
WHERE id = sqlc.arg(id);

-- name: ResubmitProperty :exec
UPDATE properties SET moderation_status = 'pending', moderation_note = NULL, updated_at = now()
WHERE id = sqlc.arg(id);
