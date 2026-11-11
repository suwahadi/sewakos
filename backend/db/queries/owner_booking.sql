-- name: ListOwnerBookings :many
SELECT
  b.id, b.status, b.rent_type, b.start_date, b.duration, b.end_date,
  t.id AS tenant_id, t.name AS tenant_name, t.phone AS tenant_phone,
  p.id AS property_id, p.name AS property_name, r.id AS room_id, r.name AS room_name,
  b.price, b.deposit, (i.total - i.admin_fee)::bigint AS payout_amount,
  b.paid_at, b.owner_deadline_at, b.reject_reason, b.deposit_returned_at, b.created_at
FROM bookings b
JOIN rooms r ON r.id = b.room_id
JOIN properties p ON p.id = r.property_id
JOIN users t ON t.id = b.tenant_id
JOIN invoices i ON i.booking_id = b.id AND i.type = 'initial'
WHERE p.owner_id = sqlc.arg(owner_id) AND b.paid_at IS NOT NULL
  AND (sqlc.narg(booking_id)::bigint IS NULL OR b.id = sqlc.narg(booking_id))
  AND (sqlc.narg(status)::text IS NULL OR b.status = sqlc.narg(status))
  AND (sqlc.narg(property_id)::bigint IS NULL OR p.id = sqlc.narg(property_id))
ORDER BY (b.status = 'paid') DESC, CASE WHEN b.status = 'paid' THEN b.owner_deadline_at END ASC, b.created_at DESC, b.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: LockPaidBooking :one
SELECT
  b.id, b.status, b.tenant_id, b.room_id, b.stock_held, b.start_date, b.owner_deadline_at,
  r.name AS room_name, p.id AS property_id, p.name AS property_name, p.owner_id,
  i.id AS invoice_id, i.number AS invoice_number, i.total AS invoice_total, i.admin_fee AS invoice_admin_fee
FROM bookings b
JOIN rooms r ON r.id = b.room_id
JOIN properties p ON p.id = r.property_id
JOIN invoices i ON i.booking_id = b.id AND i.type = 'initial'
WHERE b.id = sqlc.arg(id) AND b.paid_at IS NOT NULL
  AND (sqlc.narg(owner_id)::bigint IS NULL OR p.owner_id = sqlc.narg(owner_id))
FOR UPDATE OF b;

-- name: ApproveBooking :exec
UPDATE bookings SET status = 'approved', responded_at = now(), updated_at = now() WHERE id = sqlc.arg(id);

-- name: RejectPaidBooking :exec
UPDATE bookings SET status = 'rejected', rejected_by = sqlc.arg(rejected_by), reject_reason = sqlc.narg(reject_reason),
  responded_at = now(), stock_held = false, updated_at = now()
WHERE id = sqlc.arg(id);

-- name: SelectOwnerTimeoutBookingIDs :many
SELECT id FROM bookings
WHERE status = 'paid' AND owner_deadline_at < now()
ORDER BY id
LIMIT 100
FOR UPDATE SKIP LOCKED;

-- name: SelectOwnerReminderBookingIDs :many
SELECT id FROM bookings
WHERE status = 'paid' AND reminder_sent_at IS NULL
  AND owner_deadline_at - make_interval(mins => sqlc.arg(before_minutes)::int) <= now() AND now() < owner_deadline_at
ORDER BY id
LIMIT 100
FOR UPDATE SKIP LOCKED;

-- name: MarkOwnerReminderSent :one
UPDATE bookings b SET reminder_sent_at = now(), updated_at = now()
FROM rooms r, properties p
WHERE b.id = sqlc.arg(id) AND r.id = b.room_id AND p.id = r.property_id
  AND b.status = 'paid' AND b.reminder_sent_at IS NULL AND now() < b.owner_deadline_at
RETURNING b.id, b.owner_deadline_at, p.owner_id;
