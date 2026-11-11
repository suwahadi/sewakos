-- name: GetPaymentByOrderID :one
SELECT id, invoice_id, order_id, gross_amount, status FROM payments WHERE order_id = sqlc.arg(order_id);

-- name: GetPaymentForWebhook :one
SELECT p.id AS payment_id, p.status AS payment_status, p.gross_amount,
       i.id AS invoice_id, i.type AS invoice_type, i.status AS invoice_status,
       i.number AS invoice_number, i.total AS invoice_total, i.admin_fee AS invoice_admin_fee,
       i.periods AS invoice_periods, i.period_end AS invoice_period_end,
       b.id AS booking_id, b.tenant_id, b.room_id, r.name AS room_name,
       b.status AS booking_status, b.stock_held, b.owner_deadline_at,
       pr.id AS property_id, pr.name AS property_name, pr.owner_id
FROM payments p
JOIN invoices i ON i.id = p.invoice_id
JOIN bookings b ON b.id = i.booking_id
JOIN rooms r ON r.id = b.room_id
JOIN properties pr ON pr.id = r.property_id
WHERE p.order_id = sqlc.arg(order_id)
FOR UPDATE OF b;

-- name: FinalizePayment :exec
UPDATE payments SET status = sqlc.arg(status), payment_type = sqlc.arg(payment_type),
  midtrans_transaction_id = sqlc.arg(midtrans_transaction_id), paid_at = sqlc.arg(paid_at),
  last_notification = sqlc.arg(last_notification), updated_at = now()
WHERE id = sqlc.arg(id);

-- name: MarkBookingPaid :exec
UPDATE bookings SET status = 'paid', paid_at = sqlc.arg(paid_at), owner_deadline_at = sqlc.arg(owner_deadline_at),
  updated_at = now()
WHERE id = sqlc.arg(id);

-- name: RejectBookingForStockUnavailable :exec
UPDATE bookings SET status = 'rejected', rejected_by = 'system', reject_reason = sqlc.arg(reject_reason),
  responded_at = now(), updated_at = now()
WHERE id = sqlc.arg(id);

-- name: MarkInvoicePaid :exec
UPDATE invoices SET status = 'paid', paid_at = sqlc.arg(paid_at), updated_at = now() WHERE id = sqlc.arg(id);

-- name: ExtendBookingDuration :exec
UPDATE bookings SET duration = duration + sqlc.arg(periods), end_date = sqlc.arg(end_date), updated_at = now()
WHERE id = sqlc.arg(id);

-- name: InsertRefund :one
INSERT INTO refunds (booking_id, invoice_id, amount, reason)
VALUES (sqlc.arg(booking_id), sqlc.arg(invoice_id), sqlc.arg(amount), sqlc.arg(reason))
RETURNING id;

-- name: InsertPayout :one
INSERT INTO payouts (booking_id, invoice_id, owner_id, amount)
VALUES (sqlc.arg(booking_id), sqlc.arg(invoice_id), sqlc.arg(owner_id), sqlc.arg(amount))
RETURNING id;
