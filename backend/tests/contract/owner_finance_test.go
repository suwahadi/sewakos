package contract

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
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
