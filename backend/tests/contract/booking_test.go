package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// bookingAuthedRequest is authedRequest plus a per-test X-Forwarded-For (see
// discoveryTestIP): without it, every contract test file shares the loopback
// IP's "api" 60/min bucket in the one Redis instance the whole suite runs
// against, and this file's extra register/OTP/submit calls starve unrelated
// OTP/register tests elsewhere in the package.
func bookingAuthedRequest(t *testing.T, srv *httptest.Server, method, path, token, body string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, srv.URL+path, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("x-api-client", "KOSAPI WEB:WEB")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", discoveryTestIP(t))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
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

func bookingRegisterAndVerify(t *testing.T, srv *httptest.Server) (phone, token string) {
	t.Helper()
	phone = fmt.Sprintf("081%09d", time.Now().UnixNano()%1_000_000_000)
	resp, body := bookingAuthedRequest(t, srv, http.MethodPost, "/api/auth/register", "", fmt.Sprintf(`{
		"name": "Booking Test User",
		"phone_number": "%s",
		"password": "Rahasia123",
		"is_agree_privacy_policy": true
	}`, phone))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register: status = %d, body = %s", resp.StatusCode, body)
	}

	token = bookingVerifyPhone(t, srv, phone)
	return phone, token
}

func bookingVerifyPhone(t *testing.T, srv *httptest.Server, phone string) string {
	t.Helper()
	resp, body := bookingAuthedRequest(t, srv, http.MethodPost, "/api/auth/otp/request", "",
		fmt.Sprintf(`{"for":"tenant_verification","via":"0","destination":"%s"}`, phone))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("otp/request: status = %d, body = %s", resp.StatusCode, body)
	}
	var requested struct {
		DebugCode string `json:"debug_code"`
	}
	if err := json.Unmarshal(body, &requested); err != nil {
		t.Fatalf("parse otp/request body: %v", err)
	}
	if requested.DebugCode == "" {
		t.Fatalf("debug_code missing from response (OTP_DEBUG_ECHO must be true for this test): %s", body)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, "/api/auth/otp/check", "", fmt.Sprintf(
		`{"for":"tenant_verification","destination":"%s","code":"%s"}`, phone, requested.DebugCode))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("otp/check: status = %d, body = %s", resp.StatusCode, body)
	}
	var parsed struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse otp/check body: %v", err)
	}
	return parsed.Token
}

func seedBookingFixture(t *testing.T, pool *pgxpool.Pool, priceMonthly int64, available int, flashSale bool) (roomID int64) {
	t.Helper()
	ownerID := seedOwner(t, pool)
	areaID := seedArea(t, pool, fmt.Sprintf("Area %d", time.Now().UnixNano()), "city", false, -6.2, 106.8)
	propertyID := seedProperty(t, pool, propertyFixture{
		ownerID: ownerID, areaID: areaID, name: "Kost Booking Test", gender: "campur",
		lat: -6.2, lng: 106.8, flashSale: flashSale,
	})
	return seedRoom(t, pool, propertyID, "Kamar Tes", priceMonthly, available)
}

func submitBooking(t *testing.T, srv *httptest.Server, token string, body string) (*http.Response, []byte) {
	t.Helper()
	return bookingAuthedRequest(t, srv, http.MethodPost, "/api/booking/submit", token, body)
}

func TestBookingSubmitListAndDetailFixture(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 2, true)
	_, token := bookingRegisterAndVerify(t, srv)

	startDate := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	resp, body := submitBooking(t, srv, token, fmt.Sprintf(
		`{"room_id":%d,"rent_type":"monthly","start_date":%q}`, roomID, startDate))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("submit: status = %d, body = %s", resp.StatusCode, body)
	}

	var submitted struct {
		Booking struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
		} `json:"booking"`
	}
	if err := json.Unmarshal(body, &submitted); err != nil {
		t.Fatalf("parse submit body: %v", err)
	}
	if submitted.Booking.Status != "pending" {
		t.Errorf("booking.status = %q, want pending", submitted.Booking.Status)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodGet, "/api/booking/my", token, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: status = %d, body = %s", resp.StatusCode, body)
	}
	want := fmt.Sprintf(`{
		"status": true, "meta": { "response_code": 200, "code": 200, "severity": "OK", "message": "OK" },
		"bookings": [
			{ "id": %d, "status": "pending", "rent_type": "monthly",
			  "room": { "id": %d, "name": "Kamar Tes" },
			  "start_date": %q, "duration": 1, "end_date": "0",
			  "price": 2000000, "price_label": "Rp2.000.000",
			  "expires_at": "0", "created_at": "0" }
		],
		"current_page": 1
	}`, submitted.Booking.ID, roomID, startDate)
	AssertJSONMatches(t, body, []byte(want), map[string]*regexp.Regexp{
		"$.bookings[0].end_date":   regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`),
		"$.bookings[0].expires_at": regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\+00:00$`),
		"$.bookings[0].created_at": regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$`),
	})

	resp, body = bookingAuthedRequest(t, srv, http.MethodGet, fmt.Sprintf("/api/booking/%d", submitted.Booking.ID), token, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("detail: status = %d, body = %s", resp.StatusCode, body)
	}
	wantDetail := fmt.Sprintf(`{
		"status": true, "meta": { "response_code": 200, "code": 200, "severity": "OK", "message": "OK" },
		"booking": {
			"id": %d, "status": "pending", "rent_type": "monthly",
			"room": { "id": %d, "name": "Kamar Tes" },
			"property": { "id": 0, "name": "Kost Booking Test" },
			"start_date": %q, "duration": 1, "end_date": "0",
			"price": 2000000, "price_label": "Rp2.000.000",
			"deposit": 2250000, "deposit_label": "Rp2.250.000",
			"admin_fee": 0, "admin_fee_label": "Rp0",
			"total_payment": 4250000, "total_payment_label": "Rp4.250.000",
			"expires_at": "0", "paid_at": null, "owner_deadline_at": null,
			"reject_reason": null, "deposit_returned_at": null,
			"can_extend": false, "can_review": false, "created_at": "0"
		},
		"payment": null, "refund": null
	}`, submitted.Booking.ID, roomID, startDate)
	AssertJSONMatches(t, body, []byte(wantDetail), map[string]*regexp.Regexp{
		"$.booking.property.id": regexp.MustCompile(`^\d+$`),
		"$.booking.end_date":    regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`),
		"$.booking.expires_at":  regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\+00:00$`),
		"$.booking.created_at":  regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$`),
	})
}

