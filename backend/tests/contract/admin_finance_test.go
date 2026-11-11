package contract

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/suwahadi/sewakos/backend/internal/midtrans"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/worker"
)

func adminFinanceSummary(t *testing.T, srv *httptest.Server, token string) (paymentsReceived, adminFeeRevenue int64) {
	t.Helper()
	resp, body := bookingAuthedRequest(t, srv, http.MethodGet, "/api/admin/finance/summary", token, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("summary: status = %d, body = %s", resp.StatusCode, body)
	}
	var summary struct {
		Summary struct {
			PaymentsReceived int64 `json:"payments_received"`
			AdminFeeRevenue  int64 `json:"admin_fee_revenue"`
		} `json:"summary"`
	}
	mustUnmarshal(t, body, &summary)
	return summary.Summary.PaymentsReceived, summary.Summary.AdminFeeRevenue
}

func setAdminFee(t *testing.T, srv *httptest.Server, token string, fee int64) {
	t.Helper()
	resp, body := bookingAuthedRequest(t, srv, http.MethodPatch, "/api/admin/settings", token, fmt.Sprintf(`{"admin_fee":%d}`, fee))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("set admin_fee: status = %d, body = %s", resp.StatusCode, body)
	}
}

func TestAdminFinanceSummaryAndLedger(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	t.Setenv("MIDTRANS_SERVER_KEY", webhookTestServerKey)
	const adminFee = int64(5000)
	var grossAmount int64
	mock := &midtrans.MockClient{
		CreateTransactionFunc: func(_ context.Context, req midtrans.TransactionRequest) (midtrans.TransactionResponse, error) {
			grossAmount = req.TransactionDetails.GrossAmount
			return midtrans.TransactionResponse{Token: "snap-token", RedirectURL: "https://example.test/redirect"}, nil
		},
		GetStatusFunc: func(_ context.Context, orderID string) (midtrans.StatusResponse, error) {
			return midtrans.StatusResponse{OrderID: orderID, TransactionStatus: "settlement", FraudStatus: "accept", GrossAmount: grossAmountString(grossAmount)}, nil
		},
	}
	srv, pool := newTestServerAndPoolWithMidtrans(t, mock)
	token := adminToken(t, srv)
	setAdminFee(t, srv, token, adminFee)
	t.Cleanup(func() { setAdminFee(t, srv, token, 0) })
	paymentsBefore, adminFeeBefore := adminFinanceSummary(t, srv, token)

	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, tenantToken := bookingRegisterAndVerify(t, srv)
	bookingID := submitBookingFixture(t, srv, tenantToken, roomID)
	resp, body := checkoutBooking(t, srv, tenantToken, bookingID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("checkout: status = %d, body = %s", resp.StatusCode, body)
	}
	var checkout struct {
		Payment struct {
			OrderID     string `json:"order_id"`
			GrossAmount int64  `json:"gross_amount"`
		} `json:"payment"`
	}
	mustUnmarshal(t, body, &checkout)
	if resp, body := postWebhookNotification(t, srv, checkout.Payment.OrderID, "200", grossAmountString(checkout.Payment.GrossAmount)); resp.StatusCode != http.StatusOK {
		t.Fatalf("webhook: status = %d, body = %s", resp.StatusCode, body)
	}

	paymentsAfter, adminFeeAfter := adminFinanceSummary(t, srv, token)
	if paymentsAfter-paymentsBefore != checkout.Payment.GrossAmount || adminFeeAfter-adminFeeBefore != adminFee {
		t.Errorf("summary delta = payments %d admin_fee %d, want %d / %d",
			paymentsAfter-paymentsBefore, adminFeeAfter-adminFeeBefore, checkout.Payment.GrossAmount, adminFee)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodGet, "/api/admin/finance/ledger?type=payment_received", token, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ledger: status = %d, body = %s", resp.StatusCode, body)
	}
	var ledger struct {
		Entries []struct {
			Type      string `json:"type"`
			Amount    int64  `json:"amount"`
			BookingID int64  `json:"booking_id"`
		} `json:"entries"`
	}
	mustUnmarshal(t, body, &ledger)
	var found bool
	for _, entry := range ledger.Entries {
		if entry.BookingID == bookingID {
			found = true
			if entry.Amount != checkout.Payment.GrossAmount {
				t.Errorf("entry for booking %d: amount = %d, want %d", bookingID, entry.Amount, checkout.Payment.GrossAmount)
			}
		}
	}
	if !found {
		t.Errorf("ledger entries %+v missing booking %d", ledger.Entries, bookingID)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodGet, "/api/admin/finance/ledger.csv", token, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("csv: status = %d, body = %s", resp.StatusCode, body)
	}
	lines := strings.Split(strings.TrimRight(string(body), "\n"), "\n")
	if lines[0] != "id,occurred_at_wib,type,amount,owner_id,booking_id,invoice_number,description" {
		t.Errorf("csv header = %q", lines[0])
	}
	if !strings.Contains(string(body), strconv.FormatInt(bookingID, 10)) {
		t.Errorf("csv body missing booking id: %s", body)
	}
}

