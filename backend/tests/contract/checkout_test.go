package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/sewakos/backend/internal/midtrans"
)

func submitBookingFixture(t *testing.T, srv *httptest.Server, token string, roomID int64) int64 {
	t.Helper()
	startDate := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	resp, body := submitBooking(t, srv, token, fmt.Sprintf(
		`{"room_id":%d,"rent_type":"monthly","start_date":%q}`, roomID, startDate))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("submit: status = %d, body = %s", resp.StatusCode, body)
	}
	var submitted struct {
		Booking struct {
			ID int64 `json:"id"`
		} `json:"booking"`
	}
	if err := json.Unmarshal(body, &submitted); err != nil {
		t.Fatalf("parse submit body: %v", err)
	}
	return submitted.Booking.ID
}

func checkoutBooking(t *testing.T, srv *httptest.Server, token string, bookingID int64) (*http.Response, []byte) {
	t.Helper()
	return bookingAuthedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/booking/%d/checkout", bookingID), token, "")
}

func TestBookingCheckoutHappyPath(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, token := bookingRegisterAndVerify(t, srv)
	bookingID := submitBookingFixture(t, srv, token, roomID)

	resp, body := checkoutBooking(t, srv, token, bookingID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("checkout: status = %d, body = %s", resp.StatusCode, body)
	}
	var parsed struct {
		Payment struct {
			OrderID     string `json:"order_id"`
			GrossAmount int64  `json:"gross_amount"`
			SnapToken   string `json:"snap_token"`
			RedirectURL string `json:"redirect_url"`
		} `json:"payment"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse checkout body: %v", err)
	}
	if parsed.Payment.SnapToken == "" || parsed.Payment.RedirectURL == "" {
		t.Errorf("payment snap_token/redirect_url empty: %+v", parsed.Payment)
	}
	wantGross := int64(2250000 + 2250000)
	if parsed.Payment.GrossAmount != wantGross {
		t.Errorf("gross_amount = %d, want %d", parsed.Payment.GrossAmount, wantGross)
	}

	var available int
	if err := pool.QueryRow(context.Background(), `SELECT available FROM rooms WHERE id = $1`, roomID).Scan(&available); err != nil {
		t.Fatalf("read room available: %v", err)
	}
	if available != 1 {
		t.Errorf("room available = %d, want 1 (checkout must lock exactly one unit)", available)
	}
}

func TestBookingCheckoutIsIdempotent(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, token := bookingRegisterAndVerify(t, srv)
	bookingID := submitBookingFixture(t, srv, token, roomID)

	resp, body := checkoutBooking(t, srv, token, bookingID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first checkout: status = %d, body = %s", resp.StatusCode, body)
	}
	var first struct {
		Payment struct {
			OrderID string `json:"order_id"`
		} `json:"payment"`
	}
	_ = json.Unmarshal(body, &first)

	resp, body = checkoutBooking(t, srv, token, bookingID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("second checkout: status = %d, body = %s", resp.StatusCode, body)
	}
	var second struct {
		Payment struct {
			OrderID string `json:"order_id"`
		} `json:"payment"`
	}
	_ = json.Unmarshal(body, &second)

	if first.Payment.OrderID == "" || first.Payment.OrderID != second.Payment.OrderID {
		t.Errorf("order_id = %q then %q, want the same active transaction returned", first.Payment.OrderID, second.Payment.OrderID)
	}

	var available int
	if err := pool.QueryRow(context.Background(), `SELECT available FROM rooms WHERE id = $1`, roomID).Scan(&available); err != nil {
		t.Fatalf("read room available: %v", err)
	}
	if available != 1 {
		t.Errorf("room available = %d, want 1 (re-checkout must not decrement the lock again)", available)
	}
}

func TestBookingCheckoutMidtransFailureReleasesLock(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	failingMidtrans := &midtrans.MockClient{
		CreateTransactionFunc: func(_ context.Context, _ midtrans.TransactionRequest) (midtrans.TransactionResponse, error) {
			return midtrans.TransactionResponse{}, fmt.Errorf("midtrans sandbox unreachable")
		},
	}
	srv, pool := newTestServerAndPoolWithMidtrans(t, failingMidtrans)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, token := bookingRegisterAndVerify(t, srv)
	bookingID := submitBookingFixture(t, srv, token, roomID)

	resp, body := checkoutBooking(t, srv, token, bookingID)
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s, want 502", resp.StatusCode, body)
	}

	var available int
	if err := pool.QueryRow(context.Background(), `SELECT available FROM rooms WHERE id = $1`, roomID).Scan(&available); err != nil {
		t.Fatalf("read room available: %v", err)
	}
	if available != 2 {
		t.Errorf("room available = %d, want 2 (lock must be released when Midtrans fails)", available)
	}

	var stockHeld bool
	if err := pool.QueryRow(context.Background(), `SELECT stock_held FROM bookings WHERE id = $1`, bookingID).Scan(&stockHeld); err != nil {
		t.Fatalf("read booking stock_held: %v", err)
	}
	if stockHeld {
		t.Error("booking.stock_held = true, want false after Midtrans failure")
	}

	var paymentStatus string
	if err := pool.QueryRow(context.Background(),
		`SELECT p.status FROM payments p JOIN invoices i ON i.id = p.invoice_id WHERE i.booking_id = $1 ORDER BY p.id DESC LIMIT 1`,
		bookingID).Scan(&paymentStatus); err != nil {
		t.Fatalf("read payment status: %v", err)
	}
	if paymentStatus != "failure" {
		t.Errorf("payment.status = %q, want failure", paymentStatus)
	}
}

func TestBookingCheckoutForbiddenForOtherTenant(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, tokenA := bookingRegisterAndVerify(t, srv)
	_, tokenB := bookingRegisterAndVerify(t, srv)
	bookingID := submitBookingFixture(t, srv, tokenA, roomID)

	resp, body := checkoutBooking(t, srv, tokenB, bookingID)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s, want 403 for another tenant's booking", resp.StatusCode, body)
	}
}

// TestBookingCheckoutConcurrencyOnlyOneSucceeds is B4.4's hard completion gate
// (docs/IMPLEMENTATION.md): two simultaneous checkouts for the last available
// room unit, driven by real goroutines against the real Postgres pool, must
// leave exactly one booking paid and the other rejected.
func TestBookingCheckoutConcurrencyOnlyOneSucceeds(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 1, false)

	_, tokenA := bookingRegisterAndVerify(t, srv)
	_, tokenB := bookingRegisterAndVerify(t, srv)
	bookingA := submitBookingFixture(t, srv, tokenA, roomID)
	bookingB := submitBookingFixture(t, srv, tokenB, roomID)

	var wg sync.WaitGroup
	statuses := make([]int, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		resp, _ := checkoutBooking(t, srv, tokenA, bookingA)
		statuses[0] = resp.StatusCode
	}()
	go func() {
		defer wg.Done()
		resp, _ := checkoutBooking(t, srv, tokenB, bookingB)
		statuses[1] = resp.StatusCode
	}()
	wg.Wait()

	successes, failures := 0, 0
	for _, status := range statuses {
		switch status {
		case http.StatusOK:
			successes++
		case http.StatusBadRequest:
			failures++
		default:
			t.Errorf("unexpected checkout status %d", status)
		}
	}
	if successes != 1 || failures != 1 {
		t.Fatalf("statuses = %v, want exactly one 200 and one 400 for the last room unit", statuses)
	}

	var available int
	if err := pool.QueryRow(context.Background(), `SELECT available FROM rooms WHERE id = $1`, roomID).Scan(&available); err != nil {
		t.Fatalf("read room available: %v", err)
	}
	if available != 0 {
		t.Errorf("room available = %d, want 0 (exactly one unit locked)", available)
	}
}

func TestBookingCheckoutRetriedAfterFirstCheckoutReleasedLock(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 1, false)

	_, tokenA := bookingRegisterAndVerify(t, srv)
	_, tokenB := bookingRegisterAndVerify(t, srv)
	bookingA := submitBookingFixture(t, srv, tokenA, roomID)
	bookingB := submitBookingFixture(t, srv, tokenB, roomID)

	resp, body := checkoutBooking(t, srv, tokenA, bookingA)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("checkout A: status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body = checkoutBooking(t, srv, tokenB, bookingB)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("checkout B: status = %d, body = %s, want 400 (no units left)", resp.StatusCode, body)
	}

	if _, err := pool.Exec(context.Background(), `UPDATE bookings SET status = 'cancelled' WHERE id = $1`, bookingA); err != nil {
		t.Fatalf("cancel booking A: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE rooms SET available = available + 1 WHERE id = $1`, roomID); err != nil {
		t.Fatalf("release room lock: %v", err)
	}

	resp, body = checkoutBooking(t, srv, tokenB, bookingB)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("checkout B retry: status = %d, body = %s, want 200 once the unit is free again", resp.StatusCode, body)
	}
}

