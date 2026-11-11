package contract

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/sewakos/backend/internal/booking"
	"github.com/suwahadi/sewakos/backend/internal/midtrans"
	"github.com/suwahadi/sewakos/backend/internal/worker"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func workerDeps(pool *pgxpool.Pool, midtransClient midtrans.Client, sender *fakeEmailSender) worker.Deps {
	return worker.Deps{
		Booking: booking.Deps{DB: pool, Midtrans: midtransClient, MidtransServerKey: webhookTestServerKey},
		Email:   sender,
	}
}

type fakeEmailSender struct {
	err  error
	sent []string
}

func (s *fakeEmailSender) Send(_ context.Context, to, subject, body string) error {
	s.sent = append(s.sent, to+"|"+subject+"|"+body)
	return s.err
}

func backdatePaymentExpiry(t *testing.T, pool *pgxpool.Pool, orderID string, ago time.Duration) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `UPDATE payments SET expires_at = $1 WHERE order_id = $2`,
		time.Now().Add(-ago), orderID); err != nil {
		t.Fatalf("backdate payment expiry: %v", err)
	}
}

func backdateBookingExpiry(t *testing.T, pool *pgxpool.Pool, bookingID int64, ago time.Duration) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `UPDATE bookings SET expires_at = $1 WHERE id = $2`,
		time.Now().Add(-ago), bookingID); err != nil {
		t.Fatalf("backdate booking expiry: %v", err)
	}
}

func TestExpirePaymentsMarksStillPendingAndReleasesLock(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	expireCalled := false
	var lastOrderID string
	mock := &midtrans.MockClient{
		CreateTransactionFunc: func(_ context.Context, req midtrans.TransactionRequest) (midtrans.TransactionResponse, error) {
			lastOrderID = req.TransactionDetails.OrderID
			return midtrans.TransactionResponse{Token: "snap-token", RedirectURL: "https://example.test/redirect"}, nil
		},
		GetStatusFunc: func(_ context.Context, orderID string) (midtrans.StatusResponse, error) {
			return midtrans.StatusResponse{OrderID: orderID, TransactionStatus: "pending"}, nil
		},
		ExpireTransactionFunc: func(_ context.Context, _ string) error {
			expireCalled = true
			return nil
		},
	}
	srv, pool := newTestServerAndPoolWithMidtrans(t, mock)
	roomID := seedBookingFixture(t, pool, 2250000, 1, false)
	_, token := bookingRegisterAndVerify(t, srv)
	bookingID := submitBookingFixture(t, srv, token, roomID)
	resp, body := checkoutBooking(t, srv, token, bookingID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("checkout: status = %d, body = %s", resp.StatusCode, body)
	}
	backdatePaymentExpiry(t, pool, lastOrderID, 2*time.Minute)

	worker.ExpirePayments(context.Background(), workerDeps(pool, mock, nil), discardLogger())

	if !expireCalled {
		t.Error("ExpireTransaction was not called")
	}
	var paymentStatus string
	if err := pool.QueryRow(context.Background(), `SELECT status FROM payments WHERE order_id = $1`, lastOrderID).Scan(&paymentStatus); err != nil {
		t.Fatalf("read payment: %v", err)
	}
	if paymentStatus != "expire" {
		t.Errorf("payment.status = %q, want expire", paymentStatus)
	}
	var available int
	if err := pool.QueryRow(context.Background(), `SELECT available FROM rooms WHERE id = $1`, roomID).Scan(&available); err != nil {
		t.Fatalf("read room: %v", err)
	}
	if available != 1 {
		t.Errorf("room available = %d, want 1 (lock released)", available)
	}
}

func TestExpirePaymentsAppliesLateSettlement(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	var lastOrderID string
	mock := &midtrans.MockClient{
		CreateTransactionFunc: func(_ context.Context, req midtrans.TransactionRequest) (midtrans.TransactionResponse, error) {
			lastOrderID = req.TransactionDetails.OrderID
			return midtrans.TransactionResponse{Token: "snap-token", RedirectURL: "https://example.test/redirect"}, nil
		},
		GetStatusFunc: func(_ context.Context, orderID string) (midtrans.StatusResponse, error) {
			return midtrans.StatusResponse{
				OrderID: orderID, TransactionStatus: "settlement", FraudStatus: "accept",
				GrossAmount: grossAmountString(4500000), PaymentType: "bank_transfer", TransactionID: "txn-worker-1",
			}, nil
		},
	}
	srv, pool := newTestServerAndPoolWithMidtrans(t, mock)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, token := bookingRegisterAndVerify(t, srv)
	bookingID := submitBookingFixture(t, srv, token, roomID)
	resp, body := checkoutBooking(t, srv, token, bookingID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("checkout: status = %d, body = %s", resp.StatusCode, body)
	}
	backdatePaymentExpiry(t, pool, lastOrderID, 2*time.Minute)

	worker.ExpirePayments(context.Background(), workerDeps(pool, mock, nil), discardLogger())

	var bookingStatus string
	if err := pool.QueryRow(context.Background(), `SELECT status FROM bookings WHERE id = $1`, bookingID).Scan(&bookingStatus); err != nil {
		t.Fatalf("read booking: %v", err)
	}
	if bookingStatus != "paid" {
		t.Errorf("booking.status = %q, want paid (late settlement found by the job must still apply)", bookingStatus)
	}
}

