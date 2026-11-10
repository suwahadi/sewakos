// Package otp implements the Redis-backed verification code mechanism shared
// by every OTP flow (API.md §2.2-2.3, §11.6; SPEC-GO.md §6.3-6.4): generate,
// store, send, and one-time-check a code at whatever Redis key the caller
// supplies. It carries no business rules of its own (who may request a code,
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

// Channel is the delivery channel ("0"=SMS, "1"=WhatsApp per API.md §2.2).
type Channel string

const (
	ChannelSMS      Channel = "sms"
	ChannelWhatsApp Channel = "whatsapp"
)

// Sender delivers one OTP message. SPEC-GO.md §6.3: sending happens
// synchronously inside the request; only LogSender exists until a real
// SMS/WhatsApp provider is chosen (PRD.md §8 #1).
type Sender interface {
	Send(ctx context.Context, channel Channel, destination, message string) error
}

// ErrCooldown is returned by Request when the resend cooldown hasn't elapsed.
var ErrCooldown = errors.New("otp: cooldown active")

// Config holds the SPEC-GO.md §3 OTP settings a Service needs.
type Config struct {
	CodeLength  int
	TTL         time.Duration
	ResendAfter time.Duration
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

// RequestResult is what Request returns on success.
type RequestResult struct {
	Code                  string
	CanResendAt           int64
	CanResendAfterSeconds int
}

// Request generates a new code, stores it at codeKey (TTL cfg.TTL), sets the
// cooldown marker at cooldownKey (TTL cfg.ResendAfter), and sends it over
// channel to destination. It returns ErrCooldown without touching anything if
// cooldownKey is already set (API.md §2.2: 60s resend cooldown).
func (s *Service) Request(ctx context.Context, codeKey, cooldownKey string, channel Channel, destination string) (RequestResult, error) {
	exists, err := s.rdb.Exists(ctx, cooldownKey).Result()
	if err != nil {
		return RequestResult{}, fmt.Errorf("check otp cooldown: %w", err)
	}
	if exists > 0 {
		return RequestResult{}, ErrCooldown
	}

	code, err := generateCode(s.cfg.CodeLength)
	if err != nil {
		return RequestResult{}, fmt.Errorf("generate otp code: %w", err)
	}

	if err := s.rdb.Set(ctx, codeKey, code, s.cfg.TTL).Err(); err != nil {
		return RequestResult{}, fmt.Errorf("store otp code: %w", err)
	}
	if err := s.rdb.Set(ctx, cooldownKey, "1", s.cfg.ResendAfter).Err(); err != nil {
		return RequestResult{}, fmt.Errorf("store otp cooldown: %w", err)
	}

	message := fmt.Sprintf("Kode verifikasi Sewa Kos kamu: %s", code)
	if err := s.sender.Send(ctx, channel, destination, message); err != nil {
		return RequestResult{}, fmt.Errorf("send otp: %w", err)
	}

	return RequestResult{
		Code:                  code,
		CanResendAt:           time.Now().Add(s.cfg.ResendAfter).Unix(),
		CanResendAfterSeconds: int(s.cfg.ResendAfter.Seconds()),
	}, nil
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
