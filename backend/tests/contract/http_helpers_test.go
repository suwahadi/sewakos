package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

// idPattern is the dynamic-field map for a fixture whose only unpredictable
// value is a numeric database id at path.
func idPattern(path string) map[string]*regexp.Regexp {
	return map[string]*regexp.Regexp{path: regexp.MustCompile(`^\d+$`)}
}

// authedRequest issues method/path with the given body (ignored for GET) and
// an Authorization: Bearer token header, alongside the usual x-api-client one.
func authedRequest(t *testing.T, srv *httptest.Server, method, path, token, body string) (*http.Response, []byte) {
	t.Helper()
	var bodyReader *bytes.Buffer
	if body == "" {
		bodyReader = bytes.NewBuffer(nil)
	} else {
		bodyReader = bytes.NewBufferString(body)
	}
	req, err := http.NewRequest(method, srv.URL+path, bodyReader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("x-api-client", "KOSAPI WEB:WEB")
	req.Header.Set("Content-Type", "application/json")
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

// registerAndVerify registers a fresh user on srv and verifies its phone via
// the debug_code OTP flow, returning its phone number and access token. It
// requires OTP_DEBUG_ECHO=true (set it with t.Setenv before calling newTestServer).
func registerAndVerify(t *testing.T, srv *httptest.Server) (phone, token string) {
	t.Helper()
	phone = registerFreshUserOn(t, srv)
	code := requestOTPCode(t, srv, "tenant_verification", phone)

	resp, body := postJSON(t, srv, "/api/auth/otp/check", `{"for":"tenant_verification","destination":"`+phone+`","code":"`+code+`"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("otp/check: status = %d, body = %s", resp.StatusCode, body)
	}
	var parsed struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("parse otp/check body: %v", err)
	}
	return phone, parsed.Token
}
