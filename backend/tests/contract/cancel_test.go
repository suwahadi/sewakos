package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/suwahadi/sewakos/backend/internal/midtrans"
)

func cancelBooking(t *testing.T, srv *httptest.Server, token string, bookingID int64) (*http.Response, []byte) {
	t.Helper()
	return bookingAuthedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/booking/%d/cancel", bookingID), token, "")
}

func setRefundAccount(t *testing.T, srv *httptest.Server, token string, bookingID int64, body string) (*http.Response, []byte) {
	t.Helper()
	return bookingAuthedRequest(t, srv, http.MethodPut, fmt.Sprintf("/api/booking/%d/refund-account", bookingID), token, body)
}

func TestCancelPendingBookingReleasesLockWithoutRefund(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, token := bookingRegisterAndVerify(t, srv)
	bookingID := submitBookingFixture(t, srv, token, roomID)

	resp, body := cancelBooking(t, srv, token, bookingID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("cancel: status = %d, body = %s", resp.StatusCode, body)
	}
	var parsed struct {
		Booking struct {
			Status string `json:"status"`
		} `json:"booking"`
		Refund any `json:"refund"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if parsed.Booking.Status != "cancelled" || parsed.Refund != nil {
		t.Errorf("booking.status = %q, refund = %v, want cancelled with no refund", parsed.Booking.Status, parsed.Refund)
	}

	var available int
	if err := pool.QueryRow(context.Background(), `SELECT available FROM rooms WHERE id = $1`, roomID).Scan(&available); err != nil {
		t.Fatalf("read room: %v", err)
	}
	if available != 2 {
		t.Errorf("room available = %d, want 2 (pending cancel had no lock to release)", available)
	}
}

func TestCancelPendingBookingWithUncheckedPaymentReleasesLockAndExpires(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	expireCalled := false
	mock := &midtrans.MockClient{
		CreateTransactionFunc: func(_ context.Context, req midtrans.TransactionRequest) (midtrans.TransactionResponse, error) {
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

	resp, body = cancelBooking(t, srv, token, bookingID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("cancel: status = %d, body = %s", resp.StatusCode, body)
	}
	if !expireCalled {
		t.Error("ExpireTransaction was not called for a still-pending payment")
	}

	var available int
	if err := pool.QueryRow(context.Background(), `SELECT available FROM rooms WHERE id = $1`, roomID).Scan(&available); err != nil {
		t.Fatalf("read room: %v", err)
	}
	if available != 1 {
		t.Errorf("room available = %d, want 1 (lock released on cancel)", available)
	}

	var paymentStatus string
	if err := pool.QueryRow(context.Background(), `SELECT status FROM payments WHERE invoice_id = (
		SELECT id FROM invoices WHERE booking_id = $1 AND type = 'initial')`, bookingID).Scan(&paymentStatus); err != nil {
		t.Fatalf("read payment: %v", err)
	}
	if paymentStatus != "expire" {
		t.Errorf("payment.status = %q, want expire", paymentStatus)
	}
}

func TestCancelPendingBookingPromotesLateSettlementToPaidThenCancels(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	mock := &midtrans.MockClient{
		CreateTransactionFunc: func(_ context.Context, req midtrans.TransactionRequest) (midtrans.TransactionResponse, error) {
			return midtrans.TransactionResponse{Token: "snap-token", RedirectURL: "https://example.test/redirect"}, nil
		},
		GetStatusFunc: func(_ context.Context, orderID string) (midtrans.StatusResponse, error) {
			return midtrans.StatusResponse{
				OrderID: orderID, TransactionStatus: "settlement", FraudStatus: "accept",
				GrossAmount: grossAmountString(4500000), PaymentType: "bank_transfer", TransactionID: "txn-cancel-1",
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

	resp, body = cancelBooking(t, srv, token, bookingID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("cancel: status = %d, body = %s", resp.StatusCode, body)
	}
	var parsed struct {
		Booking struct {
			Status string `json:"status"`
		} `json:"booking"`
		Refund struct {
			Amount int64  `json:"amount"`
			Status string `json:"status"`
		} `json:"refund"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if parsed.Booking.Status != "cancelled" {
		t.Errorf("booking.status = %q, want cancelled", parsed.Booking.Status)
	}
	if parsed.Refund.Amount != 4500000 || parsed.Refund.Status != "awaiting_account" {
		t.Errorf("refund = %+v, want amount=4500000 status=awaiting_account", parsed.Refund)
	}

	var paymentStatus string
	if err := pool.QueryRow(context.Background(), `SELECT status FROM payments WHERE invoice_id = (
		SELECT id FROM invoices WHERE booking_id = $1 AND type = 'initial')`, bookingID).Scan(&paymentStatus); err != nil {
		t.Fatalf("read payment: %v", err)
	}
	if paymentStatus != "settlement" {
		t.Errorf("payment.status = %q, want settlement (ApplyPaymentStatus must still run)", paymentStatus)
	}
}

func TestCancelPaidBookingRefundsExcludingAdminFee(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	t.Setenv("MIDTRANS_SERVER_KEY", webhookTestServerKey)
	mock := &midtrans.MockClient{
		CreateTransactionFunc: func(_ context.Context, req midtrans.TransactionRequest) (midtrans.TransactionResponse, error) {
			return midtrans.TransactionResponse{Token: "snap-token", RedirectURL: "https://example.test/redirect"}, nil
		},
		GetStatusFunc: func(_ context.Context, orderID string) (midtrans.StatusResponse, error) {
			return midtrans.StatusResponse{
				OrderID: orderID, TransactionStatus: "settlement", FraudStatus: "accept",
				GrossAmount: grossAmountString(4500000), PaymentType: "bank_transfer", TransactionID: "txn-cancel-2",
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
	orderID := ""
	var parsedCheckout struct {
		Payment struct {
			OrderID string `json:"order_id"`
		} `json:"payment"`
	}
	_ = json.Unmarshal(body, &parsedCheckout)
	orderID = parsedCheckout.Payment.OrderID

	resp, body = postWebhookNotification(t, srv, orderID, "200", grossAmountString(4500000))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook: status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body = cancelBooking(t, srv, token, bookingID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("cancel: status = %d, body = %s", resp.StatusCode, body)
	}
	var parsed struct {
		Booking struct {
			Status string `json:"status"`
		} `json:"booking"`
		Refund struct {
			Amount int64  `json:"amount"`
			Status string `json:"status"`
		} `json:"refund"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if parsed.Booking.Status != "cancelled" {
		t.Errorf("booking.status = %q, want cancelled", parsed.Booking.Status)
	}
	if parsed.Refund.Amount != 4500000 || parsed.Refund.Status != "awaiting_account" {
		t.Errorf("refund = %+v, want amount=4500000 (admin_fee is 0 in this fixture) status=awaiting_account", parsed.Refund)
	}

	var available int
	if err := pool.QueryRow(context.Background(), `SELECT available FROM rooms WHERE id = $1`, roomID).Scan(&available); err != nil {
		t.Fatalf("read room: %v", err)
	}
	if available != 1 {
		t.Errorf("room available = %d, want 1 (lock released on cancel-after-paid)", available)
	}
}

func TestCancelAlreadyCancelledBookingIs400(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, token := bookingRegisterAndVerify(t, srv)
	bookingID := submitBookingFixture(t, srv, token, roomID)

	resp, body := cancelBooking(t, srv, token, bookingID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first cancel: status = %d, body = %s", resp.StatusCode, body)
	}
	resp, body = cancelBooking(t, srv, token, bookingID)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("second cancel: status = %d, body = %s, want 400", resp.StatusCode, body)
	}
}

func TestCancelForbiddenForOtherTenant(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, tokenA := bookingRegisterAndVerify(t, srv)
	_, tokenB := bookingRegisterAndVerify(t, srv)
	bookingID := submitBookingFixture(t, srv, tokenA, roomID)

	resp, body := cancelBooking(t, srv, tokenB, bookingID)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s, want 403", resp.StatusCode, body)
	}
}

func TestRefundAccountSavesAndReturnsReady(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	t.Setenv("MIDTRANS_SERVER_KEY", webhookTestServerKey)
	mock := &midtrans.MockClient{
		CreateTransactionFunc: func(_ context.Context, req midtrans.TransactionRequest) (midtrans.TransactionResponse, error) {
			return midtrans.TransactionResponse{Token: "snap-token", RedirectURL: "https://example.test/redirect"}, nil
		},
		GetStatusFunc: func(_ context.Context, orderID string) (midtrans.StatusResponse, error) {
			return midtrans.StatusResponse{
				OrderID: orderID, TransactionStatus: "settlement", FraudStatus: "accept",
				GrossAmount: grossAmountString(4500000), PaymentType: "bank_transfer", TransactionID: "txn-cancel-3",
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
	var parsedCheckout struct {
		Payment struct {
			OrderID string `json:"order_id"`
		} `json:"payment"`
	}
	_ = json.Unmarshal(body, &parsedCheckout)

	resp, body = postWebhookNotification(t, srv, parsedCheckout.Payment.OrderID, "200", grossAmountString(4500000))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook: status = %d, body = %s", resp.StatusCode, body)
	}
	resp, body = cancelBooking(t, srv, token, bookingID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("cancel: status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body = setRefundAccount(t, srv, token, bookingID,
		`{"bank_name":"BCA","account_number":"1234567890","account_name":"Anak Kos Contoh"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("refund-account: status = %d, body = %s", resp.StatusCode, body)
	}
	var parsed struct {
		Refund struct {
			Amount int64  `json:"amount"`
			Status string `json:"status"`
		} `json:"refund"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if parsed.Refund.Status != "ready" || parsed.Refund.Amount != 4500000 {
		t.Errorf("refund = %+v, want status=ready amount=4500000", parsed.Refund)
	}
}

func TestRefundAccountInvalidAccountNumberIs422(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	t.Setenv("MIDTRANS_SERVER_KEY", webhookTestServerKey)
	mock := &midtrans.MockClient{
		CreateTransactionFunc: func(_ context.Context, req midtrans.TransactionRequest) (midtrans.TransactionResponse, error) {
			return midtrans.TransactionResponse{Token: "snap-token", RedirectURL: "https://example.test/redirect"}, nil
		},
		GetStatusFunc: func(_ context.Context, orderID string) (midtrans.StatusResponse, error) {
			return midtrans.StatusResponse{
				OrderID: orderID, TransactionStatus: "settlement", FraudStatus: "accept",
				GrossAmount: grossAmountString(4500000), PaymentType: "bank_transfer", TransactionID: "txn-cancel-4",
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
	var parsedCheckout struct {
		Payment struct {
			OrderID string `json:"order_id"`
		} `json:"payment"`
	}
	_ = json.Unmarshal(body, &parsedCheckout)
	resp, body = postWebhookNotification(t, srv, parsedCheckout.Payment.OrderID, "200", grossAmountString(4500000))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook: status = %d, body = %s", resp.StatusCode, body)
	}
	resp, body = cancelBooking(t, srv, token, bookingID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("cancel: status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body = setRefundAccount(t, srv, token, bookingID,
		`{"bank_name":"BCA","account_number":"abc","account_name":"Anak Kos Contoh"}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s, want 422", resp.StatusCode, body)
	}
}

func TestRefundAccountWithoutRefundIs400(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, token := bookingRegisterAndVerify(t, srv)
	bookingID := submitBookingFixture(t, srv, token, roomID)

	resp, body := setRefundAccount(t, srv, token, bookingID,
		`{"bank_name":"BCA","account_number":"1234567890","account_name":"Anak Kos Contoh"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s, want 400", resp.StatusCode, body)
	}
}
