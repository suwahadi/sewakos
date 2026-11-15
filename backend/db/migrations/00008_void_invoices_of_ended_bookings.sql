-- +goose Up
UPDATE invoices i
SET status = 'void',
    void_reason = CASE WHEN b.status = 'cancelled' AND b.cancelled_by = 'system' THEN 'booking_expired' ELSE 'booking_cancelled' END,
    updated_at = now()
FROM bookings b
WHERE b.id = i.booking_id
  AND i.status = 'unpaid'
  AND b.status IN ('cancelled', 'rejected', 'terminated')
  AND NOT EXISTS (SELECT 1 FROM payments p WHERE p.invoice_id = i.id AND p.status = 'settlement');

-- +goose Down
SELECT 1;
