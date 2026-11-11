package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type adminListRow struct {
	id     int64
	status string
}

func adminListIDs(t *testing.T, srv *httptest.Server, token, path, field string) []adminListRow {
	t.Helper()
	resp, body := bookingAuthedRequest(t, srv, http.MethodGet, path, token, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s: status = %d, body = %s", path, resp.StatusCode, body)
	}
	var envelope map[string]json.RawMessage
	var list []struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	if err := json.Unmarshal(envelope[field], &list); err != nil {
		t.Fatalf("parse %s.%s: %v", path, field, err)
	}
	rows := make([]adminListRow, len(list))
	for i, row := range list {
		rows[i] = adminListRow{row.ID, row.Status}
	}
	return rows
}

func TestAdminPropertyModeration(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	token := adminToken(t, srv)
	ownerID, _ := newOwner(t, srv, pool)
	propertyID := newOwnerProperty(t, pool, ownerID)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE properties SET moderation_status = 'pending', name = $2 WHERE id = $1`, propertyID, fmt.Sprintf("Kost Moderasi %d", propertyID)); err != nil {
		t.Fatal(err)
	}
	base := fmt.Sprintf("/api/admin/properties/%d", propertyID)

	resp, body := bookingAuthedRequest(t, srv, http.MethodGet, fmt.Sprintf("/api/admin/properties?moderation_status=pending&q=Moderasi%%20%d", propertyID), token, "")
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), fmt.Sprintf(`"owner":{"id":%d`, ownerID)) {
		t.Fatalf("list pending: status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, base+"/reject", token, `{}`)
	if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(body), "The reason field is required.") {
		t.Fatalf("reject without reason: status = %d, body = %s", resp.StatusCode, body)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, base+"/reject", token, `{"reason":"Foto kurang jelas"}`)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"Properti ditolak."`) {
		t.Fatalf("reject: status = %d, body = %s", resp.StatusCode, body)
	}
	if title, notice := latestNotification(t, pool, ownerID); title != "Properti ditolak" || !strings.HasSuffix(notice, "Alasan: Foto kurang jelas") {
		t.Errorf("owner notification = %q / %q", title, notice)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, base+"/approve", token, "")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "Status properti tidak sesuai untuk tindakan ini.") {
		t.Errorf("approve rejected property: status = %d, body = %s", resp.StatusCode, body)
	}

	if _, err := pool.Exec(ctx, `UPDATE properties SET moderation_status = 'pending' WHERE id = $1`, propertyID); err != nil {
		t.Fatal(err)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, base+"/approve", token, "")
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"Properti disetujui."`) {
		t.Fatalf("approve: status = %d, body = %s", resp.StatusCode, body)
	}
	_, body = bookingAuthedRequest(t, srv, http.MethodGet, base, token, "")
	if !strings.Contains(string(body), `"moderation_status":"approved"`) || !strings.Contains(string(body), `"moderation_note":null`) {
		t.Errorf("approved property = %s", body)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, base+"/suspend", token, `{"reason":"Laporan penipuan"}`)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"Properti dinonaktifkan."`) {
		t.Fatalf("suspend: status = %d, body = %s", resp.StatusCode, body)
	}
	if resp, _ := discoveryGetJSON(t, srv, fmt.Sprintf("/api/listings/%d", propertyID)); resp.StatusCode != http.StatusNotFound {
		t.Errorf("suspended listing detail: status = %d, want 404", resp.StatusCode)
	}
	if resp, _ := bookingAuthedRequest(t, srv, http.MethodPost, base+"/approve", token, ""); resp.StatusCode != http.StatusOK {
		t.Errorf("approve suspended property: status = %d", resp.StatusCode)
	}
}

