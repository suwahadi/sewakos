// Package otp implements the Redis-backed verification code mechanism shared
// by every OTP flow: generate, store, send, and one-time-check a code at
// whatever Redis key the caller supplies. It carries no business rules of its
// own (who may request a code,
// what happens after a correct check), so the same Service backs both the
// tenant_verification/login/reset_password flow (keyed
// "otp:code:{for}:{destination}") and the change-phone flow (keyed
// "otp:code:change_phone:{user_id}:{new_phone}").
package otp

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
)

type Sender interface {
	Send(ctx context.Context, destination, code string) error
}

type Channel string

const (
	ChannelWhatsApp Channel = "whatsapp"
	ChannelSMS      Channel = "sms"
)

type ReleasePolicy int

const (
	KeepCooldownOnFailure ReleasePolicy = iota
	ReleaseCooldownWhenRejected
	ReleaseCooldownOnAnyFailure
)

type ChannelConfig struct {
	Sender           Sender
	MaxPerDay        int
	ReleaseOnFailure ReleasePolicy
}

func (c ChannelConfig) releasesAfter(err error) bool {
	switch c.ReleaseOnFailure {
	case ReleaseCooldownOnAnyFailure:
		return true
	case ReleaseCooldownWhenRejected:
		return errors.Is(err, ErrProviderRejected)
	default:
		return false
	}
}

var (
	ErrCooldown           = errors.New("otp: cooldown active")
	ErrSendLimit          = errors.New("otp: send limit reached")
	ErrSendFailed         = errors.New("otp: provider failed to send")
	ErrProviderRejected   = errors.New("otp: provider rejected the message")
	ErrCooldownReleased   = errors.New("otp: cooldown released after the failure")
	ErrChannelUnavailable = errors.New("otp: channel not configured")
	ErrChannelLimit       = errors.New("otp: channel daily limit reached")
)

var wib = time.FixedZone("WIB", 7*60*60)

// CodeKey builds the Redis key for a tenant_verification/login/reset_password
// code (SPEC-GO.md §6.3: "otp:code:{for}:{destination}").
func CodeKey(forType, destination string) string {
	return "otp:code:" + forType + ":" + destination
}

// ChangePhoneCodeKey builds the Redis key for a change-phone code
// (SPEC-GO.md §6.4: "otp:code:change_phone:{user_id}:{new_phone}").
func ChangePhoneCodeKey(userID int64, newPhone string) string {
	return fmt.Sprintf("otp:code:change_phone:%d:%s", userID, newPhone)
}

// CooldownKey builds the Redis resend-cooldown key, shared across every "for"
// value and the change-phone flow for the same destination (SPEC-GO.md §6.3:
// "otp:cooldown:{destination}" has no "for" component).
func CooldownKey(destination string) string {
	return "otp:cooldown:" + destination
}

type Config struct {
	CodeLength      int
	TTL             time.Duration
	ResendAfter     time.Duration
	MaxPerDay       int
	MaxPerIPPerHour int
}

// Service generates, stores, sends, and verifies OTP codes.
type Service struct {
	rdb      *redis.Client
	channels map[Channel]ChannelConfig
	cfg      Config
}

func NewService(rdb *redis.Client, channels map[Channel]ChannelConfig, cfg Config) *Service {
	return &Service{rdb: rdb, channels: channels, cfg: cfg}
}

func (s *Service) Channels() []Channel {
	ordered := make([]Channel, 0, len(s.channels))
	for _, channel := range []Channel{ChannelWhatsApp, ChannelSMS} {
		if _, ok := s.channels[channel]; ok {
			ordered = append(ordered, channel)
		}
	}
	return ordered
}

func (s *Service) HasChannel(channel Channel) bool {
	_, ok := s.channels[channel]
	return ok
}

type RequestParams struct {
	Channel     Channel
	CodeKey     string
	CooldownKey string
	Destination string
	ClientIP    string
}

type RequestResult struct {
	Code                  string
	CanResendAt           int64
	CanResendAfterSeconds int
}

