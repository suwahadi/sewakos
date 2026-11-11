package contract

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/suwahadi/sewakos/backend/internal/money"
)

func TestExtendRejectsInvalidRequests(t *testing.T) {
	srv, pool := newTestServerAndPool(t)
	ctx := context.Background()
	start := money.TodayWIB().AddDate(0, 0, 2)
	f := insertApprovedMonthlyBooking(t, pool, start.Format(time.DateOnly), 1)
	token := mintTenantToken(t, pool, f.tenantID)
	path := fmt.Sprintf("/api/booking/%d/extend", f.bookingID)

	for _, body := range []string{`{}`, `{"duration":2}`, `{"duration":"tiga"}`} {
		resp, respBody := bookingAuthedRequest(t, srv, http.MethodPost, path, token, body)
		if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(respBody), "The selected duration is invalid.") {
			t.Errorf("extend %s: status = %d, body = %s, want 422", body, resp.StatusCode, respBody)
		}
	}

	other := insertApprovedMonthlyBooking(t, pool, start.Format(time.DateOnly), 1)
	if resp, respBody := bookingAuthedRequest(t, srv, http.MethodPost, path, mintTenantToken(t, pool, other.tenantID), `{"duration":1}`); resp.StatusCode != http.StatusForbidden {
		t.Errorf("another tenant: status = %d, body = %s, want 403", resp.StatusCode, respBody)
	}
	if resp, respBody := bookingAuthedRequest(t, srv, http.MethodPost, path, mintTenantToken(t, pool, f.ownerID), `{"duration":1}`); resp.StatusCode != http.StatusForbidden {
		t.Errorf("owner role: status = %d, body = %s, want 403", resp.StatusCode, respBody)
	}
	if resp, _ := bookingAuthedRequest(t, srv, http.MethodPost, "/api/booking/999999999/extend", token, `{"duration":1}`); resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown booking: status = %d, want 404", resp.StatusCode)
	}

	if _, err := pool.Exec(ctx, `UPDATE bookings SET status = 'paid' WHERE id = $1`, f.bookingID); err != nil {
		t.Fatal(err)
	}
	resp, respBody := bookingAuthedRequest(t, srv, http.MethodPost, path, token, `{"duration":1}`)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(respBody), "Sewa tidak dapat diperpanjang.") {
		t.Errorf("not approved: status = %d, body = %s, want 400", resp.StatusCode, respBody)
	}

	if _, err := pool.Exec(ctx, `UPDATE bookings SET status = 'approved', start_date = start_date - 40, end_date = current_date - 1 WHERE id = $1`, f.bookingID); err != nil {
		t.Fatal(err)
	}
	resp, respBody = bookingAuthedRequest(t, srv, http.MethodPost, path, token, `{"duration":1}`)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(respBody), "Sewa tidak dapat diperpanjang.") {
		t.Errorf("past end_date: status = %d, body = %s, want 400", resp.StatusCode, respBody)
	}
}

func TestInvoicePDFFilenameAdminAccessAndVoidMessage(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, token := bookingRegisterAndVerify(t, srv)
	bookingID := submitBookingFixture(t, srv, token, roomID)
	invoiceID := getInitialInvoiceID(t, pool, bookingID)
	path := fmt.Sprintf("/api/invoices/%d/pdf", invoiceID)

	resp, body := invoiceAuthedGet(t, srv, adminToken(t, srv), path)
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "application/pdf" {
		t.Fatalf("admin: status = %d, content-type = %q, body = %.40s", resp.StatusCode, resp.Header.Get("Content-Type"), body)
	}
	filename := regexp.MustCompile(fmt.Sprintf(`^attachment; filename="INV-\d{4}-\d{2}-%06d\.pdf"$`, invoiceID))
	if got := resp.Header.Get("Content-Disposition"); !filename.MatchString(got) {
		t.Errorf("content-disposition = %q, want %s", got, filename)
	}

	if resp, _ := invoiceAuthedGet(t, srv, token, "/api/invoices/999999999/pdf"); resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown invoice: status = %d, want 404", resp.StatusCode)
	}

	if _, err := pool.Exec(context.Background(), `UPDATE invoices SET status = 'void' WHERE id = $1`, invoiceID); err != nil {
		t.Fatal(err)
	}
	resp, body = invoiceAuthedGet(t, srv, token, path)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "Tagihan sudah dibatalkan.") ||
		!strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		t.Errorf("void: status = %d, content-type = %q, body = %s", resp.StatusCode, resp.Header.Get("Content-Type"), body)
	}
}

