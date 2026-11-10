-- name: InsertLedgerEntry :one
INSERT INTO ledger_entries (
  type, amount, owner_id, tenant_id, booking_id, invoice_id, payment_id, refund_id,
  payout_id, description, created_by
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING id;
