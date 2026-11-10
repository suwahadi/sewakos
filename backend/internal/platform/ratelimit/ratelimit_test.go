package ratelimit

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func newTestClient(t *testing.T) *redis.Client {
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

func TestAllowEnforcesLimitWithinWindow(t *testing.T) {
	rdb := newTestClient(t)
	ctx := context.Background()
	key := fmt.Sprintf("test-%d", time.Now().UnixNano())

	for i := int64(1); i <= 2; i++ {
		result, err := Allow(ctx, rdb, "otp", key, 2, time.Minute)
		if err != nil {
			t.Fatalf("Allow: %v", err)
		}
		if !result.Allowed || result.Count != i {
			t.Fatalf("call %d: Allowed=%v Count=%d, want Allowed=true Count=%d", i, result.Allowed, result.Count, i)
		}
	}

	result, err := Allow(ctx, rdb, "otp", key, 2, time.Minute)
	if err != nil {
		t.Fatalf("Allow: %v", err)
	}
	if result.Allowed || result.Remaining != 0 {
		t.Errorf("3rd call over a limit of 2: Allowed=%v Remaining=%d, want false/0", result.Allowed, result.Remaining)
	}
}

func TestAllowKeysAreIndependentPerScopeAndKey(t *testing.T) {
	rdb := newTestClient(t)
	ctx := context.Background()
	key := fmt.Sprintf("test-%d", time.Now().UnixNano())

	if _, err := Allow(ctx, rdb, "api", key, 1, time.Minute); err != nil {
		t.Fatalf("Allow: %v", err)
	}
	result, err := Allow(ctx, rdb, "otp", key, 1, time.Minute)
	if err != nil {
		t.Fatalf("Allow: %v", err)
	}
	if !result.Allowed {
		t.Error("a different scope with the same key must have its own counter")
	}
}
