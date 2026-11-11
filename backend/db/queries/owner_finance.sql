-- name: ListInvoicesForOwner :many
SELECT i.id, i.number, i.booking_id, i.type, i.status,
       i.period_start, i.period_end, i.rent_amount, i.discount, i.deposit, i.admin_fee, i.total,
       i.due_date, i.paid_at, i.issued_at
FROM invoices i
JOIN bookings b ON b.id = i.booking_id
JOIN rooms r ON r.id = b.room_id
JOIN properties p ON p.id = r.property_id
WHERE p.owner_id = sqlc.arg(owner_id) AND b.paid_at IS NOT NULL
  AND (sqlc.narg(booking_id)::bigint IS NULL OR i.booking_id = sqlc.narg(booking_id))
  AND (sqlc.narg(status)::text IS NULL OR i.status = sqlc.narg(status))
ORDER BY i.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: GetOwnerBankAccount :one
SELECT bank_name, account_number, account_name FROM owner_bank_accounts WHERE user_id = sqlc.arg(user_id);

-- name: UpsertOwnerBankAccount :exec
INSERT INTO owner_bank_accounts (user_id, bank_name, account_number, account_name)
VALUES (sqlc.arg(user_id), sqlc.arg(bank_name), sqlc.arg(account_number), sqlc.arg(account_name))
ON CONFLICT (user_id) DO UPDATE SET
  bank_name = EXCLUDED.bank_name, account_number = EXCLUDED.account_number,
  account_name = EXCLUDED.account_name, updated_at = now();

-- name: ListOwnerPayouts :many
SELECT id, booking_id, amount, status, transfer_reference, completed_at, created_at
FROM payouts
WHERE owner_id = sqlc.arg(owner_id)
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status))
ORDER BY id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: GetOwnerFinanceSummary :one
SELECT
  COALESCE(SUM(amount) FILTER (WHERE type = 'owner_earning'), 0)::bigint AS earnings,
  COALESCE(-SUM(amount) FILTER (WHERE type = 'payout_completed'), 0)::bigint AS payouts_completed,
  COUNT(DISTINCT invoice_id) FILTER (WHERE type = 'owner_earning')::bigint AS paid_invoices
FROM ledger_entries
WHERE owner_id = sqlc.arg(owner_id) AND occurred_at >= sqlc.arg(from_time) AND occurred_at < sqlc.arg(to_time);

-- name: GetOwnerPendingBalance :one
SELECT COALESCE(SUM(amount) FILTER (WHERE type IN ('owner_earning', 'payout_completed')), 0)::bigint AS pending_balance
FROM ledger_entries WHERE owner_id = sqlc.arg(owner_id);

-- name: GetOwnerDepositsHeld :one
SELECT COALESCE(SUM(b.deposit), 0)::bigint AS deposits_held
FROM bookings b
JOIN rooms r ON r.id = b.room_id
JOIN properties p ON p.id = r.property_id
WHERE p.owner_id = sqlc.arg(owner_id) AND b.status IN ('approved', 'completed', 'terminated') AND b.deposit_returned_at IS NULL;

-- name: ListOwnerLedgerEntries :many
SELECT l.id, l.occurred_at, l.type, l.amount, l.booking_id, i.number AS invoice_number, l.description
FROM ledger_entries l
LEFT JOIN invoices i ON i.id = l.invoice_id
WHERE l.owner_id = sqlc.arg(owner_id) AND l.occurred_at >= sqlc.arg(from_time) AND l.occurred_at < sqlc.arg(to_time)
  AND (sqlc.narg(type)::varchar IS NULL OR l.type = sqlc.narg(type))
ORDER BY l.occurred_at DESC, l.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: StreamOwnerLedgerEntries :many
SELECT l.id, l.occurred_at, l.type, l.amount, l.booking_id, i.number AS invoice_number, l.description
FROM ledger_entries l
LEFT JOIN invoices i ON i.id = l.invoice_id
WHERE l.owner_id = sqlc.arg(owner_id) AND l.occurred_at >= sqlc.arg(from_time) AND l.occurred_at < sqlc.arg(to_time)
  AND (sqlc.narg(type)::varchar IS NULL OR l.type = sqlc.narg(type))
ORDER BY l.occurred_at DESC, l.id DESC;

-- name: GetOwnerStatsBookingCounts :one
SELECT
  COUNT(*) FILTER (WHERE b.paid_at >= sqlc.arg(from_time) AND b.paid_at < sqlc.arg(to_time))::bigint AS bookings_paid,
  COUNT(*) FILTER (WHERE b.status = 'approved' AND b.responded_at >= sqlc.arg(from_time) AND b.responded_at < sqlc.arg(to_time))::bigint AS bookings_approved,
  COUNT(*) FILTER (WHERE b.status = 'rejected' AND b.responded_at >= sqlc.arg(from_time) AND b.responded_at < sqlc.arg(to_time))::bigint AS bookings_rejected
FROM bookings b
JOIN rooms r ON r.id = b.room_id
JOIN properties p ON p.id = r.property_id
WHERE p.owner_id = sqlc.arg(owner_id) AND (sqlc.narg(property_id)::bigint IS NULL OR p.id = sqlc.narg(property_id));

-- name: GetOwnerStatsViews :one
SELECT COALESCE(SUM(v.views), 0)::bigint AS views
FROM property_views_daily v
JOIN properties p ON p.id = v.property_id
WHERE p.owner_id = sqlc.arg(owner_id) AND (sqlc.narg(property_id)::bigint IS NULL OR p.id = sqlc.narg(property_id))
  AND v.view_date >= sqlc.arg(from_date) AND v.view_date <= sqlc.arg(to_date);

-- name: GetOwnerStatsEarnings :one
SELECT COALESCE(SUM(l.amount), 0)::bigint AS earnings
FROM ledger_entries l
JOIN bookings b ON b.id = l.booking_id
JOIN rooms r ON r.id = b.room_id
WHERE l.owner_id = sqlc.arg(owner_id) AND l.type = 'owner_earning'
  AND l.occurred_at >= sqlc.arg(from_time) AND l.occurred_at < sqlc.arg(to_time)
  AND (sqlc.narg(property_id)::bigint IS NULL OR r.property_id = sqlc.narg(property_id));

-- name: GetOwnerOccupancy :one
SELECT
  COALESCE(SUM(r.available), 0)::bigint AS available,
  COALESCE(SUM((SELECT count(*) FROM bookings b WHERE b.room_id = r.id AND b.stock_held AND b.status IN ('pending', 'paid'))), 0)::bigint AS held,
  COALESCE(SUM((SELECT count(*) FROM bookings b WHERE b.room_id = r.id AND b.stock_held AND b.status = 'approved')), 0)::bigint AS occupied
FROM rooms r
JOIN properties p ON p.id = r.property_id
WHERE p.owner_id = sqlc.arg(owner_id) AND (sqlc.narg(property_id)::bigint IS NULL OR p.id = sqlc.narg(property_id));

-- name: GetOwnerRating :one
SELECT COALESCE(AVG(rv.rating), 0)::float8 AS average, COUNT(*)::bigint AS count
FROM reviews rv
JOIN properties p ON p.id = rv.property_id
WHERE p.owner_id = sqlc.arg(owner_id) AND NOT rv.is_hidden
  AND (sqlc.narg(property_id)::bigint IS NULL OR p.id = sqlc.narg(property_id));
