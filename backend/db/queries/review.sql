-- name: LockBookingForReview :one
SELECT b.id, b.tenant_id, b.status, b.start_date, r.name AS room_name,
       p.id AS property_id, p.name AS property_name, p.owner_id,
       EXISTS (SELECT 1 FROM reviews rv WHERE rv.booking_id = b.id)::bool AS reviewed
FROM bookings b
JOIN rooms r ON r.id = b.room_id
JOIN properties p ON p.id = r.property_id
WHERE b.id = sqlc.arg(id)
FOR UPDATE OF b;

-- name: InsertReview :one
INSERT INTO reviews (booking_id, property_id, tenant_id, rating, comment)
VALUES (sqlc.arg(booking_id), sqlc.arg(property_id), sqlc.arg(tenant_id), sqlc.arg(rating), sqlc.narg(comment))
RETURNING id, created_at;

-- name: ExistsListedProperty :one
SELECT EXISTS (
  SELECT 1 FROM properties p
  JOIN users o ON o.id = p.owner_id
  WHERE p.id = sqlc.arg(id)
    AND p.is_active = true AND p.moderation_status = 'approved' AND o.suspended_at IS NULL
);

-- name: GetPropertyRating :one
SELECT coalesce(round(avg(rating)::numeric, 1), 0)::float8 AS average, count(*)::bigint AS count
FROM reviews
WHERE property_id = sqlc.arg(property_id) AND NOT is_hidden;

-- name: ListPropertyReviews :many
SELECT rv.id, rv.rating, rv.comment, u.name AS tenant_name, r.name AS room_name, rv.created_at
FROM reviews rv
JOIN users u ON u.id = rv.tenant_id
JOIN bookings b ON b.id = rv.booking_id
JOIN rooms r ON r.id = b.room_id
WHERE rv.property_id = sqlc.arg(property_id) AND NOT rv.is_hidden
ORDER BY rv.created_at DESC, rv.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: ListOwnerReviews :many
SELECT rv.id, rv.property_id, rv.rating, rv.comment, rv.is_hidden, u.name AS tenant_name, r.name AS room_name, rv.created_at
FROM reviews rv
JOIN properties p ON p.id = rv.property_id
JOIN users u ON u.id = rv.tenant_id
JOIN bookings b ON b.id = rv.booking_id
JOIN rooms r ON r.id = b.room_id
WHERE p.owner_id = sqlc.arg(owner_id)
  AND (sqlc.narg(property_id)::bigint IS NULL OR rv.property_id = sqlc.narg(property_id))
ORDER BY rv.created_at DESC, rv.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: AdminListReviews :many
SELECT rv.id, rv.property_id, rv.rating, rv.comment, rv.is_hidden, rv.hidden_reason,
       u.name AS tenant_name, r.name AS room_name, rv.created_at
FROM reviews rv
JOIN users u ON u.id = rv.tenant_id
JOIN bookings b ON b.id = rv.booking_id
JOIN rooms r ON r.id = b.room_id
WHERE (sqlc.narg(is_hidden)::bool IS NULL OR rv.is_hidden = sqlc.narg(is_hidden))
  AND (sqlc.narg(property_id)::bigint IS NULL OR rv.property_id = sqlc.narg(property_id))
ORDER BY rv.created_at DESC, rv.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: LockReviewForModeration :one
SELECT rv.id, rv.property_id, rv.is_hidden, rv.hidden_reason, p.owner_id
FROM reviews rv
JOIN properties p ON p.id = rv.property_id
WHERE rv.id = sqlc.arg(id)
FOR UPDATE OF rv;

-- name: SetReviewHidden :exec
UPDATE reviews SET is_hidden = sqlc.arg(is_hidden), hidden_reason = sqlc.narg(hidden_reason), updated_at = now()
WHERE id = sqlc.arg(id);
