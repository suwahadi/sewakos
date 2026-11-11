package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/suwahadi/sewakos/backend/internal/auth"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
}

func TestRequestIDSetsHeaderAndContext(t *testing.T) {
	var sawID string
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawID = RequestIDFromContext(r.Context())
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	header := w.Header().Get("X-Request-Id")
	if header == "" || header != sawID {
		t.Errorf("header=%q context=%q, want equal and non-empty", header, sawID)
	}
}

func TestSecurityHeaders(t *testing.T) {
	for _, hsts := range []bool{false, true} {
		w := httptest.NewRecorder()
		SecurityHeaders(hsts)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		if w.Header().Get("X-Content-Type-Options") != "nosniff" || w.Header().Get("X-Frame-Options") != "DENY" ||
			w.Header().Get("Content-Security-Policy") == "" || w.Header().Get("Referrer-Policy") != "no-referrer" {
			t.Errorf("hsts=%v: headers = %v", hsts, w.Header())
		}
		if got := w.Header().Get("Strict-Transport-Security") != ""; got != hsts {
			t.Errorf("hsts=%v: Strict-Transport-Security present = %v", hsts, got)
		}
	}
}

func TestForwardedClientIP(t *testing.T) {
	trusted := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}
	cases := []struct {
		name, remote, forwarded, want string
	}{
		{"untrusted peer ignores header", "203.0.113.9:4000", "198.51.100.1", "203.0.113.9"},
		{"trusted peer without header keeps peer", "10.0.0.2:4000", "", "10.0.0.2"},
		{"trusted peer uses right-most untrusted hop", "10.0.0.2:4000", "198.51.100.66, 198.51.100.7, 10.0.0.3", "198.51.100.7"},
		{"malformed hop falls back to peer", "10.0.0.2:4000", "198.51.100.7, bogus", "10.0.0.2"},
		{"ipv6 client", "10.0.0.2:4000", "2001:db8::1", "2001:db8::1"},
	}
	for _, tc := range cases {
		var got string
		h := ForwardedClientIP(trusted)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { got = clientIP(r) }))
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = tc.remote
		if tc.forwarded != "" {
			r.Header.Set("X-Forwarded-For", tc.forwarded)
		}
		h.ServeHTTP(httptest.NewRecorder(), r)
		if got != tc.want {
			t.Errorf("%s: client IP = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestRecovererTurnsPanicInto500Envelope(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	h := Recoverer(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", w.Code)
	}
	if got := w.Body.String(); !strings.Contains(got, "Terjadi kesalahan pada server.") {
		t.Errorf("body = %s", got)
	}
}

func TestCORSNoOriginsConfiguredMeansNoHeaders(t *testing.T) {
	h := CORS(nil)(okHandler())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Origin", "https://sewakos.web.id")
	h.ServeHTTP(w, r)

	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("no CORS headers should be set when CORS_ALLOWED_ORIGINS is empty")
	}
}

func TestCORSAllowedOriginGetsHeadersAndPreflightNoContent(t *testing.T) {
	h := CORS([]string{"https://sewakos.web.id"})(okHandler())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Origin", "https://sewakos.web.id")
	h.ServeHTTP(w, r)
	if w.Header().Get("Access-Control-Allow-Origin") != "https://sewakos.web.id" {
		t.Errorf("Access-Control-Allow-Origin = %q", w.Header().Get("Access-Control-Allow-Origin"))
	}

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodOptions, "/", nil)
	r2.Header.Set("Origin", "https://sewakos.web.id")
	h.ServeHTTP(w2, r2)
	if w2.Code != http.StatusNoContent {
		t.Errorf("preflight status = %d, want 204", w2.Code)
	}
}

func TestCORSDisallowedOriginGetsNoHeaders(t *testing.T) {
	h := CORS([]string{"https://sewakos.web.id"})(okHandler())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Origin", "https://evil.example")
	h.ServeHTTP(w, r)

	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("a disallowed origin must not get CORS headers")
	}
}

func TestClientCheck(t *testing.T) {
	h := ClientCheck("KOSAPI WEB:WEB")(okHandler())

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("x-api-client", "KOSAPI WEB:WEB")
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("correct client header: status = %d", w.Code)
	}

	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/", nil))
	if w2.Code != http.StatusUnauthorized {
		t.Errorf("missing client header: status = %d, want 401", w2.Code)
	}
}

func newTestRedisClient(t *testing.T) *redis.Client {
	t.Helper()
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379/0"
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		t.Fatalf("parse redis url: %v", err)
	}
	client := redis.NewClient(opts)
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Skipf("redis not reachable, skipping: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestRateLimitAPIEnforcesLimitAndHeaders(t *testing.T) {
	rdb := newTestRedisClient(t)
	h := RateLimitAPI(rdb)(okHandler())

	// A fresh fake IP per run, so repeated test runs within the same
	// one-minute window never share (and so never collide on) a rate-limit
	// counter left over from an earlier run.
	fakeIP := fmt.Sprintf("203.0.113.%d", time.Now().UnixNano()%250+1)
	req := func() *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = fakeIP + ":12345"
		return r
	}

	var last *httptest.ResponseRecorder
	for i := 0; i < 61; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req())
		last = w
	}
	if last.Code != http.StatusTooManyRequests {
		t.Fatalf("61st request status = %d, want 429", last.Code)
	}
	if last.Header().Get("Retry-After") == "" {
		t.Error("429 response must set Retry-After")
	}
}

type fakeVerifier struct {
	user auth.VerifiedUser
	err  error
}

func (f fakeVerifier) Verify(_ context.Context, _ string) (auth.VerifiedUser, error) {
	return f.user, f.err
}

func TestRequireAuthRejectsMissingOrInvalidToken(t *testing.T) {
	h := RequireAuth(fakeVerifier{err: auth.ErrInvalidToken})(okHandler())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestRequireAuthAttachesUserOnSuccess(t *testing.T) {
	h := RequireAuth(fakeVerifier{user: auth.VerifiedUser{ID: 2, Role: "tenant"}})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, ok := UserFromContext(r.Context())
			if !ok || u.ID != 2 || u.Role != "tenant" {
				t.Errorf("user in context = %+v, ok=%v", u, ok)
			}
		}),
	)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer 2|whatever")
	h.ServeHTTP(w, r)
}

func TestOptionalAuthNeverRejects(t *testing.T) {
	h := OptionalAuth(fakeVerifier{err: errors.New("no token")})(okHandler())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 even without a valid token", w.Code)
	}
}

func TestRequireRole(t *testing.T) {
	h := RequireRole("owner", "admin")(okHandler())

	ctx := context.WithValue(context.Background(), ctxKeyUser, AuthUser{ID: 1, Role: "tenant"})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx))
	if w.Code != http.StatusForbidden {
		t.Errorf("tenant hitting an owner/admin route: status = %d, want 403", w.Code)
	}

	ctx2 := context.WithValue(context.Background(), ctxKeyUser, AuthUser{ID: 1, Role: "owner"})
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx2))
	if w2.Code != http.StatusOK {
		t.Errorf("owner hitting an owner/admin route: status = %d, want 200", w2.Code)
	}
}