func TestExpireBookingsCancelsExpiredPendingBookingWithoutPayment(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, token := bookingRegisterAndVerify(t, srv)
	bookingID := submitBookingFixture(t, srv, token, roomID)
	backdateBookingExpiry(t, pool, bookingID, time.Minute)

	worker.ExpireBookings(context.Background(), workerDeps(pool, nil, nil), discardLogger())

	var status, cancelledBy string
	if err := pool.QueryRow(context.Background(), `SELECT status, cancelled_by FROM bookings WHERE id = $1`, bookingID).
		Scan(&status, &cancelledBy); err != nil {
		t.Fatalf("read booking: %v", err)
	}
	if status != "cancelled" || cancelledBy != "system" {
		t.Errorf("booking = (status=%q, cancelled_by=%q), want cancelled by system", status, cancelledBy)
	}
}

func TestExpireBookingsSkipsBookingWithPendingPayment(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	mock := &midtrans.MockClient{
		CreateTransactionFunc: func(_ context.Context, req midtrans.TransactionRequest) (midtrans.TransactionResponse, error) {
			return midtrans.TransactionResponse{Token: "snap-token", RedirectURL: "https://example.test/redirect"}, nil
		},
	}
	srv, pool := newTestServerAndPoolWithMidtrans(t, mock)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, token := bookingRegisterAndVerify(t, srv)
	bookingID := submitBookingFixture(t, srv, token, roomID)
	resp, body := checkoutBooking(t, srv, token, bookingID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("checkout: status = %d, body = %s", resp.StatusCode, body)
	}
	backdateBookingExpiry(t, pool, bookingID, time.Minute)

	worker.ExpireBookings(context.Background(), workerDeps(pool, mock, nil), discardLogger())

	var status string
	if err := pool.QueryRow(context.Background(), `SELECT status FROM bookings WHERE id = $1`, bookingID).Scan(&status); err != nil {
		t.Fatalf("read booking: %v", err)
	}
	if status != "pending" {
		t.Errorf("booking.status = %q, want pending (a pending payment exists, expire_payments owns this row)", status)
	}
}

func TestSendEmailsSendsAndMarksSent(t *testing.T) {
	_, pool := newTestServerAndPool(t)
	var emailID int64
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO email_outbox (to_email, subject, body_text) VALUES ($1, $2, $3) RETURNING id`,
		"tenant@example.test", "Subjek", "Isi email",
	).Scan(&emailID); err != nil {
		t.Fatalf("insert email_outbox: %v", err)
	}
	sender := &fakeEmailSender{}

	worker.SendEmails(context.Background(), workerDeps(pool, nil, sender), discardLogger())

	var sentAtSet bool
	if err := pool.QueryRow(context.Background(), `SELECT sent_at IS NOT NULL FROM email_outbox WHERE id = $1`, emailID).Scan(&sentAtSet); err != nil {
		t.Fatalf("read email_outbox: %v", err)
	}
	if !sentAtSet {
		t.Error("sent_at not set after successful send")
	}
	wantSent := "tenant@example.test|Subjek|Isi email"
	found := false
	for _, sent := range sender.sent {
		if sent == wantSent {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("sender.sent = %v, want it to contain %q", sender.sent, wantSent)
	}
}

func TestSendEmailsBacksOffOnFailure(t *testing.T) {
	_, pool := newTestServerAndPool(t)
	var emailID int64
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO email_outbox (to_email, subject, body_text) VALUES ($1, $2, $3) RETURNING id`,
		"tenant@example.test", "Subjek", "Isi email",
	).Scan(&emailID); err != nil {
		t.Fatalf("insert email_outbox: %v", err)
	}
	sender := &fakeEmailSender{err: errSendFailed}

	worker.SendEmails(context.Background(), workerDeps(pool, nil, sender), discardLogger())

	var attempts int16
	var nextAttemptAfterNow bool
	var lastError string
	if err := pool.QueryRow(context.Background(),
		`SELECT attempts, next_attempt_at > now(), last_error FROM email_outbox WHERE id = $1`, emailID,
	).Scan(&attempts, &nextAttemptAfterNow, &lastError); err != nil {
		t.Fatalf("read email_outbox: %v", err)
	}
	if attempts != 1 || !nextAttemptAfterNow || lastError == "" {
		t.Errorf("email_outbox = (attempts=%d, next_attempt_at future=%v, last_error=%q), want attempts=1 with backoff and an error",
			attempts, nextAttemptAfterNow, lastError)
	}
}

var errSendFailed = errors.New("smtp unreachable")
