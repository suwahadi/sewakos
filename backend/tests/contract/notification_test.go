package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/sewakos/backend/internal/notification"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

// currentUserID fetches the authenticated user's id via GET /me.
func currentUserID(t *testing.T, srv *httptest.Server, token string) int64 {
	t.Helper()
	resp, body := authedRequest(t, srv, http.MethodGet, "/api/me", token, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/api/me: status = %d, body = %s", resp.StatusCode, body)
	}
	var parsed struct {
		User struct {
			ID int64 `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse /me body: %v", err)
	}
	return parsed.User.ID
}

func seedNotification(t *testing.T, pool *pgxpool.Pool, userID int64, category, title, body string) int64 {
	t.Helper()
	id, err := notification.Create(context.Background(), dbgen.New(pool), "", notification.Params{
		UserID:   userID,
		Category: category,
		Title:    title,
		Body:     body,
	})
	if err != nil {
		t.Fatalf("seed notification: %v", err)
	}
	return id
}

func TestNotificationsListAndCounterFixture(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	_, token := registerAndVerify(t, srv)
	uid := currentUserID(t, srv, token)

	bookingID := seedNotification(t, pool, uid, notification.CategoryBooking, "Booking dikonfirmasi", "Booking kamu sudah dikonfirmasi.")
	seedNotification(t, pool, uid, notification.CategoryPromo, "Promo spesial", "Dapatkan diskon 10%.")

	resp, body := authedRequest(t, srv, http.MethodGet, "/api/notifications?category=booking", token, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	want := fmt.Sprintf(`{
		"status": true,
		"meta": { "response_code": 200, "code": 200, "severity": "OK", "message": "OK" },
		"notifications": [
			{ "id": %d, "category": "booking", "title": "Booking dikonfirmasi",
			  "body": "Booking kamu sudah dikonfirmasi.", "link": null,
			  "read_at": null, "created_at": "0" }
		],
		"current_page": 1
	}`, bookingID)
	AssertJSONMatches(t, body, []byte(want), map[string]*regexp.Regexp{
		"$.notifications[0].created_at": regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$`),
	})

	resp, body = authedRequest(t, srv, http.MethodGet, "/api/notifications/counter", token, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("counter: status = %d, body = %s", resp.StatusCode, body)
	}
	wantCounter := `{
		"status": true,
		"meta": { "response_code": 200, "code": 200, "severity": "OK", "message": "OK" },
		"unread": 2
	}`
	AssertJSONMatches(t, body, []byte(wantCounter), nil)
}

func TestNotificationMarkReadFixture(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	_, token := registerAndVerify(t, srv)
	uid := currentUserID(t, srv, token)
	id := seedNotification(t, pool, uid, notification.CategoryPayment, "Pembayaran diterima", "Pembayaran kamu sudah diterima.")

	resp, body := authedRequest(t, srv, http.MethodPatch, fmt.Sprintf("/api/notifications/%d/read", id), token, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	want := `{
		"status": true,
		"meta": { "response_code": 200, "code": 200, "severity": "OK", "message": "OK" }
	}`
	AssertJSONMatches(t, body, []byte(want), nil)

	// Idempotent: reading it again still 200, read_at unchanged.
	resp, body = authedRequest(t, srv, http.MethodPatch, fmt.Sprintf("/api/notifications/%d/read", id), token, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("second read: status = %d, body = %s", resp.StatusCode, body)
	}
	AssertJSONMatches(t, body, []byte(want), nil)
}

func TestNotificationMarkReadRejectsOtherUser(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	_, ownerToken := registerAndVerify(t, srv)
	ownerID := currentUserID(t, srv, ownerToken)
	id := seedNotification(t, pool, ownerID, notification.CategoryChat, "Pesan baru", "Kamu mendapat pesan baru.")

	_, otherToken := registerAndVerify(t, srv)

	resp, body := authedRequest(t, srv, http.MethodPatch, fmt.Sprintf("/api/notifications/%d/read", id), otherToken, "")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	want := `{
		"status": false,
		"meta": { "response_code": 403, "code": 403, "severity": "WARNING", "message": "Akses ditolak." }
	}`
	AssertJSONMatches(t, body, []byte(want), nil)
}

func TestNotificationMarkReadRejectsMissing(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, _ := newTestServerAndPool(t)
	_, token := registerAndVerify(t, srv)

	resp, body := authedRequest(t, srv, http.MethodPatch, "/api/notifications/999999999/read", token, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
}
