-- name: SuggestAreas :many
SELECT id, name, latitude, longitude
FROM areas
WHERE name ILIKE '%' || sqlc.arg(keyword)::text || '%'
ORDER BY id ASC
LIMIT 5;

-- name: SuggestLandings :many
SELECT p.id, p.name, p.slug, p.gender
FROM properties p
JOIN users o ON o.id = p.owner_id
WHERE p.is_active = true AND p.moderation_status = 'approved' AND o.suspended_at IS NULL
  AND p.name ILIKE '%' || sqlc.arg(keyword)::text || '%'
ORDER BY p.id ASC
LIMIT 5;

-- name: SuggestRoomTypes :many
SELECT r.id, r.property_id, r.name, p.slug
FROM rooms r
JOIN properties p ON p.id = r.property_id
JOIN users o ON o.id = p.owner_id
WHERE p.is_active = true AND p.moderation_status = 'approved' AND o.suspended_at IS NULL
  AND r.name ILIKE '%' || sqlc.arg(keyword)::text || '%'
ORDER BY r.id ASC
LIMIT 5;

-- name: SearchRooms :many
SELECT
  r.id, r.property_id, r.name AS room_name, p.name AS property_name, a.name AS area_name,
  -- Empty string means "no thumbnail"; rating_count = 0 means "no reviews yet",
  -- used instead of NULL columns because sqlc does not infer nullability
  -- through these LEFT JOIN subqueries (see DECISIONS.md).
  coalesce(thumb.object_key, '') AS thumbnail_key,
  p.gender,
  (CASE sqlc.arg(rent_type)::smallint
    WHEN 1 THEN r.price_daily
    WHEN 2 THEN r.price_weekly
    ELSE r.price_monthly
  END)::bigint AS active_price,
  r.available, p.is_flash_sale,
  coalesce(rv.rating_average, 0) AS rating_average,
  coalesce(rv.rating_count, 0) AS rating_count
FROM rooms r
JOIN properties p ON p.id = r.property_id
JOIN areas a ON a.id = p.area_id
JOIN users o ON o.id = p.owner_id
LEFT JOIN LATERAL (
  SELECT ph.object_key FROM property_photos ph
  WHERE ph.property_id = p.id AND ph.room_id IS NULL
  ORDER BY ph.sort_order, ph.id LIMIT 1
) thumb ON true
LEFT JOIN (
  SELECT property_id, round(avg(rating)::numeric, 1)::float8 AS rating_average, count(*) AS rating_count
  FROM reviews WHERE NOT is_hidden GROUP BY property_id
) rv ON rv.property_id = p.id
WHERE p.is_active = true AND p.moderation_status = 'approved' AND o.suspended_at IS NULL
  AND (CASE sqlc.arg(rent_type)::smallint WHEN 1 THEN r.price_daily WHEN 2 THEN r.price_weekly ELSE r.price_monthly END) IS NOT NULL
  AND (sqlc.narg(has_gender_filter)::bool IS NULL OR NOT sqlc.narg(has_gender_filter) OR p.gender = ANY(sqlc.arg(genders)::text[]))
  AND (sqlc.narg(price_min)::bigint IS NULL OR (CASE sqlc.arg(rent_type)::smallint WHEN 1 THEN r.price_daily WHEN 2 THEN r.price_weekly ELSE r.price_monthly END) >= sqlc.narg(price_min))
  AND (sqlc.narg(price_max)::bigint IS NULL OR (CASE sqlc.arg(rent_type)::smallint WHEN 1 THEN r.price_daily WHEN 2 THEN r.price_weekly ELSE r.price_monthly END) <= sqlc.narg(price_max))
  AND (sqlc.narg(flash_sale)::bool IS NULL OR p.is_flash_sale = sqlc.narg(flash_sale))
  AND (sqlc.narg(is_available)::bool IS NULL OR r.available >= 1)
  AND (
    sqlc.narg(area_id)::bigint IS NULL
    OR (sqlc.narg(area_type)::text = 'city' AND p.area_id = sqlc.narg(area_id))
    OR (
      sqlc.narg(area_type)::text IN ('campus', 'mrt')
      AND p.latitude BETWEEN sqlc.narg(area_lat_min)::float8 AND sqlc.narg(area_lat_max)::float8
      AND p.longitude BETWEEN sqlc.narg(area_lng_min)::float8 AND sqlc.narg(area_lng_max)::float8
      AND (
        6371 * acos(
          greatest(-1, least(1,
            cos(radians(sqlc.narg(area_lat)::float8)) * cos(radians(p.latitude)) * cos(radians(p.longitude) - radians(sqlc.narg(area_lng)::float8))
            + sin(radians(sqlc.narg(area_lat)::float8)) * sin(radians(p.latitude))
          ))
        )
      ) <= sqlc.narg(area_radius_km)::float8
    )
  )
