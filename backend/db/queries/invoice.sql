-- name: ListInvoicesForTenant :many
SELECT i.id, i.number, i.booking_id, i.type, i.status,
       i.period_start, i.period_end, i.rent_amount, i.discount, i.deposit, i.admin_fee, i.total,
       i.due_date, i.paid_at, i.issued_at
FROM invoices i
JOIN bookings b ON b.id = i.booking_id
WHERE b.tenant_id = sqlc.arg(tenant_id)
  AND (sqlc.narg(booking_id)::bigint IS NULL OR i.booking_id = sqlc.narg(booking_id))
  AND (sqlc.narg(status)::varchar IS NULL OR i.status = sqlc.narg(status))
ORDER BY i.id DESC
LIMIT sqlc.arg(result_limit) OFFSET sqlc.arg(result_offset);

-- name: GetInvoiceForTenant :one
SELECT i.id, i.number, i.booking_id, i.type, i.status,
       i.period_start, i.period_end, i.rent_amount, i.discount, i.deposit, i.admin_fee, i.total,
       i.due_date, i.paid_at, i.issued_at, b.tenant_id
FROM invoices i
JOIN bookings b ON b.id = i.booking_id
WHERE i.id = sqlc.arg(id);

-- name: SelectInvoiceForCheckout :one
SELECT i.id, i.type, i.status, i.total, i.admin_fee, i.number,
       b.id AS booking_id, b.tenant_id, b.room_id, b.rent_type, b.status AS booking_status, b.stock_held,
       r.name AS room_name
FROM invoices i
JOIN bookings b ON b.id = i.booking_id
JOIN rooms r ON r.id = b.room_id
WHERE i.id = sqlc.arg(id)
FOR UPDATE OF b;
