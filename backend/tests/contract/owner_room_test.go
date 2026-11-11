package contract

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOwnerRoomCreateFixture(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	ownerID, token := newOwner(t, srv, pool)
	propertyID := newOwnerProperty(t, pool, ownerID)

	resp, body := bookingAuthedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/owner/properties/%d/rooms", propertyID), token, `{
		"name": "Kamar Tipe A", "price_monthly": 2250000, "available": 2, "capacity": 1,
		"facilities": ["WiFi", "AC", "K. Mandi Dalam", "Kasur", "Akses 24 Jam"],
		"monthly_duration_options": [12, 1, 6, 3]
	}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	want := fmt.Sprintf(`{
		"status": true,
		"meta": {"response_code": 200, "code": 200, "severity": "OK", "message": "Kamar ditambahkan."},
		"room": {
			"id": 0, "property_id": %d, "name": "Kamar Tipe A",
			"price_monthly": 2250000, "price_weekly": null, "price_daily": null,
			"available": 2, "held": 0, "occupied": 0, "capacity": 1,
			"monthly_duration_options": [1, 3, 6, 12],
			"facilities": ["WiFi", "AC", "K. Mandi Dalam", "Kasur", "Akses 24 Jam"],
			"photos": []
		}
	}`, propertyID)
	AssertJSONMatches(t, body, []byte(want), idPattern("$.room.id"))
}

func TestOwnerRoomValidation(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	ownerID, token := newOwner(t, srv, pool)
	propertyID := newOwnerProperty(t, pool, ownerID)
	path := fmt.Sprintf("/api/owner/properties/%d/rooms", propertyID)

	resp, body := bookingAuthedRequest(t, srv, http.MethodPost, path, token, `{
		"name": "Kamar", "price_monthly": 0, "price_weekly": 1.5, "available": 300, "capacity": 0,
		"facilities": ["WiFi", 7, "`+strings.Repeat("x", 51)+`"],
		"monthly_duration_options": [1, 13, 1]
	}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	AssertJSONMatches(t, body, []byte(`{
		"status": false,
		"meta": {"response_code": 422, "code": 422, "severity": "WARNING", "message": "Validasi gagal."},
		"messages": {
			"price_monthly": ["The price monthly field must be at least 1."],
			"price_weekly": ["The price weekly field must be an integer."],
			"available": ["The available field must be between 0 and 255."],
			"capacity": ["The capacity field must be between 1 and 20."],
			"facilities.1": ["The facilities.1 field must be a string.", "The facilities.1 field must not be greater than 50 characters."],
			"facilities.2": ["The facilities.2 field must not be greater than 50 characters."],
			"monthly_duration_options.1": ["The monthly duration options.1 field must be between 1 and 12."],
			"monthly_duration_options.2": ["The monthly duration options.2 field has a duplicate value."]
		}
	}`), nil)

	if _, err := pool.Exec(context.Background(), `UPDATE properties SET moderation_status = 'suspended' WHERE id = $1`, propertyID); err != nil {
		t.Fatal(err)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, path, token, `{"name":"Kamar","price_monthly":1,"available":1,"capacity":1,"facilities":[]}`)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "Properti sedang dinonaktifkan admin.") {
		t.Errorf("suspended property: status = %d, body = %s", resp.StatusCode, body)
	}
}

func TestOwnerRoomUpdateHeldAndDelete(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	ownerID, token := newOwner(t, srv, pool)
	propertyID := newOwnerProperty(t, pool, ownerID)
	ctx := context.Background()
	roomID := seedRoom(t, pool, propertyID, "Kamar Standar", 1500000, 3)
	if _, err := pool.Exec(ctx, `UPDATE rooms SET price_weekly = 400000 WHERE id = $1`, roomID); err != nil {
		t.Fatal(err)
	}
	roomPath := fmt.Sprintf("/api/owner/rooms/%d", roomID)

	resp, body := bookingAuthedRequest(t, srv, http.MethodPatch, roomPath, token, `{"price_weekly":null,"price_daily":90000,"available":5,"monthly_duration_options":[6,3]}`)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"Kamar diperbarui."`) {
		t.Fatalf("patch: status = %d, body = %s", resp.StatusCode, body)
	}
	for _, fragment := range []string{`"price_weekly":null`, `"price_daily":90000`, `"available":5`, `"monthly_duration_options":[3,6]`, `"name":"Kamar Standar"`} {
		if !strings.Contains(string(body), fragment) {
			t.Errorf("patched room lacks %s: %s", fragment, body)
		}
	}

	tenantPhone, _ := bookingRegisterAndVerify(t, srv)
	var tenantID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE phone = $1`, tenantPhone).Scan(&tenantID); err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{"paid", "approved"} {
		if _, err := pool.Exec(ctx, `INSERT INTO bookings (tenant_id, room_id, rent_type, start_date, end_date, duration, status, period_price, price, deposit, admin_fee, expires_at, stock_held)
			VALUES ($1, $2, 'monthly', current_date, current_date + 30, 1, $3, 1500000, 1500000, 1500000, 0, now() + interval '1 day', true)`, tenantID, roomID, status); err != nil {
			t.Fatalf("seed %s booking: %v", status, err)
		}
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodGet, fmt.Sprintf("/api/owner/properties/%d/rooms", propertyID), token, "")
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"held":1`) || !strings.Contains(string(body), `"occupied":1`) {
		t.Errorf("rooms list: status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodDelete, roomPath, token, "")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "Kamar sudah memiliki booking dan tidak dapat dihapus.") {
		t.Errorf("delete booked room: status = %d, body = %s", resp.StatusCode, body)
	}

	emptyRoomID := seedRoom(t, pool, propertyID, "Kamar Kosong", 1000000, 1)
	var photoKey string
	photoID := seedRoomPhoto(t, pool, propertyID, emptyRoomID)
	if err := pool.QueryRow(ctx, `SELECT object_key FROM property_photos WHERE id = $1`, photoID).Scan(&photoKey); err != nil {
		t.Fatal(err)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodDelete, fmt.Sprintf("/api/owner/rooms/%d", emptyRoomID), token, "")
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"Kamar dihapus."`) {
		t.Fatalf("delete room: status = %d, body = %s", resp.StatusCode, body)
	}
	var queued int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM storage_deletions WHERE object_key = $1`, photoKey).Scan(&queued); err != nil {
		t.Fatal(err)
	}
	if queued != 1 {
		t.Errorf("room photo queued for deletion %d times, want 1", queued)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM storage_deletions WHERE object_key LIKE $1`, fmt.Sprintf("properties/%d/%%", propertyID)); err != nil {
		t.Fatal(err)
	}

	_, otherToken := newOwner(t, srv, pool)
	if resp, _ := bookingAuthedRequest(t, srv, http.MethodPatch, roomPath, otherToken, `{"name":"X"}`); resp.StatusCode != http.StatusNotFound {
		t.Errorf("other owner PATCH: status = %d, want 404", resp.StatusCode)
	}
}

func seedRoomPhoto(t *testing.T, pool *pgxpool.Pool, propertyID, roomID int64) int64 {
	t.Helper()
	key := fmt.Sprintf("properties/%d/room-%d", propertyID, roomID)
	var id int64
	err := pool.QueryRow(context.Background(),
		`INSERT INTO property_photos (property_id, room_id, object_key, thumb_key, width, height, size_bytes)
		 VALUES ($1, $2, $3, $4, 1600, 900, 1000) RETURNING id`,
		propertyID, roomID, key+".webp", key+"_thumb.webp").Scan(&id)
	if err != nil {
		t.Fatalf("seed room photo: %v", err)
	}
	return id
}
