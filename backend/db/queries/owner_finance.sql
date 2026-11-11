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
