package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/sewakos/backend/internal/midtrans"
)

func invoiceAuthedGet(t *testing.T, srv *httptest.Server, token, path string) (*http.Response, []byte) {
	t.Helper()
	return bookingAuthedRequest(t, srv, http.MethodGet, path, token, "")
}

func invoiceCheckout(t *testing.T, srv *httptest.Server, token string, invoiceID int64) (*http.Response, []byte) {
	t.Helper()
	return bookingAuthedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/invoices/%d/checkout", invoiceID), token, "")
}

func getInitialInvoiceID(t *testing.T, pool *pgxpool.Pool, bookingID int64) int64 {
	t.Helper()
	var invoiceID int64
	if err := pool.QueryRow(context.Background(),
		`SELECT id FROM invoices WHERE booking_id = $1 AND type = 'initial'`, bookingID).Scan(&invoiceID); err != nil {
		t.Fatalf("read initial invoice id: %v", err)
	}
	return invoiceID
}

func approveBooking(t *testing.T, pool *pgxpool.Pool, bookingID int64) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `UPDATE bookings SET status = 'approved' WHERE id = $1`, bookingID); err != nil {
		t.Fatalf("approve booking: %v", err)
	}
}

func insertRenewalInvoice(t *testing.T, pool *pgxpool.Pool, bookingID, rentAmount, adminFee int64) int64 {
	t.Helper()
	today := time.Now()
	total := rentAmount + adminFee
	var invoiceID int64
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO invoices (booking_id, number, type, period_start, period_end, periods,
			rent_amount, deposit, admin_fee, total, due_date, issued_at)
		VALUES ($1, $2, 'renewal', $3::date, $3::date + interval '1 month', 1, $4, 0, $5, $6, $3::date, now())
		RETURNING id`,
		bookingID, fmt.Sprintf("INV/RENEWAL/%d", bookingID), today, rentAmount, adminFee, total,
	).Scan(&invoiceID); err != nil {
		t.Fatalf("insert renewal invoice: %v", err)
	}
	return invoiceID
}

func TestInvoiceListAndDetailForInitialInvoice(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, token := bookingRegisterAndVerify(t, srv)
	bookingID := submitBookingFixture(t, srv, token, roomID)
	invoiceID := getInitialInvoiceID(t, pool, bookingID)

	resp, body := invoiceAuthedGet(t, srv, token, "/api/invoices")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: status = %d, body = %s", resp.StatusCode, body)
	}
	var listParsed struct {
		Invoices []struct {
			ID   int64  `json:"id"`
			Type string `json:"type"`
		} `json:"invoices"`
	}
	if err := json.Unmarshal(body, &listParsed); err != nil {
		t.Fatalf("parse list body: %v", err)
	}
	found := false
	for _, inv := range listParsed.Invoices {
		if inv.ID == invoiceID {
			found = true
			if inv.Type != "initial" {
				t.Errorf("invoice.type = %q, want initial", inv.Type)
			}
		}
	}
	if !found {
		t.Errorf("initial invoice %d not found in list: %+v", invoiceID, listParsed.Invoices)
	}

	resp, body = invoiceAuthedGet(t, srv, token, fmt.Sprintf("/api/invoices/%d", invoiceID))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("detail: status = %d, body = %s", resp.StatusCode, body)
	}
	var detailParsed struct {
		Invoice struct {
			ID        int64  `json:"id"`
			BookingID int64  `json:"booking_id"`
			Total     int64  `json:"total"`
			Status    string `json:"status"`
		} `json:"invoice"`
		Payment any `json:"payment"`
		Refund  any `json:"refund"`
	}
	if err := json.Unmarshal(body, &detailParsed); err != nil {
		t.Fatalf("parse detail body: %v", err)
	}
	if detailParsed.Invoice.BookingID != bookingID || detailParsed.Invoice.Status != "unpaid" {
		t.Errorf("invoice = %+v, want booking_id=%d status=unpaid", detailParsed.Invoice, bookingID)
	}
	if detailParsed.Payment != nil {
		t.Errorf("payment = %v, want nil (no checkout yet)", detailParsed.Payment)
	}
}

func TestInvoiceDetailForbiddenForOtherTenant(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, tokenA := bookingRegisterAndVerify(t, srv)
	_, tokenB := bookingRegisterAndVerify(t, srv)
	bookingID := submitBookingFixture(t, srv, tokenA, roomID)
	invoiceID := getInitialInvoiceID(t, pool, bookingID)

	resp, body := invoiceAuthedGet(t, srv, tokenB, fmt.Sprintf("/api/invoices/%d", invoiceID))
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s, want 403", resp.StatusCode, body)
	}
}

func TestInvoiceCheckoutInitialBehavesLikeBookingCheckout(t *testing.T) {
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
	invoiceID := getInitialInvoiceID(t, pool, bookingID)

	resp, body := invoiceCheckout(t, srv, token, invoiceID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("checkout: status = %d, body = %s", resp.StatusCode, body)
	}

	var available int
	if err := pool.QueryRow(context.Background(), `SELECT available FROM rooms WHERE id = $1`, roomID).Scan(&available); err != nil {
		t.Fatalf("read room: %v", err)
	}
	if available != 1 {
		t.Errorf("room available = %d, want 1 (initial invoice checkout must still lock a room unit)", available)
	}
}

func TestInvoiceCheckoutRenewalDoesNotLockRoom(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	var lastGrossAmount int64
	mock := &midtrans.MockClient{
		CreateTransactionFunc: func(_ context.Context, req midtrans.TransactionRequest) (midtrans.TransactionResponse, error) {
			lastGrossAmount = req.TransactionDetails.GrossAmount
			return midtrans.TransactionResponse{Token: "snap-token", RedirectURL: "https://example.test/redirect"}, nil
		},
	}
	srv, pool := newTestServerAndPoolWithMidtrans(t, mock)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, token := bookingRegisterAndVerify(t, srv)
	bookingID := submitBookingFixture(t, srv, token, roomID)
	approveBooking(t, pool, bookingID)
	renewalID := insertRenewalInvoice(t, pool, bookingID, 2250000, 0)

	resp, body := invoiceCheckout(t, srv, token, renewalID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("checkout: status = %d, body = %s", resp.StatusCode, body)
	}
	if lastGrossAmount != 2250000 {
		t.Errorf("gross_amount sent to Midtrans = %d, want 2250000", lastGrossAmount)
	}

	var available int
	if err := pool.QueryRow(context.Background(), `SELECT available FROM rooms WHERE id = $1`, roomID).Scan(&available); err != nil {
		t.Fatalf("read room: %v", err)
	}
	if available != 2 {
		t.Errorf("room available = %d, want 2 (renewal checkout must not lock a room unit)", available)
	}
}

func TestInvoiceCheckoutNonUnpaidIs400(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, token := bookingRegisterAndVerify(t, srv)
	bookingID := submitBookingFixture(t, srv, token, roomID)
	approveBooking(t, pool, bookingID)
	renewalID := insertRenewalInvoice(t, pool, bookingID, 2250000, 0)
	if _, err := pool.Exec(context.Background(), `UPDATE invoices SET status = 'void' WHERE id = $1`, renewalID); err != nil {
		t.Fatalf("void invoice: %v", err)
	}

	resp, body := invoiceCheckout(t, srv, token, renewalID)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s, want 400", resp.StatusCode, body)
	}
}

func TestInvoicePDFForTenantIsPDF(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, token := bookingRegisterAndVerify(t, srv)
	bookingID := submitBookingFixture(t, srv, token, roomID)
	invoiceID := getInitialInvoiceID(t, pool, bookingID)

	resp, body := invoiceAuthedGet(t, srv, token, fmt.Sprintf("/api/invoices/%d/pdf", invoiceID))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("content-type = %q, want application/pdf", ct)
	}
	if len(body) < 4 || string(body[:4]) != "%PDF" {
		t.Errorf("body does not start with %%PDF: %q", body[:min(len(body), 20)])
	}
}

func TestInvoicePDFForbiddenForOtherTenant(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, tokenA := bookingRegisterAndVerify(t, srv)
	_, tokenB := bookingRegisterAndVerify(t, srv)
	bookingID := submitBookingFixture(t, srv, tokenA, roomID)
	invoiceID := getInitialInvoiceID(t, pool, bookingID)

	resp, body := invoiceAuthedGet(t, srv, tokenB, fmt.Sprintf("/api/invoices/%d/pdf", invoiceID))
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s, want 403", resp.StatusCode, body)
	}
}

func TestInvoicePDFPaidShowsLunasAndAllowsOwner(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, token := bookingRegisterAndVerify(t, srv)
	bookingID := submitBookingFixture(t, srv, token, roomID)
	invoiceID := getInitialInvoiceID(t, pool, bookingID)

	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE invoices SET status = 'paid', paid_at = now() WHERE id = $1`, invoiceID); err != nil {
		t.Fatalf("mark invoice paid: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO payments (invoice_id, order_id, status, payment_type, gross_amount, expires_at)
		VALUES ($1, $2, 'settlement', 'bank_transfer', 2250000, now() + interval '15 minutes')`,
		invoiceID, fmt.Sprintf("KOS-TEST-PAID-%d", invoiceID)); err != nil {
		t.Fatalf("insert settled payment: %v", err)
	}

	var ownerID int64
	if err := pool.QueryRow(ctx, `
		SELECT p.owner_id FROM rooms r JOIN properties p ON p.id = r.property_id WHERE r.id = $1`, roomID).Scan(&ownerID); err != nil {
		t.Fatalf("read owner id: %v", err)
	}
	ownerToken := mintTenantToken(t, pool, ownerID)

	resp, body := invoiceAuthedGet(t, srv, ownerToken, fmt.Sprintf("/api/invoices/%d/pdf", invoiceID))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("content-type = %q, want application/pdf", ct)
	}
}

func TestInvoicePDFVoidIs400(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, token := bookingRegisterAndVerify(t, srv)
	bookingID := submitBookingFixture(t, srv, token, roomID)
	invoiceID := getInitialInvoiceID(t, pool, bookingID)
	if _, err := pool.Exec(context.Background(), `UPDATE invoices SET status = 'void' WHERE id = $1`, invoiceID); err != nil {
		t.Fatalf("void invoice: %v", err)
	}

	resp, body := invoiceAuthedGet(t, srv, token, fmt.Sprintf("/api/invoices/%d/pdf", invoiceID))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s, want 400", resp.StatusCode, body)
	}
}

func TestInvoiceCheckoutBookingNotApprovedIs400(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, token := bookingRegisterAndVerify(t, srv)
	bookingID := submitBookingFixture(t, srv, token, roomID)
	renewalID := insertRenewalInvoice(t, pool, bookingID, 2250000, 0)

	resp, body := invoiceCheckout(t, srv, token, renewalID)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s, want 400 (booking is still pending, not approved)", resp.StatusCode, body)
	}
}
