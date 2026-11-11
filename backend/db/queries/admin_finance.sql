-- name: GetAdminFinanceSummary :one
SELECT
  COALESCE(SUM(amount) FILTER (WHERE type = 'payment_received'), 0)::bigint AS payments_received,
  COALESCE(SUM(amount) FILTER (WHERE type = 'admin_fee_earned'), 0)::bigint AS admin_fee_revenue,
  COALESCE(-SUM(amount) FILTER (WHERE type = 'refund_completed'), 0)::bigint AS refunds_completed,
  COALESCE(-SUM(amount) FILTER (WHERE type = 'payout_completed'), 0)::bigint AS payouts_completed
FROM ledger_entries
WHERE occurred_at >= sqlc.arg(from_time) AND occurred_at < sqlc.arg(to_time);

-- name: GetAdminFinanceOutstanding :one
SELECT
  (SELECT COALESCE(SUM(amount), 0) FROM refunds WHERE status != 'completed')::bigint AS refunds_outstanding,
  (SELECT COALESCE(SUM(amount), 0) FROM payouts WHERE status = 'pending')::bigint AS payouts_outstanding,
  (SELECT COUNT(*) FROM reconciliation_issues WHERE status = 'open')::bigint AS open_reconciliation_issues;

-- name: ListAdminLedgerEntries :many
SELECT l.id, l.occurred_at, l.type, l.amount, l.owner_id, l.booking_id, i.number AS invoice_number, l.description
FROM ledger_entries l
LEFT JOIN invoices i ON i.id = l.invoice_id
WHERE l.occurred_at >= sqlc.arg(from_time) AND l.occurred_at < sqlc.arg(to_time)
  AND (sqlc.narg(type)::varchar IS NULL OR l.type = sqlc.narg(type))
  AND (sqlc.narg(owner_id)::bigint IS NULL OR l.owner_id = sqlc.narg(owner_id))
ORDER BY l.occurred_at DESC, l.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: StreamAdminLedgerEntries :many
SELECT l.id, l.occurred_at, l.type, l.amount, l.owner_id, l.booking_id, i.number AS invoice_number, l.description
FROM ledger_entries l
LEFT JOIN invoices i ON i.id = l.invoice_id
WHERE l.occurred_at >= sqlc.arg(from_time) AND l.occurred_at < sqlc.arg(to_time)
  AND (sqlc.narg(type)::varchar IS NULL OR l.type = sqlc.narg(type))
  AND (sqlc.narg(owner_id)::bigint IS NULL OR l.owner_id = sqlc.narg(owner_id))
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
WHERE updated_at >= sqlc.arg(since) OR (status = 'pending' AND expires_at < now())
ORDER BY id
LIMIT 500;

-- name: InsertReconciliationIssue :exec
INSERT INTO reconciliation_issues (payment_id, kind, local_status, midtrans_status, local_amount, midtrans_amount)
VALUES (sqlc.arg(payment_id), sqlc.arg(kind), sqlc.arg(local_status), sqlc.narg(midtrans_status), sqlc.arg(local_amount), sqlc.narg(midtrans_amount))
ON CONFLICT (payment_id, kind) WHERE status = 'open' DO NOTHING;
