package contract

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

func TestOwnerPropertyCreateFixture(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	_, token := newOwner(t, srv, pool)
	areaID := seedArea(t, pool, "Kota Depok", "city", false, -6.4, 106.8)

	resp, body := bookingAuthedRequest(t, srv, http.MethodPost, "/api/owner/properties", token, fmt.Sprintf(`{
		"area_id": %d, "name": "Kost Asti Beji", "gender": "campur",
		"description": "Kost dekat stasiun UI Depok", "address": "Jl. Contoh No. 42",
		"latitude": -6.38, "longitude": 106.81
	}`, areaID))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	want := fmt.Sprintf(`{
		"status": true,
		"meta": {"response_code": 200, "code": 200, "severity": "OK", "message": "Properti dibuat dan menunggu persetujuan admin."},
		"property": {
			"id": 0, "name": "Kost Asti Beji", "slug": "",
			"gender": "campur", "description": "Kost dekat stasiun UI Depok",
			"address": "Jl. Contoh No. 42", "latitude": -6.38, "longitude": 106.81,
			"area": {"id": %d, "name": "Kota Depok"},
			"is_active": true,
			"moderation_status": "pending", "moderation_note": null,
			"is_flash_sale": false, "flash_sale_discount": 0, "flash_sale_discount_label": "Rp0",
			"views": 0, "rooms_count": 0,
			"photos": [],
			"rating": {"average": null, "count": 0},
			"created_at": ""
		}
	}`, areaID)
	AssertJSONMatches(t, body, []byte(want), map[string]*regexp.Regexp{
		"$.property.id":         regexp.MustCompile(`^\d+$`),
		"$.property.slug":       regexp.MustCompile(`^kost-asti-beji-\d+$`),
		"$.property.created_at": regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$`),
	})

	var parsed struct {
		Property struct {
			ID   int64  `json:"id"`
			Slug string `json:"slug"`
		} `json:"property"`
	}
	mustUnmarshal(t, body, &parsed)
	if want := fmt.Sprintf("kost-asti-beji-%d", parsed.Property.ID); parsed.Property.Slug != want {
		t.Errorf("slug = %q, want %q", parsed.Property.Slug, want)
	}
}

func TestOwnerPropertyValidation(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	_, token := newOwner(t, srv, pool)

	resp, body := bookingAuthedRequest(t, srv, http.MethodPost, "/api/owner/properties", token, `{}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	AssertJSONMatches(t, body, []byte(`{
		"status": false,
		"meta": {"response_code": 422, "code": 422, "severity": "WARNING", "message": "Validasi gagal."},
		"messages": {
			"area_id": ["The area id field is required."],
			"name": ["The name field is required."],
			"gender": ["The gender field is required."],
			"description": ["The description field is required."],
			"address": ["The address field is required."],
			"latitude": ["The latitude field is required."],
			"longitude": ["The longitude field is required."]
		}
	}`), nil)

	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, "/api/owner/properties", token, `{
		"area_id": 999999999, "name": "K", "gender": "laki", "description": "D", "address": "A",
		"latitude": 100, "longitude": "far"
	}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	AssertJSONMatches(t, body, []byte(`{
		"status": false,
		"meta": {"response_code": 422, "code": 422, "severity": "WARNING", "message": "Validasi gagal."},
		"messages": {
			"area_id": ["The selected area id is invalid."],
			"gender": ["The selected gender is invalid."],
			"latitude": ["The latitude field must be between -90 and 90."],
			"longitude": ["The longitude field must be a number.", "The longitude field must be between -180 and 180."]
		}
	}`), nil)
}

func TestOwnerPropertyListAndOwnership(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	ownerID, token := newOwner(t, srv, pool)
	approvedID := newOwnerProperty(t, pool, ownerID)
	pendingID := newOwnerProperty(t, pool, ownerID)
	if _, err := pool.Exec(context.Background(), `UPDATE properties SET moderation_status = 'pending' WHERE id = $1`, pendingID); err != nil {
		t.Fatal(err)
	}
	otherOwnerID, _ := newOwner(t, srv, pool)
	otherPropertyID := newOwnerProperty(t, pool, otherOwnerID)

	resp, body := bookingAuthedRequest(t, srv, http.MethodGet, "/api/owner/properties?moderation_status=pending", token, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	var list struct {
		Properties []struct {
			ID int64 `json:"id"`
		} `json:"properties"`
		CurrentPage int `json:"current_page"`
	}
	mustUnmarshal(t, body, &list)
	if len(list.Properties) != 1 || list.Properties[0].ID != pendingID || list.CurrentPage != 1 {
		t.Errorf("pending list = %s", body)
	}

	_, body = bookingAuthedRequest(t, srv, http.MethodGet, "/api/owner/properties?page=2&limit=1", token, "")
	mustUnmarshal(t, body, &list)
	if len(list.Properties) != 1 || list.Properties[0].ID != approvedID || list.CurrentPage != 2 {
		t.Errorf("page 2 = %s", body)
	}

	otherPath := fmt.Sprintf("/api/owner/properties/%d", otherPropertyID)
	for _, call := range []struct{ method, path, body string }{
		{http.MethodGet, otherPath, ""},
		{http.MethodPatch, otherPath, `{"name":"X"}`},
		{http.MethodPost, otherPath + "/submit", ""},
	} {
		if resp, _ := bookingAuthedRequest(t, srv, call.method, call.path, token, call.body); resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s %s: status = %d, want 404", call.method, call.path, resp.StatusCode)
		}
	}
}

func TestOwnerPropertyUpdateAndResubmit(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv, pool := newTestServerAndPool(t)
	ownerID, token := newOwner(t, srv, pool)
	propertyID := newOwnerProperty(t, pool, ownerID)
	ctx := context.Background()
	var slugBefore string
	if err := pool.QueryRow(ctx, `SELECT slug FROM properties WHERE id = $1`, propertyID).Scan(&slugBefore); err != nil {
		t.Fatal(err)
	}
	path := fmt.Sprintf("/api/owner/properties/%d", propertyID)

	resp, body := bookingAuthedRequest(t, srv, http.MethodPatch, path, token, `{"name":"Kost Baru","is_active":false,"is_flash_sale":true,"flash_sale_discount":100000}`)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"Properti diperbarui."`) {
		t.Fatalf("patch: status = %d, body = %s", resp.StatusCode, body)
	}
	for _, fragment := range []string{`"name":"Kost Baru"`, `"is_active":false`, `"flash_sale_discount_label":"Rp100.000"`, `"slug":"` + slugBefore + `"`} {
		if !strings.Contains(string(body), fragment) {
			t.Errorf("patched property lacks %s: %s", fragment, body)
		}
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodPatch, path, token, `{"flash_sale_discount":-1}`)
	if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(body), "The flash sale discount field must be at least 0.") {
		t.Errorf("negative discount: status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, path+"/submit", token, "")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "Properti tidak dapat diajukan ulang.") {
		t.Errorf("submit approved property: status = %d, body = %s", resp.StatusCode, body)
	}
	if _, err := pool.Exec(ctx, `UPDATE properties SET moderation_status = 'rejected', moderation_note = 'Foto kurang jelas' WHERE id = $1`, propertyID); err != nil {
		t.Fatal(err)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodPost, path+"/submit", token, "")
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"moderation_status":"pending"`) || !strings.Contains(string(body), `"moderation_note":null`) {
		t.Errorf("submit rejected property: status = %d, body = %s", resp.StatusCode, body)
	}

	if _, err := pool.Exec(ctx, `UPDATE properties SET moderation_status = 'suspended' WHERE id = $1`, propertyID); err != nil {
		t.Fatal(err)
	}
	resp, body = bookingAuthedRequest(t, srv, http.MethodPatch, path, token, `{"name":"Lagi"}`)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "Properti sedang dinonaktifkan admin.") {
		t.Errorf("patch suspended property: status = %d, body = %s", resp.StatusCode, body)
	}

	var audits int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE entity_type = 'property' AND entity_id = $1 AND action IN ('property.update', 'property.submit')`, propertyID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if audits != 2 {
		t.Errorf("audit rows = %d, want 2", audits)
	}
}
