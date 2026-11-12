package contract

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/suwahadi/sewakos/backend/internal/midtrans"
)

const webhookTestServerKey = "test-midtrans-server-key"

func webhookSignature(orderID, statusCode, grossAmount string) string {
	sum := sha512.Sum512([]byte(orderID + statusCode + grossAmount + webhookTestServerKey))
	return hex.EncodeToString(sum[:])
}

func postWebhookNotification(t *testing.T, srv *httptest.Server, orderID, statusCode, grossAmount string) (*http.Response, []byte) {
	t.Helper()
	body := fmt.Sprintf(`{"order_id":%q,"status_code":%q,"gross_amount":%q,"signature_key":%q}`,
		orderID, statusCode, grossAmount, webhookSignature(orderID, statusCode, grossAmount))
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/payments/midtrans/notification", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		t.Fatalf("read response: %v", err)
	}
	return resp, buf.Bytes()
}

func grossAmountString(amount int64) string {
	return fmt.Sprintf("%d.00", amount)
}

func TestWebhookSettlesInitialPaymentAndBooking(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	t.Setenv("MIDTRANS_SERVER_KEY", webhookTestServerKey)

	var lastOrderID string
	mock := &midtrans.MockClient{
		CreateTransactionFunc: func(_ context.Context, req midtrans.TransactionRequest) (midtrans.TransactionResponse, error) {
			lastOrderID = req.TransactionDetails.OrderID
			return midtrans.TransactionResponse{Token: "snap-token", RedirectURL: "https://example.test/redirect"}, nil
		},
		GetStatusFunc: func(_ context.Context, orderID string) (midtrans.StatusResponse, error) {
			return midtrans.StatusResponse{
				OrderID: orderID, TransactionStatus: "settlement", FraudStatus: "accept",
				GrossAmount: grossAmountString(4500000), PaymentType: "bank_transfer", TransactionID: "txn-1",
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
	orderID := lastOrderID

	resp, body = postWebhookNotification(t, srv, orderID, "200", grossAmountString(4500000))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook: status = %d, body = %s", resp.StatusCode, body)
	}

	var bookingStatus string
	var paidAtSet, ownerDeadlineSet bool
	if err := pool.QueryRow(context.Background(),
		`SELECT status, paid_at IS NOT NULL, owner_deadline_at IS NOT NULL FROM bookings WHERE id = $1`, bookingID,
	).Scan(&bookingStatus, &paidAtSet, &ownerDeadlineSet); err != nil {
		t.Fatalf("read booking: %v", err)
	}
	if bookingStatus != "paid" || !paidAtSet || !ownerDeadlineSet {
		t.Errorf("booking = (status=%q, paid_at set=%v, owner_deadline_at set=%v), want paid with both set",
			bookingStatus, paidAtSet, ownerDeadlineSet)
	}

	var paymentStatus string
	if err := pool.QueryRow(context.Background(),
		`SELECT p.status FROM payments p WHERE p.order_id = $1`, orderID,
	).Scan(&paymentStatus); err != nil {
		t.Fatalf("read payment: %v", err)
	}
	if paymentStatus != "settlement" {
		t.Errorf("payment.status = %q, want settlement", paymentStatus)
	}

	var receivedCount, adminFeeCount int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM ledger_entries WHERE booking_id = $1 AND type = 'payment_received'`, bookingID,
	).Scan(&receivedCount); err != nil {
		t.Fatalf("count payment_received ledger: %v", err)
	}
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM ledger_entries WHERE booking_id = $1 AND type = 'admin_fee_earned'`, bookingID,
	).Scan(&adminFeeCount); err != nil {
		t.Fatalf("count admin_fee_earned ledger: %v", err)
	}
	if receivedCount != 1 {
		t.Errorf("payment_received ledger rows = %d, want 1", receivedCount)
	}
	if adminFeeCount != 0 {
		t.Errorf("admin_fee_earned ledger rows = %d, want 0 (admin_fee is 0 in this fixture)", adminFeeCount)
	}
}

func TestWebhookDuplicateNotificationIsIdempotent(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	t.Setenv("MIDTRANS_SERVER_KEY", webhookTestServerKey)

	var lastOrderID string
	mock := &midtrans.MockClient{
		CreateTransactionFunc: func(_ context.Context, req midtrans.TransactionRequest) (midtrans.TransactionResponse, error) {
			lastOrderID = req.TransactionDetails.OrderID
			return midtrans.TransactionResponse{Token: "snap-token", RedirectURL: "https://example.test/redirect"}, nil
		},
		GetStatusFunc: func(_ context.Context, orderID string) (midtrans.StatusResponse, error) {
			return midtrans.StatusResponse{
				OrderID: orderID, TransactionStatus: "settlement", FraudStatus: "accept",
				GrossAmount: grossAmountString(4500000), PaymentType: "bank_transfer", TransactionID: "txn-2",
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
	orderID := lastOrderID

	for i := 0; i < 2; i++ {
		resp, body := postWebhookNotification(t, srv, orderID, "200", grossAmountString(4500000))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("webhook call %d: status = %d, body = %s", i, resp.StatusCode, body)
		}
	}

	var receivedCount int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM ledger_entries WHERE booking_id = $1 AND type = 'payment_received'`, bookingID,
	).Scan(&receivedCount); err != nil {
		t.Fatalf("count payment_received ledger: %v", err)
	}
	if receivedCount != 1 {
		t.Errorf("payment_received ledger rows = %d, want 1 (duplicate notification must not double-record)", receivedCount)
	}
}

func TestWebhookOutOfOrderNotificationIsIgnored(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	t.Setenv("MIDTRANS_SERVER_KEY", webhookTestServerKey)

	var lastOrderID string
	mock := &midtrans.MockClient{
		CreateTransactionFunc: func(_ context.Context, req midtrans.TransactionRequest) (midtrans.TransactionResponse, error) {
			lastOrderID = req.TransactionDetails.OrderID
			return midtrans.TransactionResponse{Token: "snap-token", RedirectURL: "https://example.test/redirect"}, nil
		},
		GetStatusFunc: func(_ context.Context, orderID string) (midtrans.StatusResponse, error) {
			return midtrans.StatusResponse{
				OrderID: orderID, TransactionStatus: "settlement", FraudStatus: "accept",
				GrossAmount: grossAmountString(4500000), PaymentType: "bank_transfer", TransactionID: "txn-3",
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
	orderID := lastOrderID

	resp, body = postWebhookNotification(t, srv, orderID, "200", grossAmountString(4500000))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first webhook: status = %d, body = %s", resp.StatusCode, body)
	}

	// A stale notification body (different status_code, so a different
	// signature) arrives after the authoritative settlement already landed.
	// GetStatus always reports the current truth regardless of what this
	// late body claims, so this must not revert the already-paid booking.
	resp, body = postWebhookNotification(t, srv, orderID, "190", grossAmountString(4500000))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stale webhook: status = %d, body = %s", resp.StatusCode, body)
	}

	var bookingStatus string
	if err := pool.QueryRow(context.Background(), `SELECT status FROM bookings WHERE id = $1`, bookingID).Scan(&bookingStatus); err != nil {
		t.Fatalf("read booking: %v", err)
	}
	if bookingStatus != "paid" {
		t.Errorf("booking.status = %q, want paid (stale out-of-order notification must not change it)", bookingStatus)
	}

	var receivedCount int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM ledger_entries WHERE booking_id = $1 AND type = 'payment_received'`, bookingID,
	).Scan(&receivedCount); err != nil {
		t.Fatalf("count payment_received ledger: %v", err)
	}
	if receivedCount != 1 {
		t.Errorf("payment_received ledger rows = %d, want 1", receivedCount)
	}
}

func TestWebhookMismatchedGrossAmountMarksFailure(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	t.Setenv("MIDTRANS_SERVER_KEY", webhookTestServerKey)

	var lastOrderID string
	mock := &midtrans.MockClient{
		CreateTransactionFunc: func(_ context.Context, req midtrans.TransactionRequest) (midtrans.TransactionResponse, error) {
			lastOrderID = req.TransactionDetails.OrderID
			return midtrans.TransactionResponse{Token: "snap-token", RedirectURL: "https://example.test/redirect"}, nil
		},
		GetStatusFunc: func(_ context.Context, orderID string) (midtrans.StatusResponse, error) {
			return midtrans.StatusResponse{
				OrderID: orderID, TransactionStatus: "settlement", FraudStatus: "accept",
				GrossAmount: grossAmountString(999), PaymentType: "bank_transfer", TransactionID: "txn-4",
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
	orderID := lastOrderID

	resp, body = postWebhookNotification(t, srv, orderID, "200", grossAmountString(4500000))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook: status = %d, body = %s", resp.StatusCode, body)
	}

	var paymentStatus string
	if err := pool.QueryRow(context.Background(), `SELECT status FROM payments WHERE order_id = $1`, orderID).Scan(&paymentStatus); err != nil {
		t.Fatalf("read payment: %v", err)
	}
	if paymentStatus != "failure" {
		t.Errorf("payment.status = %q, want failure (gross_amount mismatch)", paymentStatus)
	}

	var bookingStatus string
	if err := pool.QueryRow(context.Background(), `SELECT status FROM bookings WHERE id = $1`, bookingID).Scan(&bookingStatus); err != nil {
		t.Fatalf("read booking: %v", err)
	}
	if bookingStatus != "pending" {
		t.Errorf("booking.status = %q, want pending (mismatched notification must not be applied)", bookingStatus)
	}

	var ledgerCount int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM ledger_entries WHERE booking_id = $1`, bookingID).Scan(&ledgerCount); err != nil {
		t.Fatalf("count ledger entries: %v", err)
	}
	if ledgerCount != 0 {
		t.Errorf("ledger entries = %d, want 0 (mismatched amount must not be processed)", ledgerCount)
	}
}

func TestWebhookPayAfterLockReleasedRelockSucceeds(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	t.Setenv("MIDTRANS_SERVER_KEY", webhookTestServerKey)

	var lastOrderID string
	mock := &midtrans.MockClient{
		CreateTransactionFunc: func(_ context.Context, req midtrans.TransactionRequest) (midtrans.TransactionResponse, error) {
			lastOrderID = req.TransactionDetails.OrderID
			return midtrans.TransactionResponse{Token: "snap-token", RedirectURL: "https://example.test/redirect"}, nil
		},
		GetStatusFunc: func(_ context.Context, orderID string) (midtrans.StatusResponse, error) {
			return midtrans.StatusResponse{
				OrderID: orderID, TransactionStatus: "settlement", FraudStatus: "accept",
				GrossAmount: grossAmountString(4500000), PaymentType: "bank_transfer", TransactionID: "txn-5",
			}, nil
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
	orderID := lastOrderID

	// Simulate the lock having been released (e.g. by the expire_bookings
	// worker) before the late settlement notification arrives, with the unit
	// still free for this handler to reclaim.
	if _, err := pool.Exec(context.Background(),
		`UPDATE bookings SET stock_held = false WHERE id = $1`, bookingID); err != nil {
		t.Fatalf("simulate released lock: %v", err)
	}
	if _, err := pool.Exec(context.Background(),
		`UPDATE rooms SET available = available + 1 WHERE id = $1`, roomID); err != nil {
		t.Fatalf("simulate released lock: %v", err)
	}

	resp, body = postWebhookNotification(t, srv, orderID, "200", grossAmountString(4500000))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook: status = %d, body = %s", resp.StatusCode, body)
	}

	var bookingStatus string
	var stockHeld bool
	if err := pool.QueryRow(context.Background(), `SELECT status, stock_held FROM bookings WHERE id = $1`, bookingID).
		Scan(&bookingStatus, &stockHeld); err != nil {
		t.Fatalf("read booking: %v", err)
	}
	if bookingStatus != "paid" || !stockHeld {
		t.Errorf("booking = (status=%q, stock_held=%v), want paid with stock re-held", bookingStatus, stockHeld)
	}

	var available int
	if err := pool.QueryRow(context.Background(), `SELECT available FROM rooms WHERE id = $1`, roomID).Scan(&available); err != nil {
		t.Fatalf("read room: %v", err)
	}
	if available != 0 {
		t.Errorf("room available = %d, want 0 (relocked)", available)
	}
}

func TestWebhookPayAfterLockReleasedRelockFails(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	t.Setenv("MIDTRANS_SERVER_KEY", webhookTestServerKey)

	var lastOrderID string
	mock := &midtrans.MockClient{
		CreateTransactionFunc: func(_ context.Context, req midtrans.TransactionRequest) (midtrans.TransactionResponse, error) {
			lastOrderID = req.TransactionDetails.OrderID
			return midtrans.TransactionResponse{Token: "snap-token", RedirectURL: "https://example.test/redirect"}, nil
		},
		GetStatusFunc: func(_ context.Context, orderID string) (midtrans.StatusResponse, error) {
			return midtrans.StatusResponse{
				OrderID: orderID, TransactionStatus: "settlement", FraudStatus: "accept",
				GrossAmount: grossAmountString(4500000), PaymentType: "bank_transfer", TransactionID: "txn-6",
			}, nil
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
	orderID := lastOrderID

	// Lock released (as above) but, unlike the relock-succeeds case, another
	// booking takes the unit before this late notification arrives, so there
	// is nothing left to reclaim.
	if _, err := pool.Exec(context.Background(),
		`UPDATE bookings SET stock_held = false WHERE id = $1`, bookingID); err != nil {
		t.Fatalf("simulate released lock: %v", err)
	}

	resp, body = postWebhookNotification(t, srv, orderID, "200", grossAmountString(4500000))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook: status = %d, body = %s", resp.StatusCode, body)
	}

	var bookingStatus string
	var rejectedBy string
	if err := pool.QueryRow(context.Background(), `SELECT status, rejected_by FROM bookings WHERE id = $1`, bookingID).
		Scan(&bookingStatus, &rejectedBy); err != nil {
		t.Fatalf("read booking: %v", err)
	}
	if bookingStatus != "rejected" || rejectedBy != "system" {
		t.Errorf("booking = (status=%q, rejected_by=%q), want rejected by system", bookingStatus, rejectedBy)
	}

	var refundAmount int64
	var refundReason string
	if err := pool.QueryRow(context.Background(), `SELECT amount, reason FROM refunds WHERE booking_id = $1`, bookingID).
		Scan(&refundAmount, &refundReason); err != nil {
		t.Fatalf("read refund: %v", err)
	}
	if refundAmount != 4500000 || refundReason != "stock_unavailable" {
		t.Errorf("refund = (amount=%d, reason=%q), want (4500000, stock_unavailable)", refundAmount, refundReason)
	}

	var receivedCount, refundDueCount int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM ledger_entries WHERE booking_id = $1 AND type = 'payment_received'`, bookingID,
	).Scan(&receivedCount); err != nil {
		t.Fatalf("count payment_received ledger: %v", err)
	}
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM ledger_entries WHERE booking_id = $1 AND type = 'refund_due'`, bookingID,
	).Scan(&refundDueCount); err != nil {
		t.Fatalf("count refund_due ledger: %v", err)
	}
	if receivedCount != 1 || refundDueCount != 1 {
		t.Errorf("ledger rows (payment_received=%d, refund_due=%d), want (1, 1)", receivedCount, refundDueCount)
	}
}

func TestWebhookUnknownSignatureIs403(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", webhookTestServerKey)
	srv, _ := newTestServerAndPool(t)

	body := `{"order_id":"KOS-1-1","status_code":"200","gross_amount":"4500000.00","signature_key":"wrong"}`
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/payments/midtrans/notification", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestWebhookUnknownOrderIDIsAcknowledged(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", webhookTestServerKey)
	srv, _ := newTestServerAndPool(t)

	resp, body := postWebhookNotification(t, srv, "payment_notif_test_M000000000_x", "200", "105000.00")
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"message":"OK"`) {
		t.Fatalf("status = %d, body = %s, want 200 OK for a dashboard test notification", resp.StatusCode, body)
	}
}

func TestWebhookVerifiesWithSignatureSecretKeyWhenSet(t *testing.T) {
	t.Setenv("MIDTRANS_SERVER_KEY", "server-key-used-for-api-calls")
	t.Setenv("MIDTRANS_SIGNATURE_KEY", webhookTestServerKey)
	srv, _ := newTestServerAndPool(t)

	resp, body := postWebhookNotification(t, srv, "payment_notif_test_M000000000_y", "200", "105000.00")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("signed with signature key: status = %d, body = %s, want 200", resp.StatusCode, body)
	}

	sum := sha512.Sum512([]byte("payment_notif_test_M000000000_y" + "200" + "105000.00" + "server-key-used-for-api-calls"))
	payload := fmt.Sprintf(`{"order_id":"payment_notif_test_M000000000_y","status_code":"200","gross_amount":"105000.00","signature_key":%q}`, hex.EncodeToString(sum[:]))
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/payments/midtrans/notification", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	signedWithServerKey, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	signedWithServerKey.Body.Close()
	if signedWithServerKey.StatusCode != http.StatusForbidden {
		t.Errorf("signed with server key while a signature key is set: status = %d, want 403", signedWithServerKey.StatusCode)
	}
}
