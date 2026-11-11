-- name: ListOwnerRooms :many
SELECT
  r.id, r.property_id, r.name, r.price_monthly, r.price_weekly, r.price_daily, r.available, r.capacity,
  r.monthly_duration_options, r.facilities,
  (SELECT count(*) FROM bookings b WHERE b.room_id = r.id AND b.stock_held AND b.status IN ('pending', 'paid')) AS held,
  (SELECT count(*) FROM bookings b WHERE b.room_id = r.id AND b.stock_held AND b.status = 'approved') AS occupied
FROM rooms r
JOIN properties p ON p.id = r.property_id
WHERE p.owner_id = sqlc.arg(owner_id)
  AND (sqlc.narg(property_id)::bigint IS NULL OR r.property_id = sqlc.narg(property_id))
  AND (sqlc.narg(room_id)::bigint IS NULL OR r.id = sqlc.narg(room_id))
ORDER BY r.id;

-- name: ListRoomPhotos :many
SELECT id, room_id, object_key, thumb_key, caption, sort_order
FROM property_photos
WHERE room_id = ANY(sqlc.arg(room_ids)::bigint[])
ORDER BY room_id, sort_order, id;

-- name: InsertRoom :one
INSERT INTO rooms (property_id, name, price_monthly, price_weekly, price_daily, available, capacity, facilities, monthly_duration_options)
VALUES (
  sqlc.arg(property_id), sqlc.arg(name), sqlc.arg(price_monthly), sqlc.narg(price_weekly), sqlc.narg(price_daily),
  sqlc.arg(available), sqlc.arg(capacity), sqlc.arg(facilities), sqlc.arg(monthly_duration_options)
)
RETURNING id;

-- name: LockOwnerRoom :one
SELECT r.id, r.property_id, r.name, r.price_monthly, r.price_weekly, r.price_daily, r.available, r.capacity,
  r.facilities, r.monthly_duration_options
FROM rooms r
JOIN properties p ON p.id = r.property_id
WHERE r.id = sqlc.arg(id) AND p.owner_id = sqlc.arg(owner_id)
FOR UPDATE OF r;

-- name: UpdateRoom :exec
UPDATE rooms SET
  name = sqlc.arg(name), price_monthly = sqlc.arg(price_monthly),
  price_weekly = sqlc.narg(price_weekly), price_daily = sqlc.narg(price_daily),
  available = sqlc.arg(available), capacity = sqlc.arg(capacity),
  facilities = sqlc.arg(facilities), monthly_duration_options = sqlc.arg(monthly_duration_options),
  updated_at = now()
WHERE id = sqlc.arg(id);

-- name: RoomHasBookings :one
SELECT EXISTS (SELECT 1 FROM bookings WHERE room_id = sqlc.arg(room_id));

-- name: DeleteRoom :exec
DELETE FROM rooms WHERE id = sqlc.arg(id);
