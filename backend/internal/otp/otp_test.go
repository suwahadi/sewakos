package otp

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

type recordingSender struct {
	calls []string
}

func (s *recordingSender) Send(_ context.Context, channel Channel, destination, message string) error {
	s.calls = append(s.calls, fmt.Sprintf("%s|%s|%s", channel, destination, message))
	return nil
}

func testKeys(t *testing.T) (codeKey, cooldownKey string) {
	t.Helper()
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	return "test:code:" + id, "test:cooldown:" + id
}

func TestRequestThenCheckRoundTrip(t *testing.T) {
	rdb := newTestClient(t)
	sender := &recordingSender{}
	svc := NewService(rdb, sender, Config{CodeLength: 4, TTL: 300 * time.Second, ResendAfter: 60 * time.Second})
	ctx := context.Background()
	codeKey, cooldownKey := testKeys(t)

	result, err := svc.Request(ctx, codeKey, cooldownKey, ChannelSMS, "081200000002")
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if len(result.Code) != 4 {
		t.Errorf("Code = %q, want length 4", result.Code)
	}
	if result.CanResendAfterSeconds != 60 {
		t.Errorf("CanResendAfterSeconds = %d, want 60", result.CanResendAfterSeconds)
	}
	if len(sender.calls) != 1 {
		t.Fatalf("sender called %d times, want 1", len(sender.calls))
	}

	ok, err := svc.Check(ctx, codeKey, result.Code)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !ok {
		t.Fatal("Check with the correct code = false, want true")
	}

	// One-time use: checking the same code again must fail since the key was deleted.
	ok, err = svc.Check(ctx, codeKey, result.Code)
	if err != nil {
		t.Fatalf("Check (reuse): %v", err)
	}
	if ok {
		t.Error("Check reused a consumed code, want false")
	}
}

func TestCheckRejectsWrongOrMissingCode(t *testing.T) {
	rdb := newTestClient(t)
	svc := NewService(rdb, &recordingSender{}, Config{CodeLength: 4, TTL: 300 * time.Second, ResendAfter: 60 * time.Second})
	ctx := context.Background()
	codeKey, cooldownKey := testKeys(t)

	if ok, err := svc.Check(ctx, codeKey, "0000"); err != nil || ok {
		t.Errorf("Check on never-requested key: ok=%v err=%v, want false/nil", ok, err)
	}

	result, err := svc.Request(ctx, codeKey, cooldownKey, ChannelSMS, "081200000002")
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	wrong := "0000"
	if wrong == result.Code {
		wrong = "1111"
	}
	if ok, err := svc.Check(ctx, codeKey, wrong); err != nil || ok {
		t.Errorf("Check with wrong code: ok=%v err=%v, want false/nil", ok, err)
	}
}

func TestRequestEnforcesCooldown(t *testing.T) {
	rdb := newTestClient(t)
	svc := NewService(rdb, &recordingSender{}, Config{CodeLength: 4, TTL: 300 * time.Second, ResendAfter: 60 * time.Second})
	ctx := context.Background()
	codeKey, cooldownKey := testKeys(t)

	if _, err := svc.Request(ctx, codeKey, cooldownKey, ChannelSMS, "081200000002"); err != nil {
		t.Fatalf("first Request: %v", err)
	}
	if _, err := svc.Request(ctx, codeKey, cooldownKey, ChannelSMS, "081200000002"); err != ErrCooldown {
		t.Errorf("second Request err = %v, want ErrCooldown", err)
	}
}
