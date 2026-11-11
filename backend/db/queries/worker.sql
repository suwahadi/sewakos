-- name: SelectExpiredPendingPaymentOrderIDs :many
SELECT order_id FROM payments
WHERE status = 'pending' AND expires_at < now() - interval '1 minute'
ORDER BY id
LIMIT 100
FOR UPDATE SKIP LOCKED;

-- name: SelectExpiredBookingIDsWithoutPendingPayment :many
SELECT b.id FROM bookings b
WHERE b.status = 'pending' AND b.expires_at < now()
  AND NOT EXISTS (
    SELECT 1 FROM invoices i JOIN payments p ON p.invoice_id = i.id
    WHERE i.booking_id = b.id AND p.status = 'pending'
  )
ORDER BY b.id
LIMIT 100
FOR UPDATE OF b SKIP LOCKED;

-- name: CancelBookingBySystem :exec
UPDATE bookings SET status = 'cancelled', cancelled_by = 'system', updated_at = now() WHERE id = sqlc.arg(id);

-- name: SelectDueEmailOutbox :many
SELECT id, to_email, subject, body_text, attempts FROM email_outbox
WHERE sent_at IS NULL AND next_attempt_at <= now() AND attempts < 8
ORDER BY id
LIMIT 100
FOR UPDATE SKIP LOCKED;

-- name: MarkEmailSent :exec
UPDATE email_outbox SET sent_at = now(), updated_at = now() WHERE id = sqlc.arg(id);

-- name: MarkEmailAttemptFailed :exec
UPDATE email_outbox SET attempts = attempts + 1, next_attempt_at = sqlc.arg(next_attempt_at),
  last_error = sqlc.arg(last_error), updated_at = now()
WHERE id = sqlc.arg(id);
