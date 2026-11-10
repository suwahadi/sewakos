-- name: GetListingProperty :one
SELECT
  p.id, p.name, p.slug, p.gender, p.description, p.address,
  p.latitude, p.longitude, p.is_active, p.is_flash_sale,
  o.name AS owner_name, a.name AS area_name,
  coalesce(rv.rating_average, 0) AS rating_average,
  coalesce(rv.rating_count, 0) AS rating_count
FROM properties p
JOIN users o ON o.id = p.owner_id
JOIN areas a ON a.id = p.area_id
LEFT JOIN (
  SELECT property_id, round(avg(rating)::numeric, 1)::float8 AS rating_average, count(*) AS rating_count
  FROM reviews WHERE NOT is_hidden GROUP BY property_id
) rv ON rv.property_id = p.id
WHERE p.id = sqlc.arg(id)
  AND p.is_active = true AND p.moderation_status = 'approved' AND o.suspended_at IS NULL;

-- name: ListListingRooms :many
SELECT id, name, price_monthly, available, facilities, monthly_duration_options
FROM rooms
WHERE property_id = sqlc.arg(property_id)
ORDER BY id ASC;

-- name: ListListingPropertyPhotos :many
SELECT id, object_key, thumb_key, caption
FROM property_photos
WHERE property_id = sqlc.arg(property_id) AND room_id IS NULL
ORDER BY sort_order, id;

-- name: ListListingRoomPhotos :many
SELECT id, room_id, object_key, thumb_key, caption
FROM property_photos
WHERE property_id = sqlc.arg(property_id) AND room_id IS NOT NULL
ORDER BY room_id, sort_order, id;

-- name: ExistsFavorite :one
SELECT EXISTS(
  SELECT 1 FROM favorites WHERE user_id = sqlc.arg(user_id) AND property_id = sqlc.arg(property_id)
);

-- name: IncrementPropertyView :one
UPDATE properties p SET views = views + 1
FROM users o
WHERE p.id = sqlc.arg(id) AND o.id = p.owner_id
  AND p.is_active = true AND p.moderation_status = 'approved' AND o.suspended_at IS NULL
RETURNING p.views;

-- name: UpsertPropertyViewDaily :exec
INSERT INTO property_views_daily (property_id, view_date, views)
VALUES (sqlc.arg(property_id), sqlc.arg(view_date), 1)
ON CONFLICT (property_id, view_date) DO UPDATE SET views = property_views_daily.views + 1;

-- name: ListGalleryRooms :many
SELECT id, name, facilities
FROM rooms
WHERE property_id = sqlc.arg(property_id)
ORDER BY id ASC;
