-- name: OwnerPropertyExists :one
SELECT EXISTS (SELECT 1 FROM properties WHERE id = sqlc.arg(id) AND owner_id = sqlc.arg(owner_id));

-- name: LockOwnerProperty :one
SELECT id FROM properties WHERE id = sqlc.arg(id) AND owner_id = sqlc.arg(owner_id) FOR UPDATE;

-- name: CountPropertyPhotos :one
SELECT count(*) FROM property_photos WHERE property_id = sqlc.arg(property_id);

-- name: RoomBelongsToProperty :one
SELECT EXISTS (SELECT 1 FROM rooms WHERE id = sqlc.arg(id) AND property_id = sqlc.arg(property_id));

-- name: InsertPropertyPhoto :one
INSERT INTO property_photos (property_id, room_id, object_key, thumb_key, width, height, size_bytes, caption, sort_order)
VALUES (
  sqlc.arg(property_id), sqlc.narg(room_id), sqlc.arg(object_key), sqlc.arg(thumb_key),
  sqlc.arg(width), sqlc.arg(height), sqlc.arg(size_bytes), sqlc.narg(caption),
  (SELECT COALESCE(MAX(sort_order), 0) + 1 FROM property_photos WHERE property_id = sqlc.arg(property_id))
)
RETURNING id, sort_order;

-- name: LockOwnerPhoto :one
SELECT ph.id, ph.property_id, ph.room_id, ph.object_key, ph.thumb_key, ph.caption, ph.sort_order
FROM property_photos ph
JOIN properties p ON p.id = ph.property_id
WHERE ph.id = sqlc.arg(id) AND p.owner_id = sqlc.arg(owner_id)
FOR UPDATE OF ph;

-- name: UpdatePropertyPhoto :exec
UPDATE property_photos
SET caption = sqlc.narg(caption), sort_order = sqlc.arg(sort_order), updated_at = now()
WHERE id = sqlc.arg(id);

-- name: DeletePropertyPhoto :exec
DELETE FROM property_photos WHERE id = sqlc.arg(id);
