-- name: SelectDueRenewals :many
SELECT b.id AS booking_id, gs.k::int AS period_index
FROM bookings b
CROSS JOIN LATERAL generate_series(1, b.duration - 1) AS gs(k)
WHERE b.status = 'approved' AND b.rent_type = 'monthly'
  AND (b.start_date + make_interval(months => gs.k))::date < b.end_date
  AND (b.start_date + make_interval(months => gs.k))::date - 3 <= sqlc.arg(today)::date
  AND NOT EXISTS (
    SELECT 1 FROM invoices i
    WHERE i.booking_id = b.id AND i.period_start = (b.start_date + make_interval(months => gs.k))::date
  )
ORDER BY b.id, gs.k
LIMIT 100;

-- name: LockBookingForRenewal :one
SELECT b.id, b.status, b.tenant_id, b.start_date, b.end_date, b.period_price,
  (b.start_date + make_interval(months => sqlc.arg(period_index)::int))::date AS period_start,
  (b.start_date + make_interval(months => sqlc.arg(period_index)::int + 1))::date AS period_end
FROM bookings b
WHERE b.id = sqlc.arg(id)
FOR UPDATE;

-- name: InvoiceExistsForPeriod :one
SELECT EXISTS (SELECT 1 FROM invoices WHERE booking_id = sqlc.arg(booking_id) AND period_start = sqlc.arg(period_start));

-- name: SelectDueRenewalReminderIDs :many
SELECT id FROM invoices
WHERE type = 'renewal' AND status = 'unpaid' AND due_date = sqlc.arg(today)::date AND reminder_sent_at IS NULL
ORDER BY id
LIMIT 100
FOR UPDATE SKIP LOCKED;

-- name: MarkInvoiceReminderSent :one
UPDATE invoices i SET reminder_sent_at = now(), updated_at = now()
FROM bookings b
WHERE i.id = sqlc.arg(id) AND b.id = i.booking_id AND i.status = 'unpaid' AND i.reminder_sent_at IS NULL
RETURNING i.id, i.number, i.due_date, i.booking_id, b.tenant_id;

-- name: SelectOverdueRenewalIDs :many
SELECT i.id FROM invoices i
JOIN bookings b ON b.id = i.booking_id
WHERE i.type = 'renewal' AND i.status = 'unpaid' AND b.status = 'approved' AND sqlc.arg(today)::date > i.due_date + sqlc.arg(grace_days)::int
ORDER BY i.id
LIMIT 100
FOR UPDATE OF i SKIP LOCKED;

-- name: LockOverdueRenewal :one
SELECT i.id, i.number, i.booking_id, i.status AS invoice_status, i.due_date,
  b.status AS booking_status, b.tenant_id, b.room_id, b.stock_held,
  r.name AS room_name, p.name AS property_name, p.owner_id, t.name AS tenant_name
FROM invoices i
JOIN bookings b ON b.id = i.booking_id
JOIN rooms r ON r.id = b.room_id
JOIN properties p ON p.id = r.property_id
JOIN users t ON t.id = b.tenant_id
WHERE i.id = sqlc.arg(id)
FOR UPDATE OF i, b;

-- name: TerminateBooking :exec
UPDATE bookings SET status = 'terminated', ended_at = now(), stock_held = false, updated_at = now() WHERE id = sqlc.arg(id);

-- name: VoidInvoice :exec
UPDATE invoices SET status = 'void', void_reason = sqlc.arg(void_reason), updated_at = now() WHERE id = sqlc.arg(id);

-- name: SelectCompletableBookingIDs :many
SELECT id FROM bookings
WHERE status = 'approved' AND end_date <= sqlc.arg(today)::date
ORDER BY id
LIMIT 100
FOR UPDATE SKIP LOCKED;

-- name: LockBookingForCompletion :one
SELECT b.id, b.status, b.end_date, b.tenant_id, b.room_id, b.stock_held, b.deposit,
  r.name AS room_name, p.name AS property_name, p.owner_id, t.name AS tenant_name
FROM bookings b
JOIN rooms r ON r.id = b.room_id
JOIN properties p ON p.id = r.property_id
JOIN users t ON t.id = b.tenant_id
WHERE b.id = sqlc.arg(id)
FOR UPDATE OF b;

-- name: CompleteBooking :exec
UPDATE bookings SET status = 'completed', ended_at = now(), stock_held = false, updated_at = now() WHERE id = sqlc.arg(id);