func TestAdminBookingsAndInvoices(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	token := adminToken(t, srv)
	_, ownerToken, roomID, tenantToken, _, tenantPhone := newOwnerBookingFixture(t, srv, pool)
	paidID := submitBookingFixture(t, srv, tenantToken, roomID)
	unpaidID := submitBookingFixture(t, srv, tenantToken, roomID)
	markBookingPaid(t, pool, paidID, roomID, 24*time.Hour)
	if resp, body := bookingAuthedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/owner/bookings/%d/approve", paidID), ownerToken, ""); resp.StatusCode != http.StatusOK {
		t.Fatalf("approve: status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body := bookingAuthedRequest(t, srv, http.MethodGet, fmt.Sprintf("/api/admin/bookings/%d", paidID), token, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("detail: status = %d, body = %s", resp.StatusCode, body)
	}
	for _, fragment := range []string{
		fmt.Sprintf(`"phone_number":%q`, tenantPhone), `"admin_fee":5000`, `"admin_fee_label":"Rp5.000"`,
		`"payment":null`, `"refund":null`, `"payout":{`, `"status":"pending"`,
	} {
		if !strings.Contains(string(body), fragment) {
			t.Errorf("admin booking detail lacks %s: %s", fragment, body)
		}
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodGet, fmt.Sprintf("/api/admin/bookings/%d", unpaidID), token, "")
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"status":"pending"`) {
		t.Errorf("unpaid booking visible to admin: status = %d, body = %s", resp.StatusCode, body)
	}
	for status, want := range map[string]int64{"pending": unpaidID, "approved": paidID} {
		ids := adminListIDs(t, srv, token, "/api/admin/bookings?status="+status+"&page=1&limit=5", "bookings")
		if len(ids) == 0 || ids[0].id != want {
			t.Errorf("admin bookings status=%s = %+v, want newest first starting with %d", status, ids, want)
		}
		for _, row := range ids {
			if row.status != status {
				t.Errorf("admin bookings status=%s returned %+v", status, row)
			}
		}
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodGet, "/api/admin/bookings", ownerToken, "")
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("owner on admin route: status = %d, body = %s, want 403", resp.StatusCode, body)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodGet, fmt.Sprintf("/api/admin/invoices?booking_id=%d&type=initial", paidID), token, "")
	if resp.StatusCode != http.StatusOK || strings.Count(string(body), `"number":`) != 1 {
		t.Errorf("admin invoices: status = %d, body = %s", resp.StatusCode, body)
	}
}

func TestAdminRefundAndPayoutCompletion(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	token := adminToken(t, srv)
	ownerID, ownerToken, roomID, tenantToken, tenantID, _ := newOwnerBookingFixture(t, srv, pool)
	rejectedID := submitBookingFixture(t, srv, tenantToken, roomID)
	approvedID := submitBookingFixture(t, srv, tenantToken, roomID)
	markBookingPaid(t, pool, rejectedID, roomID, 24*time.Hour)
	markBookingPaid(t, pool, approvedID, roomID, 24*time.Hour)
	bookingAuthedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/owner/bookings/%d/reject", rejectedID), ownerToken, `{}`)
	bookingAuthedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/owner/bookings/%d/approve", approvedID), ownerToken, "")
	ctx := context.Background()

	var refundID, payoutID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM refunds WHERE booking_id = $1`, rejectedID).Scan(&refundID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM payouts WHERE booking_id = $1`, approvedID).Scan(&payoutID); err != nil {
		t.Fatal(err)
	}
	refundPath := fmt.Sprintf("/api/admin/refunds/%d/complete", refundID)
	payoutPath := fmt.Sprintf("/api/admin/payouts/%d/complete", payoutID)
	refundListed := func(status string) bool {
		var before int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM refunds WHERE status = $1 AND (created_at, id) < (SELECT created_at, id FROM refunds WHERE id = $2)`,
			status, refundID).Scan(&before); err != nil {
			t.Fatal(err)
		}
		rows := adminListIDs(t, srv, token, fmt.Sprintf("/api/admin/refunds?status=%s&limit=50&page=%d", status, before/50+1), "refunds")
		found := false
		for _, row := range rows {
			found = found || row.id == refundID
			if row.status != status {
				t.Errorf("refunds status=%s returned %+v", status, row)
			}
		}
		return found
	}
	if !refundListed("awaiting_account") {
		t.Errorf("refund %d missing from awaiting_account list (oldest first)", refundID)
	}

	resp, body := bookingAuthedRequest(t, srv, http.MethodPost, refundPath, token, `{"transfer_reference":"TRX-1"}`)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "Refund belum siap diproses.") {
		t.Errorf("refund awaiting account: status = %d, body = %s", resp.StatusCode, body)
	}
	if resp, body := bookingAuthedRequest(t, srv, http.MethodPut, fmt.Sprintf("/api/booking/%d/refund-account", rejectedID), tenantToken,
		`{"bank_name":"BCA","account_number":"1234567890","account_name":"Penyewa"}`); resp.StatusCode != http.StatusOK {
		t.Fatalf("refund account: status = %d, body = %s", resp.StatusCode, body)
	}
	if !refundListed("ready") {
		t.Errorf("refund %d missing from ready list after the tenant added an account", refundID)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, refundPath, token, `{}`)
	if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(body), "The transfer reference field is required.") {
		t.Errorf("refund without reference: status = %d, body = %s", resp.StatusCode, body)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, refundPath, token, `{"transfer_reference":"TRX-20261110-001","note":"Transfer BCA"}`)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"Refund ditandai selesai."`) ||
		!strings.Contains(string(body), `"status":"completed"`) || !strings.Contains(string(body), `"account_number":"1234567890"`) {
		t.Fatalf("complete refund: status = %d, body = %s", resp.StatusCode, body)
	}
	var refundLedger int64
	if err := pool.QueryRow(ctx, `SELECT amount FROM ledger_entries WHERE refund_id = $1 AND type = 'refund_completed'`, refundID).Scan(&refundLedger); err != nil || refundLedger >= 0 {
		t.Errorf("refund_completed ledger = %d (err %v), want negative", refundLedger, err)
	}
	if title, _ := latestNotification(t, pool, tenantID); title != "Pengembalian dana selesai" {
		t.Errorf("tenant notification = %q", title)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, payoutPath, token, `{"transfer_reference":"TRX-2"}`)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "Pemilik belum mengisi rekening bank.") {
		t.Errorf("payout without bank account: status = %d, body = %s", resp.StatusCode, body)
	}
	bookingAuthedRequest(t, srv, http.MethodPut, "/api/owner/bank-account", ownerToken, `{"bank_name":"Mandiri","account_number":"9876543210","account_name":"Pemilik"}`)
	_, body = bookingAuthedRequest(t, srv, http.MethodGet, "/api/admin/payouts?status=pending&limit=50", token, "")
	if !strings.Contains(string(body), fmt.Sprintf(`"id":%d`, payoutID)) || !strings.Contains(string(body), `"account_number":"9876543210"`) {
		t.Errorf("pending payouts lack the owner's current account: %s", body)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, payoutPath, token, `{"transfer_reference":"TRX-20261110-002"}`)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"Payout ditandai selesai."`) || !strings.Contains(string(body), `"bank_name":"Mandiri"`) {
		t.Fatalf("complete payout: status = %d, body = %s", resp.StatusCode, body)
	}
	if title, _ := latestNotification(t, pool, ownerID); title != "Dana sudah ditransfer" {
		t.Errorf("owner notification = %q", title)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, payoutPath, token, `{"transfer_reference":"TRX-3"}`)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "Payout sudah diproses.") {
		t.Errorf("second payout completion: status = %d, body = %s", resp.StatusCode, body)
	}
}

