package otp

import (
	"context"
	"errors"
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
	err   error
}

func (s *recordingSender) Send(_ context.Context, destination, code string) error {
	s.calls = append(s.calls, destination+"|"+code)
	return s.err
}

func testKeys(t *testing.T) (codeKey, cooldownKey string) {
	t.Helper()
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	return "test:code:" + id, "test:cooldown:" + id
}

func whatsappOnly(sender Sender) map[Channel]ChannelConfig {
	return map[Channel]ChannelConfig{ChannelWhatsApp: {Sender: sender}}
}

func testParams(t *testing.T) RequestParams {
	t.Helper()
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	return RequestParams{
		Channel: ChannelWhatsApp, CodeKey: "test:code:" + id, CooldownKey: "test:cooldown:" + id,
		Destination: "0812" + id[len(id)-8:], ClientIP: "198.51.100." + id[len(id)-2:],
	}
}

func TestRequestThenCheckRoundTrip(t *testing.T) {
	rdb := newTestClient(t)
	sender := &recordingSender{}
	svc := NewService(rdb, whatsappOnly(sender), Config{CodeLength: 4, TTL: 300 * time.Second, ResendAfter: 60 * time.Second})
	ctx := context.Background()
	codeKey, cooldownKey := testKeys(t)

	result, err := svc.Request(ctx, RequestParams{Channel: ChannelWhatsApp, CodeKey: codeKey, CooldownKey: cooldownKey, Destination: "081200000002"})
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if len(result.Code) != 4 {
		t.Errorf("Code = %q, want length 4", result.Code)
	}
	if earliest := time.Now().Add(60 * time.Second); time.Unix(result.CanResendAt, 0).Before(earliest.Truncate(time.Second)) {
		t.Errorf("CanResendAt = %d, must not be before the cooldown ends (%v)", result.CanResendAt, earliest)
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
	svc := NewService(rdb, whatsappOnly(&recordingSender{}), Config{CodeLength: 4, TTL: 300 * time.Second, ResendAfter: 60 * time.Second})
	ctx := context.Background()
	codeKey, cooldownKey := testKeys(t)

	if ok, err := svc.Check(ctx, codeKey, "0000"); err != nil || ok {
		t.Errorf("Check on never-requested key: ok=%v err=%v, want false/nil", ok, err)
	}

	result, err := svc.Request(ctx, RequestParams{Channel: ChannelWhatsApp, CodeKey: codeKey, CooldownKey: cooldownKey, Destination: "081200000002"})
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
	svc := NewService(rdb, whatsappOnly(&recordingSender{}), Config{CodeLength: 4, TTL: 300 * time.Second, ResendAfter: 60 * time.Second})
	ctx := context.Background()
	params := testParams(t)

	if _, err := svc.Request(ctx, params); err != nil {
		t.Fatalf("first Request: %v", err)
	}
	if _, err := svc.Request(ctx, params); !errors.Is(err, ErrCooldown) {
		t.Errorf("second Request err = %v, want ErrCooldown", err)
	}
}

func TestRequestStopsAtDailyLimitPerDestination(t *testing.T) {
	rdb := newTestClient(t)
	sender := &recordingSender{}
	svc := NewService(rdb, whatsappOnly(sender), Config{CodeLength: 4, TTL: time.Minute, ResendAfter: time.Millisecond, MaxPerDay: 2})
	ctx := context.Background()
	params := testParams(t)
	params.ClientIP = ""
	t.Cleanup(func() { rdb.Del(context.Background(), "otp:sent:day:"+params.Destination) })

	for i := range 2 {
		if _, err := svc.Request(ctx, params); err != nil {
			t.Fatalf("Request %d: %v", i+1, err)
		}
		time.Sleep(5 * time.Millisecond)
	}
	if _, err := svc.Request(ctx, params); !errors.Is(err, ErrSendLimit) {
		t.Fatalf("third Request err = %v, want ErrSendLimit", err)
	}
	if len(sender.calls) != 2 {
		t.Errorf("sender called %d times, want 2 (the capped request must not send)", len(sender.calls))
	}
}

func TestRequestStopsAtHourlyLimitPerClientIP(t *testing.T) {
	rdb := newTestClient(t)
	sender := &recordingSender{}
	svc := NewService(rdb, whatsappOnly(sender), Config{CodeLength: 4, TTL: time.Minute, ResendAfter: time.Minute, MaxPerIPPerHour: 1})
	ctx := context.Background()
	first := testParams(t)
	second := testParams(t)
	second.ClientIP = first.ClientIP
	t.Cleanup(func() { rdb.Del(context.Background(), "otp:sent:ip:"+first.ClientIP) })

	if _, err := svc.Request(ctx, first); err != nil {
		t.Fatalf("first Request: %v", err)
	}
	if _, err := svc.Request(ctx, second); !errors.Is(err, ErrSendLimit) {
		t.Fatalf("second Request from the same IP err = %v, want ErrSendLimit", err)
	}
	if len(sender.calls) != 1 {
		t.Errorf("sender called %d times, want 1", len(sender.calls))
	}
}

func TestRequestKeepsCooldownWhenProviderFails(t *testing.T) {
	rdb := newTestClient(t)
	sender := &recordingSender{err: errors.New("gateway down")}
	svc := NewService(rdb, whatsappOnly(sender), Config{CodeLength: 4, TTL: time.Minute, ResendAfter: time.Minute})
	ctx := context.Background()
	params := testParams(t)

	if _, err := svc.Request(ctx, params); !errors.Is(err, ErrSendFailed) {
		t.Fatalf("Request err = %v, want ErrSendFailed", err)
	}
	if _, err := svc.Request(ctx, params); !errors.Is(err, ErrCooldown) {
		t.Errorf("retry right after a provider failure err = %v, want ErrCooldown", err)
	}
	if len(sender.calls) != 1 {
		t.Errorf("sender called %d times, want 1", len(sender.calls))
	}
}

func TestUnixCeilNeverRoundsDown(t *testing.T) {
	exact := time.Unix(1_800_000_000, 0)
	if got := unixCeil(exact); got != 1_800_000_000 {
		t.Errorf("unixCeil(whole second) = %d", got)
	}
	if got := unixCeil(exact.Add(time.Millisecond)); got != 1_800_000_001 {
		t.Errorf("unixCeil(just past a second) = %d, want the next second", got)
	}
}

func TestRequestRoutesToTheChosenChannel(t *testing.T) {
	rdb := newTestClient(t)
	whatsapp := &recordingSender{}
	sms := &recordingSender{}
	svc := NewService(rdb, map[Channel]ChannelConfig{ChannelWhatsApp: {Sender: whatsapp}, ChannelSMS: {Sender: sms}},
		Config{CodeLength: 4, TTL: time.Minute, ResendAfter: time.Minute})
	params := testParams(t)
	params.Channel = ChannelSMS

	if _, err := svc.Request(context.Background(), params); err != nil {
		t.Fatalf("Request: %v", err)
	}
	if len(sms.calls) != 1 || len(whatsapp.calls) != 0 {
		t.Errorf("sms calls = %d, whatsapp calls = %d, want 1 and 0", len(sms.calls), len(whatsapp.calls))
	}
	if got := svc.Channels(); len(got) != 2 || got[0] != ChannelWhatsApp || got[1] != ChannelSMS {
		t.Errorf("Channels() = %v, want [whatsapp sms]", got)
	}
}

func TestRequestRejectsAnUnconfiguredChannel(t *testing.T) {
	rdb := newTestClient(t)
	svc := NewService(rdb, whatsappOnly(&recordingSender{}), Config{CodeLength: 4, TTL: time.Minute, ResendAfter: time.Minute})
	params := testParams(t)
	params.Channel = ChannelSMS

	if _, err := svc.Request(context.Background(), params); !errors.Is(err, ErrChannelUnavailable) {
		t.Fatalf("Request err = %v, want ErrChannelUnavailable", err)
	}
	if svc.HasChannel(ChannelSMS) {
		t.Error("HasChannel(sms) = true for a WhatsApp-only service")
	}
}

func TestChannelDailyLimitRefundsTheOtherBudgets(t *testing.T) {
	rdb := newTestClient(t)
	ctx := context.Background()
	channel := Channel(fmt.Sprintf("sms-test-%d", time.Now().UnixNano()))
	sender := &recordingSender{}
	svc := NewService(rdb, map[Channel]ChannelConfig{channel: {Sender: sender, MaxPerDay: 1}},
		Config{CodeLength: 4, TTL: time.Minute, ResendAfter: time.Minute, MaxPerDay: 5})
	t.Cleanup(func() { rdb.Del(context.Background(), ChannelDayKey(channel, time.Now())) })

	first := testParams(t)
	first.Channel = channel
	if _, err := svc.Request(ctx, first); err != nil {
		t.Fatalf("first Request: %v", err)
	}
	second := testParams(t)
	second.Channel = channel
	t.Cleanup(func() {
		rdb.Del(context.Background(), "otp:sent:day:"+second.Destination, "otp:sent:day:"+first.Destination)
	})
	if _, err := svc.Request(ctx, second); !errors.Is(err, ErrChannelLimit) {
		t.Fatalf("second Request err = %v, want ErrChannelLimit", err)
	}

	if used, _ := rdb.Get(ctx, "otp:sent:day:"+second.Destination).Int(); used != 0 {
		t.Errorf("per-number budget after a channel refusal = %d, want 0 (refunded)", used)
	}
	if exists, _ := rdb.Exists(ctx, second.CooldownKey).Result(); exists != 0 {
		t.Error("a refused channel must not start the cooldown")
	}
	if len(sender.calls) != 1 {
		t.Errorf("sender called %d times, want 1", len(sender.calls))
	}
}

func TestFailingChannelReleasesCooldownWhenConfigured(t *testing.T) {
	rdb := newTestClient(t)
	ctx := context.Background()
	failing := &recordingSender{err: errors.New("gateway offline")}
	svc := NewService(rdb, map[Channel]ChannelConfig{
		ChannelSMS:      {Sender: failing, ReleaseCooldownOnFailure: true},
		ChannelWhatsApp: {Sender: &recordingSender{}},
	}, Config{CodeLength: 4, TTL: time.Minute, ResendAfter: time.Minute, MaxPerDay: 5})
	params := testParams(t)
	params.Channel = ChannelSMS
	t.Cleanup(func() { rdb.Del(context.Background(), "otp:sent:day:"+params.Destination) })

	if _, err := svc.Request(ctx, params); !errors.Is(err, ErrSendFailed) {
		t.Fatalf("SMS Request err = %v, want ErrSendFailed", err)
	}
	if used, _ := rdb.Get(ctx, "otp:sent:day:"+params.Destination).Int(); used != 0 {
		t.Errorf("per-number budget after a failed SMS = %d, want 0 (refunded)", used)
	}
	params.Channel = ChannelWhatsApp
	if _, err := svc.Request(ctx, params); err != nil {
		t.Fatalf("switching to WhatsApp right after a failed SMS: %v", err)
	}
}
