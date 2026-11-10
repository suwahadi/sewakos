package contract

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// TestGetMeFixture reproduces API.md §11.3's documented shape.
func TestGetMeFixture(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv := newTestServer(t)
	phone, token := registerAndVerify(t, srv)

	resp, body := authedRequest(t, srv, http.MethodGet, "/api/me", token, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	want := fmt.Sprintf(`{
		"status": true,
		"meta": { "response_code": 200, "code": 200, "severity": "OK", "message": "OK" },
		"user": { "id": 0, "name": "OTP Test User", "role": "tenant",
		          "phone_number": "%s", "email": null, "is_verified": true }
	}`, phone)
	AssertJSONMatches(t, body, []byte(want), idPattern("$.user.id"))
}

// TestGetMeRejectsMissingToken proves RequireAuth guards this route.
func TestGetMeRejectsMissingToken(t *testing.T) {
	srv := newTestServer(t)

	resp, body := authedRequest(t, srv, http.MethodGet, "/api/me", "", "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	want := `{
		"status": false,
		"meta": { "response_code": 401, "code": 401, "severity": "WARNING", "message": "Sesi tidak valid. Silakan login kembali." }
	}`
	AssertJSONMatches(t, body, []byte(want), nil)
}

// TestPatchMeFixture reproduces API.md §11.4.
func TestPatchMeFixture(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv := newTestServer(t)
	phone, token := registerAndVerify(t, srv)

	resp, body := authedRequest(t, srv, http.MethodPatch, "/api/me", token, `{"name":"Updated Name"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	want := fmt.Sprintf(`{
		"status": true,
		"meta": { "response_code": 200, "code": 200, "severity": "OK", "message": "Profil diperbarui." },
		"user": { "id": 0, "name": "Updated Name", "role": "tenant",
		          "phone_number": "%s", "email": null, "is_verified": true }
	}`, phone)
	AssertJSONMatches(t, body, []byte(want), idPattern("$.user.id"))
}

// TestPutPasswordRejectsWrongCurrentFixture reproduces API.md §11.5's 422.
func TestPutPasswordRejectsWrongCurrentFixture(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv := newTestServer(t)
	_, token := registerAndVerify(t, srv)

	resp, body := authedRequest(t, srv, http.MethodPut, "/api/me/password", token,
		`{"current_password":"wrong-password","password":"newPassword123"}`)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	want := `{
		"status": false,
		"meta": { "response_code": 422, "code": 422, "severity": "WARNING", "message": "Validasi gagal." },
		"messages": { "current_password": ["The current password is incorrect."] }
	}`
	AssertJSONMatches(t, body, []byte(want), nil)
}

// TestPutPasswordRevokesOtherTokensOnly proves the current session survives
// a password change while every other device's token is revoked (API.md §11.5).
func TestPutPasswordRevokesOtherTokensOnly(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv := newTestServer(t)
	phone, token := registerAndVerify(t, srv)

	resp, body := postJSON(t, srv, "/api/auth/login", fmt.Sprintf(`{"phone_number":"%s","password":"Rahasia123"}`, phone))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("second-device login: status = %d, body = %s", resp.StatusCode, body)
	}
	var loginResp struct {
		Token string `json:"token"`
	}
	mustUnmarshal(t, body, &loginResp)
	otherToken := loginResp.Token

	resp, body = authedRequest(t, srv, http.MethodPut, "/api/me/password", token,
		`{"current_password":"Rahasia123","password":"newPassword123"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("change password: status = %d, body = %s", resp.StatusCode, body)
	}

	if resp, body := authedRequest(t, srv, http.MethodGet, "/api/me", token, ""); resp.StatusCode != http.StatusOK {
		t.Errorf("the session that changed the password should still work: status = %d, body = %s", resp.StatusCode, body)
	}
	if resp, body := authedRequest(t, srv, http.MethodGet, "/api/me", otherToken, ""); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("the other device's token should be revoked: status = %d, body = %s", resp.StatusCode, body)
	}
}

// TestLogoutRevokesOnlyThisToken reproduces API.md §11.1.
func TestLogoutRevokesOnlyThisToken(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv := newTestServer(t)
	phone, token := registerAndVerify(t, srv)

	resp, body := postJSON(t, srv, "/api/auth/login", fmt.Sprintf(`{"phone_number":"%s","password":"Rahasia123"}`, phone))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("second-device login: status = %d, body = %s", resp.StatusCode, body)
	}
	var loginResp struct {
		Token string `json:"token"`
	}
	mustUnmarshal(t, body, &loginResp)
	otherToken := loginResp.Token

	resp, body = authedRequest(t, srv, http.MethodPost, "/api/auth/logout", token, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("logout: status = %d, body = %s", resp.StatusCode, body)
	}
	want := `{
		"status": true,
		"meta": { "response_code": 200, "code": 200, "severity": "OK", "message": "Berhasil keluar." }
	}`
	AssertJSONMatches(t, body, []byte(want), nil)

	if resp, body := authedRequest(t, srv, http.MethodGet, "/api/me", token, ""); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("logged-out token should be revoked: status = %d, body = %s", resp.StatusCode, body)
	}
	if resp, body := authedRequest(t, srv, http.MethodGet, "/api/me", otherToken, ""); resp.StatusCode != http.StatusOK {
		t.Errorf("the other device's token must survive logout: status = %d, body = %s", resp.StatusCode, body)
	}
}

// TestPasswordResetRoundTrip covers otp/request(for=reset_password) then
// password/reset, ending with the old password rejected and every prior
// token revoked.
func TestPasswordResetRoundTrip(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv := newTestServer(t)
	// Deliberately skips tenant_verification (SPEC-GO.md §6.3: reset_password
	// doesn't require a verified phone) so this phone's otp cooldown is still
	// free for the one otp/request this test actually needs; a login works
	// for unverified users too (SPEC-GO.md §6.5), giving us a token to assert
	// gets revoked.
	phone := registerFreshUserOn(t, srv)
	resp, body := postJSON(t, srv, "/api/auth/login", fmt.Sprintf(`{"phone_number":"%s","password":"Rahasia123"}`, phone))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login before reset: status = %d, body = %s", resp.StatusCode, body)
	}
	var loginResp struct {
		Token string `json:"token"`
	}
	mustUnmarshal(t, body, &loginResp)
	token := loginResp.Token

	code := requestOTPCode(t, srv, "reset_password", phone)

	resp, body = postJSON(t, srv, "/api/auth/password/reset", fmt.Sprintf(
		`{"destination":"%s","code":"%s","password":"brandNewPassword123"}`, phone, code))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}
	want := `{
		"status": true,
		"meta": { "response_code": 200, "code": 200, "severity": "OK",
		          "message": "Password berhasil diubah. Silakan login kembali." }
	}`
	AssertJSONMatches(t, body, []byte(want), nil)

	if resp, body := authedRequest(t, srv, http.MethodGet, "/api/me", token, ""); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("every token should be revoked after a reset: status = %d, body = %s", resp.StatusCode, body)
	}
	if resp, body := postJSON(t, srv, "/api/auth/login", fmt.Sprintf(`{"phone_number":"%s","password":"Rahasia123"}`, phone)); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("the old password should no longer work: status = %d, body = %s", resp.StatusCode, body)
	}
	if resp, body := postJSON(t, srv, "/api/auth/login", fmt.Sprintf(`{"phone_number":"%s","password":"brandNewPassword123"}`, phone)); resp.StatusCode != http.StatusOK {
		t.Errorf("the new password should work: status = %d, body = %s", resp.StatusCode, body)
	}
}

// TestPhoneChangeRoundTrip covers API.md §11.6.
func TestPhoneChangeRoundTrip(t *testing.T) {
	t.Setenv("OTP_DEBUG_ECHO", "true")
	srv := newTestServer(t)
	_, token := registerAndVerify(t, srv)
	newPhone := uniquePhone(t)

	resp, body := authedRequest(t, srv, http.MethodPost, "/api/me/phone/request", token,
		fmt.Sprintf(`{"phone_number":"%s"}`, newPhone))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("phone/request: status = %d, body = %s", resp.StatusCode, body)
	}
	var parsed struct {
		DebugCode string `json:"debug_code"`
	}
	mustUnmarshal(t, body, &parsed)
	if parsed.DebugCode == "" {
		t.Fatalf("debug_code missing: %s", body)
	}

	resp, body = authedRequest(t, srv, http.MethodPost, "/api/me/phone/verify", token,
		fmt.Sprintf(`{"phone_number":"%s","code":"%s"}`, newPhone, parsed.DebugCode))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("phone/verify: status = %d, body = %s", resp.StatusCode, body)
	}
	want := fmt.Sprintf(`{
		"status": true,
		"meta": { "response_code": 200, "code": 200, "severity": "OK", "message": "Nomor handphone diperbarui." },
		"user": { "id": 0, "name": "OTP Test User", "role": "tenant",
		          "phone_number": "%s", "email": null, "is_verified": true }
	}`, newPhone)
	AssertJSONMatches(t, body, []byte(want), idPattern("$.user.id"))

	if resp, body := postJSON(t, srv, "/api/auth/login", fmt.Sprintf(`{"phone_number":"%s","password":"Rahasia123"}`, newPhone)); resp.StatusCode != http.StatusOK {
		t.Errorf("login with the new phone number should work: status = %d, body = %s", resp.StatusCode, body)
	}
}

func mustUnmarshal(t *testing.T, body []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(body, v); err != nil {
		t.Fatalf("unmarshal response: %v (body: %s)", err, body)
	}
}