func TestBookingSubmitDurationOutOfRangeIs422(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, token := bookingRegisterAndVerify(t, srv)

	startDate := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	resp, body := submitBooking(t, srv, token, fmt.Sprintf(
		`{"room_id":%d,"rent_type":"monthly","start_date":%q,"duration":2}`, roomID, startDate))
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s, want 422 (monthly_duration_options defaults to [1])", resp.StatusCode, body)
	}
	var parsed struct {
		Messages map[string][]string `json:"messages"`
	}
	_ = json.Unmarshal(body, &parsed)
	if len(parsed.Messages["duration"]) == 0 || parsed.Messages["duration"][0] != "The selected duration is invalid." {
		t.Errorf("messages.duration = %v, want \"The selected duration is invalid.\"", parsed.Messages["duration"])
	}
}

func TestBookingSubmitUnavailableRoomIs400(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 0, false)
	_, token := bookingRegisterAndVerify(t, srv)

	startDate := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	resp, body := submitBooking(t, srv, token, fmt.Sprintf(
		`{"room_id":%d,"rent_type":"monthly","start_date":%q}`, roomID, startDate))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s, want 400", resp.StatusCode, body)
	}
	var parsed struct {
		Meta struct {
			Message string `json:"message"`
		} `json:"meta"`
	}
	_ = json.Unmarshal(body, &parsed)
	if parsed.Meta.Message != "Kamar tidak tersedia." {
		t.Errorf("meta.message = %q, want %q", parsed.Meta.Message, "Kamar tidak tersedia.")
	}
}

func TestBookingSubmitUnpricedRentTypeIs400(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, token := bookingRegisterAndVerify(t, srv)

	startDate := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	resp, body := submitBooking(t, srv, token, fmt.Sprintf(
		`{"room_id":%d,"rent_type":"daily","start_date":%q}`, roomID, startDate))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s, want 400 (room has no price_daily)", resp.StatusCode, body)
	}
}

func TestBookingSubmitRequiresTenantRole(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)

	phone := fmt.Sprintf("081%09d", time.Now().UnixNano()%1_000_000_000)
	resp, body := bookingAuthedRequest(t, srv, http.MethodPost, "/api/auth/register", "", fmt.Sprintf(`{
		"name": "Owner Role Test User",
		"phone_number": "%s",
		"password": "Rahasia123",
		"is_agree_privacy_policy": true
	}`, phone))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register: status = %d, body = %s", resp.StatusCode, body)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE users SET role = 'owner' WHERE phone = $1`, phone); err != nil {
		t.Fatalf("promote to owner: %v", err)
	}
	token := bookingVerifyPhone(t, srv, phone)

	startDate := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	resp, body = submitBooking(t, srv, token, fmt.Sprintf(
		`{"room_id":%d,"rent_type":"monthly","start_date":%q}`, roomID, startDate))
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s, want 403 for owner role", resp.StatusCode, body)
	}
}

func TestBookingDetailOwnershipIsEnforced(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	roomID := seedBookingFixture(t, pool, 2250000, 2, false)
	_, tokenA := bookingRegisterAndVerify(t, srv)
	_, tokenB := bookingRegisterAndVerify(t, srv)

	startDate := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	resp, body := submitBooking(t, srv, tokenA, fmt.Sprintf(
		`{"room_id":%d,"rent_type":"monthly","start_date":%q}`, roomID, startDate))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("submit: status = %d, body = %s", resp.StatusCode, body)
	}
	var submitted struct {
		Booking struct {
			ID int64 `json:"id"`
		} `json:"booking"`
	}
	_ = json.Unmarshal(body, &submitted)

	resp, body = bookingAuthedRequest(t, srv, http.MethodGet, fmt.Sprintf("/api/booking/%d", submitted.Booking.ID), tokenB, "")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s, want 403 for another tenant's booking", resp.StatusCode, body)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodGet, "/api/booking/999999999", tokenA, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s, want 404 for unknown booking", resp.StatusCode, body)
	}
}
