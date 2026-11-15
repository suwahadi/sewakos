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

var (
	ErrCooldown   = errors.New("otp: cooldown active")
	ErrSendLimit  = errors.New("otp: send limit reached")
	ErrSendFailed = errors.New("otp: provider failed to send")
)

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
	rdb    *redis.Client
	sender Sender
	cfg    Config
}

func NewService(rdb *redis.Client, sender Sender, cfg Config) *Service {
	return &Service{rdb: rdb, sender: sender, cfg: cfg}
}

type RequestParams struct {
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
	exists, err := s.rdb.Exists(ctx, p.CooldownKey).Result()
	if err != nil {
		return RequestResult{}, fmt.Errorf("check otp cooldown: %w", err)
	}
	if exists > 0 {
		return RequestResult{}, ErrCooldown
	}

	if err := s.consumeSendBudget(ctx, "otp:sent:day:"+p.Destination, s.cfg.MaxPerDay, 24*time.Hour); err != nil {
		return RequestResult{}, err
	}
	if p.ClientIP != "" {
		if err := s.consumeSendBudget(ctx, "otp:sent:ip:"+p.ClientIP, s.cfg.MaxPerIPPerHour, time.Hour); err != nil {
			return RequestResult{}, err
		}
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

	if err := s.sender.Send(ctx, p.Destination, code); err != nil {
		return RequestResult{}, fmt.Errorf("%w: %w", ErrSendFailed, err)
	}

	return RequestResult{
		Code:                  code,
		CanResendAt:           time.Now().Add(s.cfg.ResendAfter).Unix(),
		CanResendAfterSeconds: int(s.cfg.ResendAfter.Seconds()),
	}, nil
}

func (s *Service) consumeSendBudget(ctx context.Context, key string, limit int, window time.Duration) error {
	if limit <= 0 {
		return nil
	}
	count, err := s.rdb.Incr(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("count otp sends: %w", err)
	}
	if count == 1 {
		if err := s.rdb.Expire(ctx, key, window).Err(); err != nil {
			return fmt.Errorf("expire otp send counter: %w", err)
		}
	}
	if count > int64(limit) {
		return ErrSendLimit
	}
	return nil
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
