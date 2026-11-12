-- +goose Up
-- Initial invoices settled before ApplyPaymentStatus marked them paid stayed "unpaid"; their payment already settled.
UPDATE invoices i
SET status = 'paid', paid_at = coalesce(b.paid_at, p.paid_at, now()), updated_at = now()
FROM bookings b, payments p
WHERE b.id = i.booking_id AND p.invoice_id = i.id
  AND i.type = 'initial' AND i.status = 'unpaid' AND p.status = 'settlement'
  AND b.status IN ('paid', 'approved', 'completed', 'terminated');

-- +goose Down
SELECT 1;
