-- name: GetRoomForDraftPrice :one
SELECT r.name, r.price_monthly, r.price_weekly, r.price_daily,
       p.is_flash_sale, p.flash_sale_discount
FROM rooms r
JOIN properties p ON p.id = r.property_id
WHERE r.id = sqlc.arg(id);

-- name: GetRoomForSubmit :one
SELECT r.id, r.property_id, r.name, r.price_monthly, r.price_weekly, r.price_daily,
       r.available, r.monthly_duration_options,
       p.is_active, p.moderation_status, p.is_flash_sale, p.flash_sale_discount,
       (o.suspended_at IS NOT NULL)::bool AS owner_suspended
FROM rooms r
JOIN properties p ON p.id = r.property_id
JOIN users o ON o.id = p.owner_id
WHERE r.id = sqlc.arg(id);

-- name: InsertBooking :one
INSERT INTO bookings (
  tenant_id, room_id, rent_type, start_date, duration, end_date,
  period_price, flash_sale_discount, price, deposit, admin_fee, expires_at
) VALUES (
  sqlc.arg(tenant_id), sqlc.arg(room_id), sqlc.arg(rent_type), sqlc.arg(start_date),
  sqlc.arg(duration), sqlc.arg(end_date), sqlc.arg(period_price), sqlc.arg(flash_sale_discount),
  sqlc.arg(price), sqlc.arg(deposit), sqlc.arg(admin_fee), sqlc.arg(expires_at)
)
RETURNING id, expires_at;

-- name: InsertInvoice :one
INSERT INTO invoices (
  booking_id, number, type, period_start, period_end, periods,
  rent_amount, discount, deposit, admin_fee, total, due_date, issued_at
) VALUES (
  sqlc.arg(booking_id), '', sqlc.arg(type), sqlc.arg(period_start), sqlc.arg(period_end), sqlc.arg(periods),
  sqlc.arg(rent_amount), sqlc.arg(discount), sqlc.arg(deposit), sqlc.arg(admin_fee), sqlc.arg(total),
  sqlc.arg(due_date), sqlc.arg(issued_at)
)
RETURNING id;

-- name: SetInvoiceNumber :exec
UPDATE invoices SET number = sqlc.arg(number), updated_at = now() WHERE id = sqlc.arg(id);

-- name: ListBookingsForUser :many
SELECT b.id, b.status, b.rent_type, b.room_id, r.name AS room_name,
       b.start_date, b.duration, b.end_date, b.price, b.expires_at, b.created_at
FROM bookings b
JOIN rooms r ON r.id = b.room_id
WHERE b.tenant_id = sqlc.arg(tenant_id)
ORDER BY b.created_at DESC, b.id DESC
LIMIT sqlc.arg(result_limit) OFFSET sqlc.arg(result_offset);

-- name: GetBookingByID :one
SELECT b.id, b.tenant_id, b.status, b.rent_type, b.room_id, r.name AS room_name,
       p.id AS property_id, p.name AS property_name,
       b.start_date, b.duration, b.end_date, b.period_price, b.price, b.deposit, b.admin_fee,
       b.flash_sale_discount, b.expires_at, b.paid_at, b.owner_deadline_at, b.reject_reason,
       b.deposit_returned_at, b.created_at,
       b.status = 'approved' AS can_extend,
       (
         (
           (b.status = 'approved' AND b.start_date <= (sqlc.arg(today)::date))
           OR b.status IN ('completed', 'terminated')
         )
         AND NOT EXISTS (SELECT 1 FROM reviews rv WHERE rv.booking_id = b.id)
       )::bool AS can_review
FROM bookings b
JOIN rooms r ON r.id = b.room_id
JOIN properties p ON p.id = r.property_id
WHERE b.id = sqlc.arg(id);

-- name: GetInitialInvoiceForBooking :one
SELECT id, total, status FROM invoices WHERE booking_id = sqlc.arg(booking_id) AND type = 'initial';

-- name: GetLatestPaymentForInvoice :one
SELECT order_id, gross_amount, status, snap_token, redirect_url, expires_at
FROM payments
WHERE invoice_id = sqlc.arg(invoice_id)
ORDER BY id DESC
LIMIT 1;

-- name: GetRefundForInvoice :one
SELECT amount, status, completed_at FROM refunds WHERE invoice_id = sqlc.arg(invoice_id);