func (s *Service) Request(ctx context.Context, p RequestParams) (RequestResult, error) {
	channel, ok := s.channels[p.Channel]
	if !ok {
		return RequestResult{}, ErrChannelUnavailable
	}

	exists, err := s.rdb.Exists(ctx, p.CooldownKey).Result()
	if err != nil {
		return RequestResult{}, fmt.Errorf("check otp cooldown: %w", err)
	}
	if exists > 0 {
		return RequestResult{}, ErrCooldown
	}

	budgets := []sendBudget{{key: "otp:sent:day:" + p.Destination, limit: s.cfg.MaxPerDay, window: 24 * time.Hour, exceededErr: ErrSendLimit}}
	if p.ClientIP != "" {
		budgets = append(budgets, sendBudget{key: "otp:sent:ip:" + p.ClientIP, limit: s.cfg.MaxPerIPPerHour, window: time.Hour, exceededErr: ErrSendLimit})
	}
	budgets = append(budgets, sendBudget{
		key: ChannelDayKey(p.Channel, time.Now()), limit: channel.MaxPerDay, window: 48 * time.Hour, exceededErr: ErrChannelLimit,
	})

	consumed, err := s.consumeBudgets(ctx, budgets)
	if err != nil {
		return RequestResult{}, err
	}

	code, err := generateCode(s.cfg.CodeLength)
	if err != nil {
		return RequestResult{}, fmt.Errorf("generate otp code: %w", err)
	}

	if err := s.rdb.Set(ctx, p.CodeKey, code, s.cfg.TTL).Err(); err != nil {
		return RequestResult{}, fmt.Errorf("store otp code: %w", err)
	}
	if err := s.rdb.Set(ctx, p.CooldownKey, "1", s.cfg.ResendAfter).Err(); err != nil {
		return RequestResult{}, fmt.Errorf("store otp cooldown: %w", err)
	}
	canResendAt := unixCeil(time.Now().Add(s.cfg.ResendAfter))

	if err := channel.Sender.Send(ctx, p.Destination, code); err != nil {
		if channel.releasesAfter(err) {
			s.refund(ctx, consumed)
			_ = s.rdb.Del(ctx, p.CooldownKey, p.CodeKey).Err()
			return RequestResult{}, fmt.Errorf("%w: %w: %w", ErrSendFailed, ErrCooldownReleased, err)
		}
		return RequestResult{}, fmt.Errorf("%w: %w", ErrSendFailed, err)
	}

	return RequestResult{
		Code:                  code,
		CanResendAt:           canResendAt,
		CanResendAfterSeconds: int(s.cfg.ResendAfter.Seconds()),
	}, nil
}

func unixCeil(t time.Time) int64 {
	if t.Nanosecond() > 0 {
		return t.Unix() + 1
	}
	return t.Unix()
}

func ChannelDayKey(channel Channel, at time.Time) string {
	return "otp:sent:channel:" + string(channel) + ":" + at.In(wib).Format("2006-01-02")
}

type sendBudget struct {
	key         string
	limit       int
	window      time.Duration
	exceededErr error
}

func (s *Service) consumeBudgets(ctx context.Context, budgets []sendBudget) ([]string, error) {
	var consumed []string
	for _, budget := range budgets {
		if budget.limit <= 0 {
			continue
		}
		exceeded, err := s.consumeSendBudget(ctx, budget.key, budget.limit, budget.window)
		if err != nil {
			s.refund(ctx, consumed)
			return nil, err
		}
		if exceeded {
			if errors.Is(budget.exceededErr, ErrChannelLimit) {
				s.refund(ctx, append(consumed, budget.key))
			}
			return nil, budget.exceededErr
		}
		consumed = append(consumed, budget.key)
	}
	return consumed, nil
}

func (s *Service) consumeSendBudget(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	count, err := s.rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("count otp sends: %w", err)
	}
	if count == 1 {
		if err := s.rdb.Expire(ctx, key, window).Err(); err != nil {
			return false, fmt.Errorf("expire otp send counter: %w", err)
		}
	}
	return count > int64(limit), nil
}

func (s *Service) refund(ctx context.Context, keys []string) {
	for _, key := range keys {
		_ = s.rdb.Decr(ctx, key).Err()
	}
}

// Check compares code against the value stored at codeKey. On match, the key
// is deleted (one-time use) and ok is true; otherwise ok is false (wrong
// code, expired, or never requested) and codeKey is left untouched.
func (s *Service) Check(ctx context.Context, codeKey, code string) (bool, error) {
	stored, err := s.rdb.Get(ctx, codeKey).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("get otp code: %w", err)
	}
	if stored != code {
		return false, nil
	}
	if err := s.rdb.Del(ctx, codeKey).Err(); err != nil {
		return false, fmt.Errorf("delete otp code: %w", err)
	}
	return true, nil
}

// generateCode returns a length-digit numeric code (crypto/rand), allowed to
// start with 0 (SPEC-GO.md §6.3).
func generateCode(length int) (string, error) {
	digits := make([]byte, length)
	for i := range digits {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		digits[i] = byte('0' + n.Int64())
	}
	return string(digits), nil
}
