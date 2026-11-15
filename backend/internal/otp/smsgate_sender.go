package otp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultSMSGateBaseURL = "https://api.sms-gate.app/3rdparty/v1"
	smsGatePriorityUrgent = 100
)

type SMSGateConfig struct {
	BaseURL   string
	Username  string
	Password  string
	DeviceID  string
	SIMNumber int
	CodeTTL   time.Duration
}

type SMSGateSender struct {
	cfg  SMSGateConfig
	HTTP *http.Client
}

func NewSMSGateSender(cfg SMSGateConfig) SMSGateSender {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultSMSGateBaseURL
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return SMSGateSender{cfg: cfg, HTTP: &http.Client{Timeout: 10 * time.Second}}
}

type smsGateTextMessage struct {
	Text string `json:"text"`
}

type smsGateMessage struct {
	TextMessage        smsGateTextMessage `json:"textMessage"`
	PhoneNumbers       []string           `json:"phoneNumbers"`
	TTL                int                `json:"ttl"`
	Priority           int                `json:"priority"`
	WithDeliveryReport bool               `json:"withDeliveryReport"`
	DeviceID           string             `json:"deviceId,omitempty"`
	SIMNumber          int                `json:"simNumber,omitempty"`
}

type smsGateError struct {
	Message string `json:"message"`
}

func (s SMSGateSender) Send(ctx context.Context, destination, code string) error {
	payload, err := json.Marshal(smsGateMessage{
		TextMessage:  smsGateTextMessage{Text: SMSText(code, s.cfg.CodeTTL)},
		PhoneNumbers: []string{InternationalPhone(destination)},
		TTL:          int(s.cfg.CodeTTL.Seconds()),
		Priority:     smsGatePriorityUrgent,
		DeviceID:     s.cfg.DeviceID,
		SIMNumber:    s.cfg.SIMNumber,
	})
	if err != nil {
		return fmt.Errorf("encode sms gateway request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.BaseURL+"/messages", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build sms gateway request: %w", err)
	}
	req.SetBasicAuth(s.cfg.Username, s.cfg.Password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("call sms gateway: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	var parsed smsGateError
	_ = json.Unmarshal(body, &parsed)
	return fmt.Errorf("%w: sms gateway http %d: %s", ErrProviderRejected, resp.StatusCode, parsed.Message)
}

func SMSText(code string, ttl time.Duration) string {
	return fmt.Sprintf("Kode verifikasi Sewa Kos: %s. Berlaku %d menit. Jangan berikan kode ini kepada siapa pun.", code, int(ttl.Minutes()))
}

func InternationalPhone(destination string) string {
	if strings.HasPrefix(destination, "0") {
		return "+62" + destination[1:]
	}
	return destination
}
