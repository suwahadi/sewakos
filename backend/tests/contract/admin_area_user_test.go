package contract

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

func adminToken(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	resp, body := postJSON(t, srv, "/api/auth/login", `{"phone_number":"081200000000","password":"password123"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin login: status = %d, body = %s", resp.StatusCode, body)
	}
	var parsed struct {
		Token string `json:"token"`
	}
	mustUnmarshal(t, body, &parsed)
	return parsed.Token
}

func TestAdminAreaCRUD(t *testing.T) {
	srv, pool := newTestServerAndPool(t)
	token := adminToken(t, srv)
	slug := fmt.Sprintf("kota-uji-%d", time.Now().UnixNano())

	resp, body := bookingAuthedRequest(t, srv, http.MethodPost, "/api/admin/areas", token, fmt.Sprintf(`{
		"name": "Kota Uji", "slug": %q, "type": "city", "latitude": -6.4, "longitude": 106.8
	}`, slug))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create: status = %d, body = %s", resp.StatusCode, body)
	}
	AssertJSONMatches(t, body, []byte(fmt.Sprintf(`{
		"status": true, "meta": {"response_code": 200, "code": 200, "severity": "OK", "message": "Area ditambahkan."},
		"area": {"id": 0, "name": "Kota Uji", "slug": %q, "type": "city", "is_top": false, "latitude": -6.4, "longitude": 106.8, "place_id": null}
	}`, slug)), idPattern("$.area.id"))
	var created struct {
		Area struct {
			ID int64 `json:"id"`
		} `json:"area"`
	}
	mustUnmarshal(t, body, &created)
	areaPath := fmt.Sprintf("/api/admin/areas/%d", created.Area.ID)

	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, "/api/admin/areas", token, fmt.Sprintf(`{
		"name": "Kota Lain", "slug": %q, "type": "desa", "latitude": 1, "longitude": 1
	}`, slug))
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("duplicate slug: status = %d, body = %s", resp.StatusCode, body)
	}
	AssertJSONMatches(t, body, []byte(`{
		"status": false, "meta": {"response_code": 422, "code": 422, "severity": "WARNING", "message": "Validasi gagal."},
		"messages": {"slug": ["The slug has already been taken."], "type": ["The selected type is invalid."]}
	}`), nil)

	resp, body = bookingAuthedRequest(t, srv, http.MethodPatch, areaPath, token, `{"name":"Kota Uji Baru","is_top":true,"place_id":"ChIJ123"}`)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"name":"Kota Uji Baru"`) ||
		!strings.Contains(string(body), `"is_top":true`) || !strings.Contains(string(body), `"place_id":"ChIJ123"`) {
		t.Fatalf("patch: status = %d, body = %s", resp.StatusCode, body)
	}

	ownerID := seedOwner(t, pool)
	seedProperty(t, pool, propertyFixture{ownerID: ownerID, areaID: created.Area.ID, name: "Kost Area", gender: "campur", lat: -6.4, lng: 106.8})
	resp, body = bookingAuthedRequest(t, srv, http.MethodDelete, areaPath, token, "")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "Area masih dipakai properti.") {
		t.Errorf("delete used area: status = %d, body = %s", resp.StatusCode, body)
	}

	unusedID := seedArea(t, pool, "Kota Kosong", "city", false, 0, 0)
	resp, body = bookingAuthedRequest(t, srv, http.MethodDelete, fmt.Sprintf("/api/admin/areas/%d", unusedID), token, "")
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"Area dihapus."`) {
		t.Errorf("delete unused area: status = %d, body = %s", resp.StatusCode, body)
	}

	var audits int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_logs WHERE entity_type = 'area' AND entity_id IN ($1, $2)`, created.Area.ID, unusedID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if audits != 3 {
		t.Errorf("area audit rows = %d, want 3 (create, update, delete)", audits)
	}
}

func TestAdminUsersAndSuspension(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	token := adminToken(t, srv)
	phone, tenantToken := bookingRegisterAndVerify(t, srv)
	var tenantID int64
	if err := pool.QueryRow(context.Background(), `SELECT id FROM users WHERE phone = $1`, phone).Scan(&tenantID); err != nil {
		t.Fatal(err)
	}

	if resp, _ := bookingAuthedRequest(t, srv, http.MethodGet, "/api/admin/users", tenantToken, ""); resp.StatusCode != http.StatusForbidden {
		t.Errorf("tenant on admin endpoint: status = %d, want 403", resp.StatusCode)
	}

	resp, body := bookingAuthedRequest(t, srv, http.MethodGet, "/api/admin/users?q=08", token, "")
	if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(body), "The q field must be at least 3 characters.") {
		t.Errorf("short q: status = %d, body = %s", resp.StatusCode, body)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodGet, "/api/admin/users?role=tenant&q="+phone, token, "")
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), fmt.Sprintf(`"id":%d`, tenantID)) {
		t.Errorf("search by phone: status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodGet, fmt.Sprintf("/api/admin/users/%d", tenantID), token, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get user: status = %d, body = %s", resp.StatusCode, body)
	}
	AssertJSONMatches(t, body, []byte(fmt.Sprintf(`{
		"status": true, "meta": {"response_code": 200, "code": 200, "severity": "OK", "message": "OK"},
		"user": {"id": %d, "name": "", "role": "tenant", "phone_number": %q, "email": null, "is_verified": true, "suspended_at": null, "created_at": ""}
	}`, tenantID, phone)), map[string]*regexp.Regexp{
		"$.user.name":       regexp.MustCompile(`.+`),
		"$.user.created_at": regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$`),
	})

	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/admin/users/%d/suspend", tenantID), token, "")
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"Pengguna dinonaktifkan."`) {
		t.Fatalf("suspend: status = %d, body = %s", resp.StatusCode, body)
	}
	if resp, _ := bookingAuthedRequest(t, srv, http.MethodGet, "/api/me", tenantToken, ""); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("suspended token still works: status = %d, want 401", resp.StatusCode)
	}
	if resp, body := postJSON(t, srv, "/api/auth/login", fmt.Sprintf(`{"phone_number":%q,"password":"Rahasia123"}`, phone)); resp.StatusCode != http.StatusForbidden || !strings.Contains(string(body), "Akun dinonaktifkan.") {
		t.Errorf("suspended login: status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/admin/users/%d/unsuspend", tenantID), token, "")
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"Pengguna diaktifkan kembali."`) {
		t.Fatalf("unsuspend: status = %d, body = %s", resp.StatusCode, body)
	}
	if resp, body := postJSON(t, srv, "/api/auth/login", fmt.Sprintf(`{"phone_number":%q,"password":"Rahasia123"}`, phone)); resp.StatusCode != http.StatusOK {
		t.Errorf("login after unsuspend: status = %d, body = %s", resp.StatusCode, body)
	}

	var adminID int64
	if err := pool.QueryRow(context.Background(), `SELECT id FROM users WHERE phone = '081200000000'`).Scan(&adminID); err != nil {
		t.Fatal(err)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, fmt.Sprintf("/api/admin/users/%d/suspend", adminID), token, "")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "Admin tidak dapat dinonaktifkan.") {
		t.Errorf("suspend admin: status = %d, body = %s", resp.StatusCode, body)
	}
}
