package contract

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

// registerFreshUserOn registers a brand-new, unverified user on srv and
// returns its phone number.
func registerFreshUserOn(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	phone := uniquePhone(t)
	resp, body := postJSON(t, srv, "/api/auth/register", fmt.Sprintf(`{
		"name": "OTP Test User",
		"phone_number": "%s",
		"password": "Rahasia123",
		"is_agree_privacy_policy": true
	}`, phone))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register: status = %d, body = %s", resp.StatusCode, body)
	}
	return phone
}

func requestOTPCode(t *testing.T, srv *httptest.Server, forType, phone string) string {
	t.Helper()
	resp, body := postJSON(t, srv, "/api/auth/otp/request", fmt.Sprintf(
		`{"for":%q,"via":"0","destination":"%s"}`, forType, phone))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("otp/request: status = %d, body = %s", resp.StatusCode, body)
	}
	var parsed struct {
		DebugCode string `json:"debug_code"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse otp/request body: %v", err)
	}
	if parsed.DebugCode == "" {
		t.Fatalf("debug_code missing from response (OTP_DEBUG_ECHO must be true for this test): %s", body)
	}
	return parsed.DebugCode
}

// TestOTPTenantVerificationRoundTrip reproduces the register -> otp/request ->
// otp/check happy path (API.md §2.1-2.3), ending in a token exactly like
// otp/check's documented fixture.
func TestOTPTenantVerificationRoundTrip(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv := newTestServer(t)
	phone := registerFreshUserOn(t, srv)

	code := requestOTPCode(t, srv, "tenant_verification", phone)

	resp, body := postJSON(t, srv, "/api/auth/otp/check", fmt.Sprintf(
		`{"for":"tenant_verification","destination":"%s","code":"%s"}`, phone, code))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("otp/check: status = %d, body = %s", resp.StatusCode, body)
	}

	want := `{
		"status": true,
		"meta": { "response_code": 200, "code": 200, "severity": "OK", "message": "Nomor handphone terverifikasi." },
		"verified": true,
		"token": "1|abc",
		"user": { "id": 1, "name": "OTP Test User", "role": "tenant" }
	}`
	AssertJSONMatches(t, body, []byte(want), map[string]*regexp.Regexp{
		"$.token":   regexp.MustCompile(`^\d+\|[A-Za-z0-9]{48}$`),
		"$.user.id": regexp.MustCompile(`^\d+$`),
	})
}

// TestOTPCheckRejectsWrongCode proves the exact 400 shape (API.md §2.3) and
// that a wrong guess doesn't consume the real code. This makes 3 otp
// request/check calls for one destination, so it raises OTP_MAX_PER_MINUTE
// to isolate "wrong code behavior" from the separately-tested 2/minute shared
// quota (TestOTPRequestEnforcesCooldownFixture already covers the cooldown
// that quota shares a mechanism with).
func TestOTPCheckRejectsWrongCode(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	t.Setenv("OTP_MAX_PER_MINUTE", "10")
	srv := newTestServer(t)
	phone := registerFreshUserOn(t, srv)
	code := requestOTPCode(t, srv, "tenant_verification", phone)
	wrong := "0000"
	if wrong == code {
		wrong = "1111"
	}

	resp, body := postJSON(t, srv, "/api/auth/otp/check", fmt.Sprintf(
		`{"for":"tenant_verification","destination":"%s","code":"%s"}`, phone, wrong))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	want := `{
		"status": false,
		"meta": { "response_code": 400, "code": 400, "severity": "WARNING", "message": "Kode verifikasi salah atau kedaluwarsa." }
	}`
	AssertJSONMatches(t, body, []byte(want), nil)

	// The real code must still work afterwards.
	resp, body = postJSON(t, srv, "/api/auth/otp/check", fmt.Sprintf(
		`{"for":"tenant_verification","destination":"%s","code":"%s"}`, phone, code))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("real code after a wrong guess: status = %d, body = %s", resp.StatusCode, body)
	}
}

// TestOTPRequestEnforcesCooldownFixture reproduces API.md §2.2's documented
// cooldown 422 shape.
func TestOTPRequestEnforcesCooldownFixture(t *testing.T) {
	srv := newTestServer(t)
	phone := registerFreshUserOn(t, srv)

	if resp, body := postJSON(t, srv, "/api/auth/otp/request", fmt.Sprintf(
		`{"for":"tenant_verification","via":"0","destination":"%s"}`, phone)); resp.StatusCode != http.StatusOK {
		t.Fatalf("first request: status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body := postJSON(t, srv, "/api/auth/otp/request", fmt.Sprintf(
		`{"for":"tenant_verification","via":"0","destination":"%s"}`, phone))
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	want := `{
		"status": false,
		"meta": { "response_code": 422, "code": 422, "severity": "WARNING",
		          "message": "Silakan tunggu sebelum meminta kode baru." },
		"messages": { "destination": ["Silakan tunggu sebelum meminta kode baru."] }
	}`
	AssertJSONMatches(t, body, []byte(want), nil)
}

// TestOTPRequestLoginRejectsUnverifiedNumber reproduces SPEC-GO.md §6.3 step 4.
func TestOTPRequestLoginRejectsUnverifiedNumber(t *testing.T) {
	srv := newTestServer(t)
	phone := registerFreshUserOn(t, srv)

	resp, body := postJSON(t, srv, "/api/auth/otp/request", fmt.Sprintf(
		`{"for":"login","via":"0","destination":"%s"}`, phone))
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	want := `{
		"status": false,
		"meta": { "response_code": 422, "code": 422, "severity": "WARNING",
		          "message": "Nomor handphone belum terverifikasi." },
		"messages": { "destination": ["Nomor handphone belum terverifikasi."] }
	}`
	AssertJSONMatches(t, body, []byte(want), nil)
}

// TestOTPCheckRejectsResetPasswordFor proves reset_password codes can't be
// consumed through otp/check (SPEC-GO.md §6.4), even with a destination that
// has never requested a code at all.
func TestOTPCheckRejectsResetPasswordFor(t *testing.T) {
	srv := newTestServer(t)
	phone := uniquePhone(t)

	resp, body := postJSON(t, srv, "/api/auth/otp/check", fmt.Sprintf(
		`{"for":"reset_password","destination":"%s","code":"0000"}`, phone))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	want := `{
		"status": false,
		"meta": { "response_code": 400, "code": 400, "severity": "WARNING", "message": "Kode verifikasi salah atau kedaluwarsa." }
	}`
	AssertJSONMatches(t, body, []byte(want), nil)
}

// TestOTPSharedRateLimitFixture proves the "otp" limiter (2/minute per
// destination, API.md §1.5) is shared between otp/request and otp/check: the
// 3rd call for one destination within a minute is rejected with 429 and the
// documented rate-limit headers, regardless of which endpoint it hits.
func TestOTPSharedRateLimitFixture(t *testing.T) {
	srv := newTestServer(t)
	phone := registerFreshUserOn(t, srv)

	if resp, body := postJSON(t, srv, "/api/auth/otp/request", fmt.Sprintf(
		`{"for":"tenant_verification","via":"0","destination":"%s"}`, phone)); resp.StatusCode != http.StatusOK {
		t.Fatalf("1st call: status = %d, body = %s", resp.StatusCode, body)
	}
	if resp, body := postJSON(t, srv, "/api/auth/otp/check", fmt.Sprintf(
		`{"for":"tenant_verification","destination":"%s","code":"0000"}`, phone)); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("2nd call: status = %d, body = %s", resp.StatusCode, body)
	}

	resp, body := postJSON(t, srv, "/api/auth/otp/check", fmt.Sprintf(
		`{"for":"tenant_verification","destination":"%s","code":"0000"}`, phone))
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("3rd call: status = %d, body = %s", resp.StatusCode, body)
	}
	for _, h := range []string{"Retry-After", "X-RateLimit-Limit", "X-RateLimit-Remaining"} {
		if resp.Header.Get(h) == "" {
			t.Errorf("missing %s header on 429 response", h)
		}
	}
	want := `{
		"status": false,
		"meta": { "response_code": 429, "code": 429, "severity": "WARNING",
		          "message": "Terlalu banyak permintaan. Silakan coba lagi nanti." }
	}`
	AssertJSONMatches(t, body, []byte(want), nil)
}