func TestFinanceRangeLimitsAndCSVHeaders(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	_, ownerToken := newOwner(t, srv, pool)
	admin := adminToken(t, srv)
	_, tenantToken := bookingRegisterAndVerify(t, srv)
	const tooLong = "?from=2024-01-01&to=2025-01-01"
	const longestAllowed = "?from=2024-01-01&to=2024-12-31"

	for _, tc := range []struct{ path, token string }{
		{"/api/owner/finance/summary", ownerToken},
		{"/api/owner/finance/ledger", ownerToken},
		{"/api/owner/finance/ledger.csv", ownerToken},
		{"/api/owner/stats", ownerToken},
		{"/api/admin/finance/summary", admin},
		{"/api/admin/finance/ledger", admin},
		{"/api/admin/finance/ledger.csv", admin},
	} {
		resp, body := bookingAuthedRequest(t, srv, http.MethodGet, tc.path+tooLong, tc.token, "")
		if resp.StatusCode != http.StatusUnprocessableEntity || !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
			t.Errorf("%s over 366 days: status = %d, content-type = %q, body = %s", tc.path, resp.StatusCode, resp.Header.Get("Content-Type"), body)
		}
		if resp, body := bookingAuthedRequest(t, srv, http.MethodGet, tc.path+longestAllowed, tc.token, ""); resp.StatusCode != http.StatusOK {
			t.Errorf("%s exactly 366 days: status = %d, body = %s", tc.path, resp.StatusCode, body)
		}
		if resp, _ := bookingAuthedRequest(t, srv, http.MethodGet, tc.path, tenantToken, ""); resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s as tenant: status = %d, want 403", tc.path, resp.StatusCode)
		}
	}

	for _, tc := range []struct{ path, token, header string }{
		{"/api/owner/finance/ledger.csv", ownerToken, "id,occurred_at_wib,type,amount,booking_id,invoice_number,description"},
		{"/api/admin/finance/ledger.csv", admin, "id,occurred_at_wib,type,amount,owner_id,booking_id,invoice_number,description"},
	} {
		resp, body := bookingAuthedRequest(t, srv, http.MethodGet, tc.path+"?from=2024-02-01&to=2024-02-29", tc.token, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: status = %d, body = %s", tc.path, resp.StatusCode, body)
		}
		if got := resp.Header.Get("Content-Type"); got != "text/csv; charset=utf-8" {
			t.Errorf("%s content-type = %q", tc.path, got)
		}
		if got := resp.Header.Get("Content-Disposition"); got != `attachment; filename="ledger-2024-02-01-2024-02-29.csv"` {
			t.Errorf("%s content-disposition = %q", tc.path, got)
		}
		if got := strings.TrimSpace(string(body)); got != tc.header {
			t.Errorf("%s empty range body = %q, want only the header row", tc.path, got)
		}
	}
}

func TestOwnerAuditLogMasksBankAccountNumber(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	_, ownerToken := newOwner(t, srv, pool)
	const accountNumber = "5550001112223"

	if resp, body := bookingAuthedRequest(t, srv, http.MethodPut, "/api/owner/bank-account", ownerToken,
		fmt.Sprintf(`{"bank_name":"BCA","account_number":%q,"account_name":"Pemilik Uji"}`, accountNumber)); resp.StatusCode != http.StatusOK {
		t.Fatalf("put bank account: status = %d, body = %s", resp.StatusCode, body)
	}
	resp, body := bookingAuthedRequest(t, srv, http.MethodGet, "/api/owner/audit-logs?entity_type=owner_bank_account", ownerToken, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("audit logs: status = %d, body = %s", resp.StatusCode, body)
	}
	if strings.Contains(string(body), accountNumber) || !strings.Contains(string(body), `"***"`) {
		t.Errorf("audit log must mask the account number: %s", body)
	}
	if strings.Contains(string(body), `"ip"`) || strings.Contains(string(body), `"user_agent"`) {
		t.Errorf("owner audit log must not expose ip or user_agent: %s", body)
	}
}