func TestAdminReconciliationResolve(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	token := adminToken(t, srv)
	ctx := context.Background()
	f := insertApprovedMonthlyBooking(t, pool, "2024-01-10", 1)

	var paymentID int64
	if err := pool.QueryRow(ctx, `
		SELECT id FROM payments p JOIN invoices i ON i.id = p.invoice_id WHERE i.booking_id = $1 LIMIT 1`, f.bookingID).Scan(&paymentID); err != nil {
		if err := pool.QueryRow(ctx, `
			INSERT INTO payments (invoice_id, order_id, status, gross_amount, expires_at)
			SELECT id, $2, 'settlement', 4000000, now() FROM invoices WHERE booking_id = $1 LIMIT 1
			RETURNING id`, f.bookingID, fmt.Sprintf("KOS-RECON-%d", f.bookingID)).Scan(&paymentID); err != nil {
			t.Fatalf("insert payment: %v", err)
		}
	}
	var issueID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO reconciliation_issues (payment_id, kind, local_status, midtrans_status, local_amount, midtrans_amount)
		VALUES ($1, 'status_mismatch', 'pending', 'settlement', 2250000, 2250000)
		RETURNING id`, paymentID).Scan(&issueID); err != nil {
		t.Fatalf("insert reconciliation issue: %v", err)
	}

	resp, body := bookingAuthedRequest(t, srv, http.MethodGet, "/api/admin/finance/reconciliation?status=open", token, "")
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"kind":"status_mismatch"`) {
		t.Fatalf("list: status = %d, body = %s", resp.StatusCode, body)
	}

	resolvePath := fmt.Sprintf("/api/admin/finance/reconciliation/%d/resolve", issueID)
	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, resolvePath, token, `{}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("missing note: status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, resolvePath, token, `{"note":"Diproses ulang manual."}`)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "Selisih ditandai selesai.") {
		t.Fatalf("resolve: status = %d, body = %s", resp.StatusCode, body)
	}

	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM reconciliation_issues WHERE id = $1`, issueID).Scan(&status); err != nil || status != "resolved" {
		t.Errorf("issue status = %q (err %v), want resolved", status, err)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, resolvePath, token, `{"note":"Lagi."}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("already resolved: status = %d, body = %s", resp.StatusCode, body)
	}
}

func TestReconcilePaymentsDetectsStatusMismatch(t *testing.T) {
	ctx := context.Background()
	_, pool := newTestServerAndPool(t)
	f := insertApprovedMonthlyBooking(t, pool, "2024-01-10", 1)

	orderID := fmt.Sprintf("KOS-RECON-MISMATCH-%d", f.bookingID)
	var paymentID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO payments (invoice_id, order_id, status, gross_amount, expires_at, updated_at)
		SELECT id, $2, 'pending', 4000000, now() - interval '1 hour', now() FROM invoices WHERE booking_id = $1 LIMIT 1
		RETURNING id`, f.bookingID, orderID).Scan(&paymentID); err != nil {
		t.Fatalf("insert payment: %v", err)
	}

	mock := &midtrans.MockClient{
		GetStatusFunc: func(ctx context.Context, gotOrderID string) (midtrans.StatusResponse, error) {
			if gotOrderID == orderID {
				return midtrans.StatusResponse{OrderID: gotOrderID, TransactionStatus: "settlement", FraudStatus: "accept", GrossAmount: "4000000.00"}, nil
			}
			var localStatus string
			var grossAmount int64
			if err := pool.QueryRow(ctx, `SELECT status, gross_amount FROM payments WHERE order_id = $1`, gotOrderID).Scan(&localStatus, &grossAmount); err != nil {
				return midtrans.StatusResponse{}, err
			}
			return midtrans.StatusResponse{OrderID: gotOrderID, TransactionStatus: localStatus, GrossAmount: fmt.Sprintf("%d.00", grossAmount)}, nil
		},
	}
	deps := tenancyDeps(pool)
	deps.Booking.Midtrans = mock
	worker.ReconcilePayments(ctx, deps, discardLogger(), money.NowWIB())

	var kind, localStatus, midtransStatus string
	if err := pool.QueryRow(ctx, `
		SELECT kind, local_status, midtrans_status FROM reconciliation_issues WHERE payment_id = $1`, paymentID).
		Scan(&kind, &localStatus, &midtransStatus); err != nil {
		t.Fatalf("load reconciliation issue: %v", err)
	}
	if kind != "status_mismatch" || localStatus != "pending" || midtransStatus != "settlement" {
		t.Errorf("issue = kind=%q local=%q midtrans=%q, want status_mismatch/pending/settlement", kind, localStatus, midtransStatus)
	}

	var paymentStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM payments WHERE id = $1`, paymentID).Scan(&paymentStatus); err != nil || paymentStatus != "settlement" {
		t.Errorf("payment status = %q (err %v), want settlement (ApplyPaymentStatus should have caught it up)", paymentStatus, err)
	}
}
