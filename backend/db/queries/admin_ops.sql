-- name: AdminListProperties :many
SELECT
  p.id, p.name, p.slug, p.gender, p.description, p.address, p.latitude, p.longitude,
  p.area_id, a.name AS area_name, p.is_active, p.moderation_status, p.moderation_note,
  p.is_flash_sale, p.flash_sale_discount, p.views,
  (SELECT count(*) FROM rooms r WHERE r.property_id = p.id) AS rooms_count,
  coalesce(rv.rating_average, 0)::float8 AS rating_average,
  coalesce(rv.rating_count, 0)::bigint AS rating_count,
  p.created_at, o.id AS owner_id, o.name AS owner_name, o.phone AS owner_phone
FROM properties p
JOIN areas a ON a.id = p.area_id
JOIN users o ON o.id = p.owner_id
LEFT JOIN (
  SELECT property_id, round(avg(rating)::numeric, 1)::float8 AS rating_average, count(*) AS rating_count
  FROM reviews WHERE NOT is_hidden GROUP BY property_id
) rv ON rv.property_id = p.id
WHERE (sqlc.narg(property_id)::bigint IS NULL OR p.id = sqlc.narg(property_id))
  AND (sqlc.narg(moderation_status)::text IS NULL OR p.moderation_status = sqlc.narg(moderation_status))
  AND (sqlc.narg(q)::text IS NULL OR p.name ILIKE '%' || sqlc.narg(q) || '%')
ORDER BY p.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: AdminLockProperty :one
SELECT id, name, owner_id, moderation_status FROM properties WHERE id = sqlc.arg(id) FOR UPDATE;

-- name: AdminSetModeration :exec
UPDATE properties SET moderation_status = sqlc.arg(moderation_status), moderation_note = sqlc.narg(moderation_note),
  moderated_at = now(), updated_at = now()
WHERE id = sqlc.arg(id);

-- name: AdminListBookings :many
SELECT
  b.id, b.status, b.rent_type, b.start_date, b.duration, b.end_date,
  t.id AS tenant_id, t.name AS tenant_name, t.phone AS tenant_phone,
  p.id AS property_id, p.name AS property_name, r.id AS room_id, r.name AS room_name,
  b.price, b.deposit, b.admin_fee, (i.total - i.admin_fee)::bigint AS payout_amount, i.id AS invoice_id,
  b.paid_at, b.owner_deadline_at, b.reject_reason, b.deposit_returned_at, b.created_at
FROM bookings b
JOIN rooms r ON r.id = b.room_id
JOIN properties p ON p.id = r.property_id
JOIN users t ON t.id = b.tenant_id
JOIN invoices i ON i.booking_id = b.id AND i.type = 'initial'
WHERE (sqlc.narg(booking_id)::bigint IS NULL OR b.id = sqlc.narg(booking_id))
  AND (sqlc.narg(status)::text IS NULL OR b.status = sqlc.narg(status))
ORDER BY b.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: GetPayoutForInvoice :one
SELECT id, booking_id, amount, status, transfer_reference, completed_at, created_at
FROM payouts WHERE invoice_id = sqlc.arg(invoice_id);

-- name: AdminListInvoices :many
SELECT i.id, i.number, i.booking_id, i.type, i.status,
       i.period_start, i.period_end, i.rent_amount, i.discount, i.deposit, i.admin_fee, i.total,
       i.due_date, i.paid_at, i.issued_at
FROM invoices i
WHERE (sqlc.narg(status)::text IS NULL OR i.status = sqlc.narg(status))
  AND (sqlc.narg(type)::text IS NULL OR i.type = sqlc.narg(type))
  AND (sqlc.narg(booking_id)::bigint IS NULL OR i.booking_id = sqlc.narg(booking_id))
ORDER BY i.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: AdminListRefunds :many
SELECT rf.id, rf.booking_id, t.id AS tenant_id, t.name AS tenant_name, rf.amount, rf.reason, rf.status,
  rf.bank_name, rf.account_number, rf.account_name, rf.transfer_reference, rf.note, rf.completed_at, rf.created_at
FROM refunds rf
JOIN bookings b ON b.id = rf.booking_id
JOIN users t ON t.id = b.tenant_id
WHERE (sqlc.narg(refund_id)::bigint IS NULL OR rf.id = sqlc.narg(refund_id))
  AND (sqlc.narg(status)::text IS NULL OR rf.status = sqlc.narg(status))
ORDER BY rf.created_at ASC, rf.id ASC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: AdminLockRefund :one
SELECT rf.id, rf.booking_id, rf.invoice_id, rf.amount, rf.status, b.tenant_id
FROM refunds rf JOIN bookings b ON b.id = rf.booking_id
WHERE rf.id = sqlc.arg(id)
FOR UPDATE OF rf;

-- name: CompleteRefund :exec
UPDATE refunds SET status = 'completed', transfer_reference = sqlc.arg(transfer_reference), note = sqlc.narg(note),
  completed_by = sqlc.arg(completed_by), completed_at = now(), updated_at = now()
WHERE id = sqlc.arg(id);

-- name: AdminListPayouts :many
SELECT po.id, po.booking_id, o.id AS owner_id, o.name AS owner_name, po.amount, po.status,
  po.bank_name, po.account_number, po.account_name,
  ba.bank_name AS current_bank_name, ba.account_number AS current_account_number, ba.account_name AS current_account_name,
  po.transfer_reference, po.completed_at, po.created_at
FROM payouts po
JOIN users o ON o.id = po.owner_id
LEFT JOIN owner_bank_accounts ba ON ba.user_id = po.owner_id AND po.status = 'pending'
WHERE (sqlc.narg(payout_id)::bigint IS NULL OR po.id = sqlc.narg(payout_id))
  AND (sqlc.narg(status)::text IS NULL OR po.status = sqlc.narg(status))
ORDER BY po.created_at ASC, po.id ASC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: AdminLockPayout :one
SELECT id, booking_id, invoice_id, owner_id, amount, status FROM payouts WHERE id = sqlc.arg(id) FOR UPDATE;

-- name: CompletePayout :exec
UPDATE payouts SET status = 'completed', bank_name = sqlc.arg(bank_name), account_number = sqlc.arg(account_number),
  account_name = sqlc.arg(account_name), transfer_reference = sqlc.arg(transfer_reference), note = sqlc.narg(note),
  completed_by = sqlc.arg(completed_by), completed_at = now(), updated_at = now()
WHERE id = sqlc.arg(id);

-- name: UpsertSetting :exec
INSERT INTO settings (key, value) VALUES (sqlc.arg(key), sqlc.arg(value))
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now();

-- name: BroadcastNotification :execrows
INSERT INTO user_notifications (user_id, category, title, body)
SELECT id, 'promo', sqlc.arg(title), sqlc.arg(body)
FROM users
WHERE suspended_at IS NULL
  AND role <> 'admin'
  AND (sqlc.arg(target)::text = 'all' OR role = sqlc.arg(target)::text);
