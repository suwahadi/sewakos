package contract

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"
	"time"
)

func uniquePhone(t *testing.T) string {
	t.Helper()
	// 081 + 9 digits keeps the total length inside API.md's ^08\d{7,12}$ regex.
	return fmt.Sprintf("081%09d", time.Now().UnixNano()%1_000_000_000)
}

// TestRegisterFixture reproduces API.md §2.1's documented success example.
func TestRegisterFixture(t *testing.T) {
	srv := newTestServer(t)
	phone := uniquePhone(t)

	resp, body := postJSON(t, srv, "/api/auth/register", fmt.Sprintf(`{
		"name": "Test User",
		"phone_number": "%s",
		"password": "Rahasia123",
		"is_agree_privacy_policy": true
	}`, phone))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}

	want := fmt.Sprintf(`{
		"status": true,
		"meta": { "response_code": 200, "code": 200, "severity": "OK",
		          "message": "Registrasi berhasil. Silakan verifikasi nomor handphone." },
		"user": { "id": 0, "name": "Test User", "phone_number": "%s" },
		"next_step": "otp_verification"
	}`, phone)
	AssertJSONMatches(t, body, []byte(want), map[string]*regexp.Regexp{
		"$.user.id": regexp.MustCompile(`^\d+$`),
	})
}

// TestRegisterValidationFixture reproduces API.md §2.1's documented 422 example.
func TestRegisterValidationFixture(t *testing.T) {
	srv := newTestServer(t)

	resp, body := postJSON(t, srv, "/api/auth/register", `{}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}

	want := `{
		"status": false,
		"meta": { "response_code": 422, "code": 422, "severity": "WARNING", "message": "Validasi gagal." },
		"messages": {
			"name": ["The name field is required."],
			"phone_number": ["The phone number field is required."],
			"password": ["The password field is required."],
			"is_agree_privacy_policy": ["The is agree privacy policy field must be accepted."]
		}
	}`
	AssertJSONMatches(t, body, []byte(want), nil)
}

// TestRegisterRejectsDuplicatePhone proves the unique(users.phone) rule fires
// without caring about the DB's own seeded fixtures: it registers a user and
// immediately tries to register the exact same phone number again.
func TestRegisterRejectsDuplicatePhone(t *testing.T) {
	srv := newTestServer(t)
	phone := uniquePhone(t)
	body := fmt.Sprintf(`{"name":"Dup","phone_number":"%s","password":"Rahasia123","is_agree_privacy_policy":true}`, phone)

	if resp, b := postJSON(t, srv, "/api/auth/register", body); resp.StatusCode != http.StatusOK {
		t.Fatalf("first register: status = %d, body = %s", resp.StatusCode, b)
	}

	resp, b := postJSON(t, srv, "/api/auth/register", body)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, b)
	}
	want := `{
		"status": false,
		"meta": { "response_code": 422, "code": 422, "severity": "WARNING", "message": "Validasi gagal." },
		"messages": { "phone_number": ["The phone number has already been taken."] }
	}`
	AssertJSONMatches(t, b, []byte(want), nil)
}