func expireOpenPayments(t *testing.T, pool *pgxpool.Pool, invoiceID int64) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`UPDATE payments SET expires_at = now() - interval '5 minutes' WHERE invoice_id = $1 AND status = 'pending'`, invoiceID); err != nil {
		t.Fatalf("expire payments: %v", err)
	}
}

func countPaymentsWithStatus(t *testing.T, pool *pgxpool.Pool, invoiceID int64, status string) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM payments WHERE invoice_id = $1 AND status = $2`, invoiceID, status).Scan(&count); err != nil {
		t.Fatalf("count payments: %v", err)
	}
	return count
}

func checkoutOrderID(t *testing.T, body []byte) string {
	t.Helper()
	var parsed struct {
		Payment struct {
			OrderID string `json:"order_id"`
		} `json:"payment"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse checkout body: %v", err)
	}
	return parsed.Payment.OrderID
}

// Checkout after the previous payment window expired must issue a fresh payment and keep exactly one unit locked.
func TestBookingCheckoutAfterPaymentExpired(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, token := bookingRegisterAndVerify(t, srv)
	bookingID := submitBookingFixture(t, srv, token, roomID)
	invoiceID := getInitialInvoiceID(t, pool, bookingID)

	resp, body := checkoutBooking(t, srv, token, bookingID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first checkout: status = %d, body = %s", resp.StatusCode, body)
	}
	firstOrderID := checkoutOrderID(t, body)

	expireOpenPayments(t, pool, invoiceID)
	resp, body = checkoutBooking(t, srv, token, bookingID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("checkout after expiry: status = %d, body = %s", resp.StatusCode, body)
	}
	if checkoutOrderID(t, body) == firstOrderID {
		t.Errorf("expected a new order_id after expiry")
	}
	if got := countPaymentsWithStatus(t, pool, invoiceID, "pending"); got != 1 {
		t.Errorf("pending payments = %d, want 1", got)
	}
	if got := countPaymentsWithStatus(t, pool, invoiceID, "expire"); got != 1 {
		t.Errorf("expired payments = %d, want 1", got)
	}

	var available int
	if err := pool.QueryRow(context.Background(), `SELECT available FROM rooms WHERE id = $1`, roomID).Scan(&available); err != nil {
		t.Fatalf("read room available: %v", err)
	}
	if available != 1 {
		t.Errorf("room available = %d, want 1 (expired lock released, new lock taken)", available)
	}
}
