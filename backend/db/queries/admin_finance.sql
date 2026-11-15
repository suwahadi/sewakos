-- name: GetAdminFinanceSummary :one
SELECT
  COALESCE(SUM(l.amount) FILTER (WHERE l.type = 'payment_received'), 0)::bigint AS payments_received,
  COALESCE(SUM(l.amount) FILTER (WHERE l.type = 'admin_fee_earned'), 0)::bigint AS admin_fee_revenue,
  COALESCE(-SUM(l.amount) FILTER (WHERE l.type = 'refund_completed'), 0)::bigint AS refunds_completed,
  COALESCE(-SUM(l.amount) FILTER (WHERE l.type = 'payout_completed'), 0)::bigint AS payouts_completed
FROM ledger_entries l
LEFT JOIN bookings b ON b.id = l.booking_id
LEFT JOIN rooms r ON r.id = b.room_id
LEFT JOIN properties p ON p.id = r.property_id
WHERE l.occurred_at >= sqlc.arg(from_time) AND l.occurred_at < sqlc.arg(to_time)
  AND (sqlc.narg(owner_id)::bigint IS NULL OR COALESCE(l.owner_id, p.owner_id) = sqlc.narg(owner_id));

-- name: GetAdminFinanceOutstanding :one
SELECT
  (SELECT COALESCE(SUM(rf.amount), 0)
   FROM refunds rf
   JOIN bookings b ON b.id = rf.booking_id
   JOIN rooms r ON r.id = b.room_id
   JOIN properties p ON p.id = r.property_id
   WHERE rf.status != 'completed' AND (sqlc.narg(owner_id)::bigint IS NULL OR p.owner_id = sqlc.narg(owner_id)))::bigint AS refunds_outstanding,
  (SELECT COALESCE(SUM(po.amount), 0)
   FROM payouts po
   WHERE po.status = 'pending' AND (sqlc.narg(owner_id)::bigint IS NULL OR po.owner_id = sqlc.narg(owner_id)))::bigint AS payouts_outstanding,
  (SELECT COUNT(*)
   FROM reconciliation_issues ri
   JOIN payments pm ON pm.id = ri.payment_id
   JOIN invoices i ON i.id = pm.invoice_id
   JOIN bookings b ON b.id = i.booking_id
   JOIN rooms r ON r.id = b.room_id
   JOIN properties p ON p.id = r.property_id
   WHERE ri.status = 'open' AND (sqlc.narg(owner_id)::bigint IS NULL OR p.owner_id = sqlc.narg(owner_id)))::bigint AS open_reconciliation_issues;

-- name: ListAdminLedgerEntries :many
SELECT l.id, l.occurred_at, l.type, l.amount, o.id AS owner_id, o.name AS owner_name,
  l.booking_id, i.number AS invoice_number, l.description
FROM ledger_entries l
LEFT JOIN invoices i ON i.id = l.invoice_id
LEFT JOIN bookings b ON b.id = l.booking_id
LEFT JOIN rooms r ON r.id = b.room_id
LEFT JOIN properties p ON p.id = r.property_id
LEFT JOIN users o ON o.id = COALESCE(l.owner_id, p.owner_id)
WHERE l.occurred_at >= sqlc.arg(from_time) AND l.occurred_at < sqlc.arg(to_time)
  AND (sqlc.narg(type)::varchar IS NULL OR l.type = sqlc.narg(type))
  AND (sqlc.narg(owner_id)::bigint IS NULL OR COALESCE(l.owner_id, p.owner_id) = sqlc.narg(owner_id))
ORDER BY l.occurred_at DESC, l.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: StreamAdminLedgerEntries :many
SELECT l.id, l.occurred_at, l.type, l.amount, o.id AS owner_id, o.name AS owner_name,
  l.booking_id, i.number AS invoice_number, l.description
FROM ledger_entries l
LEFT JOIN invoices i ON i.id = l.invoice_id
LEFT JOIN bookings b ON b.id = l.booking_id
LEFT JOIN rooms r ON r.id = b.room_id
LEFT JOIN properties p ON p.id = r.property_id
LEFT JOIN users o ON o.id = COALESCE(l.owner_id, p.owner_id)
WHERE l.occurred_at >= sqlc.arg(from_time) AND l.occurred_at < sqlc.arg(to_time)
  AND (sqlc.narg(type)::varchar IS NULL OR l.type = sqlc.narg(type))
  AND (sqlc.narg(owner_id)::bigint IS NULL OR COALESCE(l.owner_id, p.owner_id) = sqlc.narg(owner_id))
ORDER BY l.occurred_at DESC, l.id DESC;

-- name: ListReconciliationIssues :many
SELECT ri.id, ri.payment_id, p.order_id, ri.kind, ri.local_status, ri.midtrans_status, ri.local_amount, ri.midtrans_amount,
       ri.status, ri.note, ri.detected_at
FROM reconciliation_issues ri
JOIN payments p ON p.id = ri.payment_id
WHERE (sqlc.narg(status)::text IS NULL OR ri.status = sqlc.narg(status))
ORDER BY ri.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: LockReconciliationIssue :one
SELECT id, status FROM reconciliation_issues WHERE id = sqlc.arg(id) FOR UPDATE;

-- name: ResolveReconciliationIssue :exec
UPDATE reconciliation_issues SET status = 'resolved', note = sqlc.arg(note), resolved_by = sqlc.arg(resolved_by), resolved_at = now(), updated_at = now()
WHERE id = sqlc.arg(id);

-- name: SelectPaymentsForReconciliation :many
SELECT id, order_id, status, gross_amount
FROM payments
WHERE id > sqlc.arg(after_id)
  AND (updated_at >= sqlc.arg(since) OR (status = 'pending' AND expires_at < now()))
ORDER BY id
LIMIT 500;

-- name: InsertReconciliationIssue :exec
INSERT INTO reconciliation_issues (payment_id, kind, local_status, midtrans_status, local_amount, midtrans_amount)
VALUES (sqlc.arg(payment_id), sqlc.arg(kind), sqlc.arg(local_status), sqlc.narg(midtrans_status), sqlc.arg(local_amount), sqlc.narg(midtrans_amount))
ON CONFLICT (payment_id, kind) WHERE status = 'open' DO NOTHING;