func TestAdminSettingsAndBroadcast(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	token := adminToken(t, srv)
	ctx := context.Background()
	var original string
	if err := pool.QueryRow(ctx, `SELECT coalesce((SELECT value::text FROM settings WHERE key = 'admin_fee'), '0')`).Scan(&original); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `INSERT INTO settings (key, value) VALUES ('admin_fee', $1::jsonb) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, original)
	})

	resp, body := bookingAuthedRequest(t, srv, http.MethodPatch, "/api/admin/settings", token, `{"admin_fee":2000000}`)
	if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(body), "The admin fee field must be between 0 and 1000000.") {
		t.Errorf("fee out of range: status = %d, body = %s", resp.StatusCode, body)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodPatch, "/api/admin/settings", token, `{"admin_fee":7000}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch settings: status = %d, body = %s", resp.StatusCode, body)
	}
	AssertJSONMatches(t, body, []byte(`{
		"status": true, "meta": {"response_code": 200, "code": 200, "severity": "OK", "message": "Pengaturan disimpan."},
		"settings": {"admin_fee": 7000, "admin_fee_label": "Rp7.000"}
	}`), nil)

	_, _, roomID, tenantToken, _, _ := newOwnerBookingFixture(t, srv, pool)
	bookingID := submitBookingFixture(t, srv, tenantToken, roomID)
	var adminFee int64
	if err := pool.QueryRow(ctx, `SELECT admin_fee FROM bookings WHERE id = $1`, bookingID).Scan(&adminFee); err != nil || adminFee != 7000 {
		t.Errorf("new booking admin_fee = %d (err %v), want 7000 right after the change", adminFee, err)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, "/api/admin/notifications/broadcast", token, `{"target":"admin","title":"","body":"x"}`)
	if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(body), "The selected target is invalid.") || !strings.Contains(string(body), "The title field is required.") {
		t.Errorf("broadcast validation: status = %d, body = %s", resp.StatusCode, body)
	}
	title := fmt.Sprintf("Promo %d", time.Now().UnixNano())
	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, "/api/admin/notifications/broadcast", token, fmt.Sprintf(`{"target":"owner","title":%q,"body":"Diskon untuk pemilik."}`, title))
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"Notifikasi dikirim."`) {
		t.Fatalf("broadcast: status = %d, body = %s", resp.StatusCode, body)
	}
	var parsed struct {
		Sent int `json:"sent"`
	}
	mustUnmarshal(t, body, &parsed)
	var owners, nonOwners int
	if err := pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE u.role = 'owner'), count(*) FILTER (WHERE u.role <> 'owner')
		FROM user_notifications n JOIN users u ON u.id = n.user_id WHERE n.title = $1 AND n.category = 'promo'`, title).Scan(&owners, &nonOwners); err != nil {
		t.Fatal(err)
	}
	if parsed.Sent == 0 || owners != parsed.Sent || nonOwners != 0 {
		t.Errorf("broadcast sent = %d, owner rows = %d, other rows = %d", parsed.Sent, owners, nonOwners)
	}
}