ORDER BY
  CASE WHEN sqlc.arg(sort_field)::text = 'price' AND sqlc.arg(sort_dir)::text = '-' THEN (CASE sqlc.arg(rent_type)::smallint WHEN 1 THEN r.price_daily WHEN 2 THEN r.price_weekly ELSE r.price_monthly END) END DESC,
  CASE WHEN sqlc.arg(sort_field)::text = 'price' AND sqlc.arg(sort_dir)::text = '+' THEN (CASE sqlc.arg(rent_type)::smallint WHEN 1 THEN r.price_daily WHEN 2 THEN r.price_weekly ELSE r.price_monthly END) END ASC,
  CASE WHEN sqlc.arg(sort_field)::text = 'availability' AND sqlc.arg(sort_dir)::text = '-' THEN r.available END DESC,
  CASE WHEN sqlc.arg(sort_field)::text = 'availability' AND sqlc.arg(sort_dir)::text = '+' THEN r.available END ASC,
  r.id ASC
LIMIT sqlc.arg(result_limit)::int OFFSET sqlc.arg(result_offset)::int;

-- name: CountSearchRooms :one
SELECT count(*)::bigint AS total, count(DISTINCT p.id)::bigint AS property_total
FROM rooms r
JOIN properties p ON p.id = r.property_id
JOIN users o ON o.id = p.owner_id
WHERE p.is_active = true AND p.moderation_status = 'approved' AND o.suspended_at IS NULL
  AND (CASE sqlc.arg(rent_type)::smallint WHEN 1 THEN r.price_daily WHEN 2 THEN r.price_weekly ELSE r.price_monthly END) IS NOT NULL
  AND (sqlc.narg(has_gender_filter)::bool IS NULL OR NOT sqlc.narg(has_gender_filter) OR p.gender = ANY(sqlc.arg(genders)::text[]))
  AND (sqlc.narg(price_min)::bigint IS NULL OR (CASE sqlc.arg(rent_type)::smallint WHEN 1 THEN r.price_daily WHEN 2 THEN r.price_weekly ELSE r.price_monthly END) >= sqlc.narg(price_min))
  AND (sqlc.narg(price_max)::bigint IS NULL OR (CASE sqlc.arg(rent_type)::smallint WHEN 1 THEN r.price_daily WHEN 2 THEN r.price_weekly ELSE r.price_monthly END) <= sqlc.narg(price_max))
  AND (sqlc.narg(flash_sale)::bool IS NULL OR p.is_flash_sale = sqlc.narg(flash_sale))
  AND (sqlc.narg(is_available)::bool IS NULL OR r.available >= 1)
  AND (
    sqlc.narg(area_id)::bigint IS NULL
    OR (sqlc.narg(area_type)::text = 'city' AND p.area_id = sqlc.narg(area_id))
    OR (
      sqlc.narg(area_type)::text IN ('campus', 'mrt')
      AND p.latitude BETWEEN sqlc.narg(area_lat_min)::float8 AND sqlc.narg(area_lat_max)::float8
      AND p.longitude BETWEEN sqlc.narg(area_lng_min)::float8 AND sqlc.narg(area_lng_max)::float8
      AND (
        6371 * acos(
          greatest(-1, least(1,
            cos(radians(sqlc.narg(area_lat)::float8)) * cos(radians(p.latitude)) * cos(radians(p.longitude) - radians(sqlc.narg(area_lng)::float8))
            + sin(radians(sqlc.narg(area_lat)::float8)) * sin(radians(p.latitude))
          ))
        )
      ) <= sqlc.narg(area_radius_km)::float8
    )
  );
