// This file is the "one green fixture" B1.10 asks for: a real HTTP call
// against the real router, a real Postgres and Redis, compared to the exact
// JSON API.md §2.4 documents. Later tasks add one file like this per endpoint
// group as that group is implemented.
package contract

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"regexp"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/suwahadi/sewakos/backend/internal/config"
	"github.com/suwahadi/sewakos/backend/internal/media"
	"github.com/suwahadi/sewakos/backend/internal/midtrans"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres"
	"github.com/suwahadi/sewakos/backend/internal/platform/storage"
	"github.com/suwahadi/sewakos/backend/internal/server"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv, _ := newTestServerAndPool(t)
	return srv
}

// newTestServerAndPool is newTestServer plus the pool, for tests that need to
// seed data directly (e.g. notifications, which no endpoint creates yet).
func newTestServerAndPool(t *testing.T) (*httptest.Server, *pgxpool.Pool) {
	t.Helper()
	return newTestServerAndPoolWithMidtrans(t, defaultMockMidtransClient())
}

// newTestServerAndPoolWithMidtrans is newTestServerAndPool with an explicit
// Midtrans double, for tests that assert on checkout's call to it.
func newTestServerAndPoolWithMidtrans(t *testing.T, midtransClient midtrans.Client) (*httptest.Server, *pgxpool.Pool) {
	t.Helper()

	databaseURL := envOr("TEST_DATABASE_URL", "postgres://kos:kos@localhost:5432/kos?sslmode=disable")
	redisURL := envOr("TEST_REDIS_URL", "redis://localhost:6379/0")

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, databaseURL)
	if err != nil {
		t.Skipf("postgres not reachable, skipping contract test: %v", err)
	}
	t.Cleanup(pool.Close)

	redisOptions, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatalf("parse redis url: %v", err)
	}
	redisClient := redis.NewClient(redisOptions)
	t.Cleanup(func() { _ = redisClient.Close() })
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not reachable, skipping contract test: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.TrustedProxies = []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8"), netip.MustParsePrefix("::1/128")}

	deps := server.Deps{
		Config:   cfg,
		DB:       pool,
		Redis:    redisClient,
		Logger:   slog.New(slog.NewJSONHandler(os.Stderr, nil)),
		Midtrans: midtransClient,
		Images: media.NewProcessor(media.Config{
			MaxPixels: cfg.ImageMaxPixels, LargePx: cfg.ImageLargePx, ThumbPx: cfg.ImageThumbPx,
			LargeQuality: cfg.ImageLargeQuality, ThumbQuality: cfg.ImageThumbQuality, Concurrency: cfg.ImageMaxConcurrency,
		}),
	}
	if cfg.StorageEndpoint != "" {
		objectStore, err := storage.New(testStorageConfig(cfg))
		if err != nil {
			t.Fatalf("create storage client: %v", err)
		}
		deps.Storage = objectStore
	}
	router := server.NewRouter(deps)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv, pool
}

func defaultMockMidtransClient() *midtrans.MockClient {
	return &midtrans.MockClient{
		CreateTransactionFunc: func(_ context.Context, req midtrans.TransactionRequest) (midtrans.TransactionResponse, error) {
			return midtrans.TransactionResponse{
				Token:       "mock-snap-token",
				RedirectURL: "https://app.sandbox.midtrans.com/snap/v4/redirection/mock-" + req.TransactionDetails.OrderID,
			}, nil
		},
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func postJSON(t *testing.T, srv *httptest.Server, path, body string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, srv.URL+path, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("x-api-client", "KOSAPI WEB:WEB")
	req.Header.Set("Content-Type", "application/json")

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

// TestLoginFixture reproduces API.md §2.4's documented example exactly, using
// the seeded tenant (SPEC-GO.md §10: "Anak Kos Contoh", 081200000002,
// password123). If that user isn't present, the database hasn't been seeded
// (cmd/seed) and the test is skipped rather than failed, since this harness
// is documented to run "terhadap DB hasil seed", not to seed it itself.
func TestLoginFixture(t *testing.T) {
	srv := newTestServer(t)

	resp, body := postJSON(t, srv, "/api/auth/login", `{"phone_number":"081200000002","password":"password123"}`)
	if resp.StatusCode == http.StatusUnauthorized {
		t.Skip("seeded tenant not found; run cmd/seed before the contract suite")
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}

	want := `{
		"status": true,
		"meta": { "response_code": 200, "code": 200, "severity": "OK", "message": "Login berhasil." },
		"token": "2|VjGS6DZX8K096pOaQG3XKq7BkVLtZV8eqj1pNXpn5c3c9053",
		"user": { "id": 2, "name": "Anak Kos Contoh", "role": "tenant", "is_verified": true }
	}`
	dynamic := map[string]*regexp.Regexp{
		// The numeric prefix is the access_tokens row id (an independent
		// sequence), not the user id — API.md's literal example token just
		// happens to show "2" for its own unrelated fixture data.
		"$.token": regexp.MustCompile(`^\d+\|[A-Za-z0-9]{48}$`),
	}
	AssertJSONMatches(t, body, []byte(want), dynamic)
}

// TestLoginWrongPasswordFixture proves the 401 envelope matches API.md §2.4's
// documented failure shape, with `messages: null` (SPEC-GO.md §5.3).
func TestLoginWrongPasswordFixture(t *testing.T) {
	srv := newTestServer(t)

	resp, body := postJSON(t, srv, "/api/auth/login", `{"phone_number":"081200000002","password":"wrong-password"}`)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, body)
	}

	want := `{
		"status": false,
		"meta": { "response_code": 401, "code": 401, "severity": "WARNING", "message": "Nomor handphone atau password salah." },
		"messages": null
	}`
	AssertJSONMatches(t, body, []byte(want), nil)
}
