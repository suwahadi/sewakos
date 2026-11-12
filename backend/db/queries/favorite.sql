-- name: ExistsProperty :one
SELECT EXISTS (SELECT 1 FROM properties WHERE id = sqlc.arg(id));

-- name: AddFavorite :exec
INSERT INTO favorites (user_id, property_id)
VALUES (sqlc.arg(user_id), sqlc.arg(property_id))
ON CONFLICT DO NOTHING;

-- name: RemoveFavorite :exec
DELETE FROM favorites WHERE user_id = sqlc.arg(user_id) AND property_id = sqlc.arg(property_id);

-- name: ListFavorites :many
SELECT
  p.id, p.name, p.gender, a.name AS area_name,
  coalesce(thumb.thumb_key, '') AS thumbnail_key,
  coalesce(pr.min_price_monthly, 0)::bigint AS min_price_monthly,
  (pr.min_price_monthly IS NOT NULL)::bool AS has_monthly_price,
  coalesce(rv.rating_average, 0)::float8 AS rating_average,
  coalesce(rv.rating_count, 0)::bigint AS rating_count,
  (p.is_active = true AND p.moderation_status = 'approved' AND o.suspended_at IS NULL)::bool AS is_available
FROM favorites f
JOIN properties p ON p.id = f.property_id
JOIN users o ON o.id = p.owner_id
JOIN areas a ON a.id = p.area_id
LEFT JOIN LATERAL (
  SELECT ph.thumb_key FROM property_photos ph
  WHERE ph.property_id = p.id AND ph.room_id IS NULL
  ORDER BY ph.sort_order, ph.id LIMIT 1
) thumb ON true
LEFT JOIN LATERAL (
  SELECT min(r.price_monthly) AS min_price_monthly FROM rooms r WHERE r.property_id = p.id
) pr ON true
LEFT JOIN LATERAL (
  SELECT round(avg(rating)::numeric, 1)::float8 AS rating_average, count(*) AS rating_count
  FROM reviews WHERE property_id = p.id AND NOT is_hidden
) rv ON true
WHERE f.user_id = sqlc.arg(user_id)
ORDER BY f.created_at DESC, p.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);
