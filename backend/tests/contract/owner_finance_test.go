package contract

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestOwnerBankAccount(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	ownerID, token := newOwner(t, srv, pool)

	resp, body := bookingAuthedRequest(t, srv, http.MethodGet, "/api/owner/bank-account", token, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get: status = %d, body = %s", resp.StatusCode, body)
	}
	AssertJSONMatches(t, body, []byte(`{
		"status": true, "meta": {"response_code": 200, "code": 200, "severity": "OK", "message": "OK"},
		"bank_account": null
	}`), nil)

	resp, body = bookingAuthedRequest(t, srv, http.MethodPut, "/api/owner/bank-account", token, `{"bank_name":"","account_number":"12ab","account_name":""}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("invalid put: status = %d, body = %s", resp.StatusCode, body)
	}
	AssertJSONMatches(t, body, []byte(`{
		"status": false, "meta": {"response_code": 422, "code": 422, "severity": "WARNING", "message": "Validasi gagal."},
		"messages": {
			"bank_name": ["The bank name field is required."],
			"account_number": ["The account number field format is invalid."],
			"account_name": ["The account name field is required."]
		}
	}`), nil)

	resp, body = bookingAuthedRequest(t, srv, http.MethodPut, "/api/owner/bank-account", token, `{"bank_name":"BCA","account_number":"1234567890","account_name":"Pemilik Contoh"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put: status = %d, body = %s", resp.StatusCode, body)
	}
	want := `{
		"status": true, "meta": {"response_code": 200, "code": 200, "severity": "OK", "message": "Rekening disimpan."},
		"bank_account": {"bank_name": "BCA", "account_number": "1234567890", "account_name": "Pemilik Contoh"}
	}`
	AssertJSONMatches(t, body, []byte(want), nil)
	_, body = bookingAuthedRequest(t, srv, http.MethodGet, "/api/owner/bank-account", token, "")
	AssertJSONMatches(t, body, []byte(strings.Replace(want, "Rekening disimpan.", "OK", 1)), nil)

	var changes string
	if err := pool.QueryRow(context.Background(), `SELECT changes::text FROM audit_logs WHERE entity_type = 'owner_bank_account' AND entity_id = $1 ORDER BY id DESC LIMIT 1`, ownerID).Scan(&changes); err != nil {
		t.Fatalf("load audit: %v", err)
	}
	if strings.Contains(changes, "1234567890") || !strings.Contains(changes, `"account_number": ["***", "***"]`) {
		t.Errorf("audit changes leak or miss the masked account number: %s", changes)
	}
}

func TestOwnerPayoutsAndInvoices(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	_, ownerToken, roomID, tenantToken, _, _ := newOwnerBookingFixture(t, srv, pool)
	paidID := submitBookingFixture(t, srv, tenantToken, roomID)
	unpaidID := submitBookingFixture(t, srv, tenantToken, roomID)
	markBookingPaid(t, pool, paidID, roomID, 24*time.Hour)
	if resp, body := bookingAuthedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/owner/bookings/%d/approve", paidID), ownerToken, ""); resp.StatusCode != http.StatusOK {
		t.Fatalf("approve: status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body := bookingAuthedRequest(t, srv, http.MethodGet, "/api/owner/payouts?status=pending", ownerToken, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("payouts: status = %d, body = %s", resp.StatusCode, body)
	}
	AssertJSONMatches(t, body, []byte(fmt.Sprintf(`{
		"status": true, "meta": {"response_code": 200, "code": 200, "severity": "OK", "message": "OK"},
		"payouts": [
			{"id": 0, "booking_id": %d, "amount": 3000000, "amount_label": "Rp3.000.000",
			 "status": "pending", "transfer_reference": null, "completed_at": null, "created_at": ""}
		],
		"current_page": 1
	}`, paidID)), map[string]*regexp.Regexp{
		"$.payouts[0].id":         regexp.MustCompile(`^\d+$`),
		"$.payouts[0].created_at": regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$`),
	})

	resp, body = bookingAuthedRequest(t, srv, http.MethodGet, "/api/owner/invoices", ownerToken, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("invoices: status = %d, body = %s", resp.StatusCode, body)
	}
	var list struct {
		Invoices []struct {
			BookingID int64  `json:"booking_id"`
			Type      string `json:"type"`
			Status    string `json:"status"`
			Total     int64  `json:"total"`
		} `json:"invoices"`
	}
	mustUnmarshal(t, body, &list)
	if len(list.Invoices) != 1 || list.Invoices[0].BookingID != paidID || list.Invoices[0].Type != "initial" ||
		list.Invoices[0].Status != "paid" || list.Invoices[0].Total != 3005000 {
		t.Errorf("owner invoices = %s, want only the paid booking's initial invoice (unpaid booking %d hidden)", body, unpaidID)
	}

	_, otherToken := newOwner(t, srv, pool)
	_, body = bookingAuthedRequest(t, srv, http.MethodGet, fmt.Sprintf("/api/owner/invoices?booking_id=%d", paidID), otherToken, "")
	if !strings.Contains(string(body), `"invoices":[]`) {
		t.Errorf("other owner sees invoices: %s", body)
	}
}

func TestOwnerFinanceSummaryAndLedger(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	_, ownerToken, roomID, tenantToken, _, _ := newOwnerBookingFixture(t, srv, pool)
	bookingID := submitBookingFixture(t, srv, tenantToken, roomID)
	markBookingPaid(t, pool, bookingID, roomID, 24*time.Hour)
	if resp, body := bookingAuthedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/owner/bookings/%d/approve", bookingID), ownerToken, ""); resp.StatusCode != http.StatusOK {
		t.Fatalf("approve: status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body := bookingAuthedRequest(t, srv, http.MethodGet, "/api/owner/finance/summary", ownerToken, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("summary: status = %d, body = %s", resp.StatusCode, body)
	}
	var summary struct {
		Summary struct {
			Earnings         int64 `json:"earnings"`
			PayoutsCompleted int64 `json:"payouts_completed"`
			PendingBalance   int64 `json:"pending_balance"`
			DepositsHeld     int64 `json:"deposits_held"`
			PaidInvoices     int64 `json:"paid_invoices"`
		} `json:"summary"`
	}
	mustUnmarshal(t, body, &summary)
	if summary.Summary.Earnings != 3000000 || summary.Summary.PayoutsCompleted != 0 || summary.Summary.PendingBalance != 3000000 ||
		summary.Summary.DepositsHeld != 1500000 || summary.Summary.PaidInvoices != 1 {
		t.Errorf("summary = %+v, want earnings=3000000 payouts_completed=0 pending_balance=3000000 deposits_held=1500000 paid_invoices=1", summary.Summary)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodGet, "/api/owner/finance/ledger?type=owner_earning", ownerToken, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ledger: status = %d, body = %s", resp.StatusCode, body)
	}
	var ledger struct {
		Entries []struct {
			Type          string `json:"type"`
			Amount        int64  `json:"amount"`
			BookingID     int64  `json:"booking_id"`
			InvoiceNumber string `json:"invoice_number"`
		} `json:"entries"`
	}
	mustUnmarshal(t, body, &ledger)
	if len(ledger.Entries) != 1 || ledger.Entries[0].Type != "owner_earning" || ledger.Entries[0].Amount != 3000000 ||
		ledger.Entries[0].BookingID != bookingID || ledger.Entries[0].InvoiceNumber == "" {
		t.Errorf("ledger entries = %+v", ledger.Entries)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodGet, "/api/owner/finance/ledger.csv", ownerToken, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("csv: status = %d, body = %s", resp.StatusCode, body)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Errorf("content-type = %q, want text/csv", ct)
	}
	lines := strings.Split(strings.TrimRight(string(body), "\n"), "\n")
	if lines[0] != "id,occurred_at_wib,type,amount,booking_id,invoice_number,description" {
		t.Errorf("csv header = %q", lines[0])
	}
	if !strings.Contains(string(body), "owner_earning") || !strings.Contains(string(body), strconv.FormatInt(bookingID, 10)) {
		t.Errorf("csv body missing expected row: %s", body)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodGet, "/api/owner/finance/summary?from=2020-01-01&to=2022-01-01", ownerToken, "")
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("range > 366 days: status = %d, body = %s, want 422", resp.StatusCode, body)
	}
}

func TestOwnerStats(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	_, ownerToken, roomID, tenantToken, _, _ := newOwnerBookingFixture(t, srv, pool)
	bookingID := submitBookingFixture(t, srv, tenantToken, roomID)
	markBookingPaid(t, pool, bookingID, roomID, 24*time.Hour)
	if resp, body := bookingAuthedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/owner/bookings/%d/approve", bookingID), ownerToken, ""); resp.StatusCode != http.StatusOK {
		t.Fatalf("approve: status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body := bookingAuthedRequest(t, srv, http.MethodGet, "/api/owner/stats", ownerToken, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stats: status = %d, body = %s", resp.StatusCode, body)
	}
	var stats struct {
		Stats struct {
			BookingsPaid     int64 `json:"bookings_paid"`
			BookingsApproved int64 `json:"bookings_approved"`
			Earnings         int64 `json:"earnings"`
			Occupancy        struct {
				Occupied int64   `json:"occupied"`
				Total    int64   `json:"total"`
				Rate     float64 `json:"rate"`
			} `json:"occupancy"`
			Rating struct {
				Average float64 `json:"average"`
				Count   int64   `json:"count"`
			} `json:"rating"`
		} `json:"stats"`
	}
	mustUnmarshal(t, body, &stats)
	if stats.Stats.BookingsPaid != 1 || stats.Stats.BookingsApproved != 1 || stats.Stats.Earnings != 3000000 {
		t.Errorf("stats = %+v, want bookings_paid=1 bookings_approved=1 earnings=3000000", stats.Stats)
	}
	if stats.Stats.Occupancy.Occupied != 1 || stats.Stats.Occupancy.Total != 3 {
		t.Errorf("occupancy = %+v, want occupied=1 total=3 (available 2 + occupied 1)", stats.Stats.Occupancy)
	}
	if stats.Stats.Rating.Count != 0 || stats.Stats.Rating.Average != 0 {
		t.Errorf("rating = %+v, want zero (no reviews yet)", stats.Stats.Rating)
	}
}
