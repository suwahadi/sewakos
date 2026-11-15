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

const DefaultFazpassBaseURL = "https://api.fazpass.com"

type FazpassSender struct {
	BaseURL     string
	MerchantKey string
	GatewayKey  string
	HTTP        *http.Client
}

func NewFazpassSender(baseURL, merchantKey, gatewayKey string) FazpassSender {
	if baseURL == "" {
		baseURL = DefaultFazpassBaseURL
	}
	return FazpassSender{
		BaseURL:     strings.TrimRight(baseURL, "/"),
		MerchantKey: merchantKey,
		GatewayKey:  gatewayKey,
		HTTP:        &http.Client{Timeout: 10 * time.Second},
	}
}

type fazpassSendRequest struct {
	Phone      string `json:"phone"`
	OTP        string `json:"otp"`
	GatewayKey string `json:"gateway_key"`
}

type fazpassResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
}

func (s FazpassSender) Send(ctx context.Context, destination, code string) error {
	payload, err := json.Marshal(fazpassSendRequest{
		Phone: FazpassPhone(destination), OTP: code, GatewayKey: s.GatewayKey,
	})
	if err != nil {
		return fmt.Errorf("encode fazpass request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.BaseURL+"/v1/otp/send", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build fazpass request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.MerchantKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("call fazpass: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return fmt.Errorf("read fazpass response: %w", err)
	}

	var parsed fazpassResponse
	_ = json.Unmarshal(body, &parsed)
	if resp.StatusCode < 200 || resp.StatusCode > 299 || !parsed.Status {
		return fmt.Errorf("fazpass rejected send: http %d: %s", resp.StatusCode, parsed.Message)
	}
	return nil
}

func FazpassPhone(destination string) string {
	if strings.HasPrefix(destination, "0") {
		return "62" + destination[1:]
	}
	return destination
}
