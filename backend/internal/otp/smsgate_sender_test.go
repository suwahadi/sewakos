package otp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSMSGateSenderEnqueuesAnUrgentExpiringMessage(t *testing.T) {
	var calls atomic.Int32
	var got smsGateMessage
	var path, user, pass string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		path = r.URL.Path
		user, pass, _ = r.BasicAuth()
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"id":"abc","state":"Pending","deviceId":"dev","createdAt":"2026-01-01T00:00:00Z","recipients":[]}`))
	}))
	defer server.Close()

	sender := NewSMSGateSender(SMSGateConfig{
		BaseURL: server.URL + "/3rdparty/v1/", Username: "user", Password: "secret", SIMNumber: 2, CodeTTL: 5 * time.Minute,
	})
	if err := sender.Send(context.Background(), "081234567890", "1234"); err != nil {
		t.Fatalf("Send: %v", err)
	}

	if calls.Load() != 1 || path != "/3rdparty/v1/messages" {
		t.Errorf("calls = %d, path = %q", calls.Load(), path)
	}
	if user != "user" || pass != "secret" {
		t.Errorf("basic auth = %q/%q", user, pass)
	}
	if len(got.PhoneNumbers) != 1 || got.PhoneNumbers[0] != "+6281234567890" {
		t.Errorf("phoneNumbers = %v", got.PhoneNumbers)
	}
	if got.TTL != 300 || got.Priority != 100 || got.WithDeliveryReport || got.SIMNumber != 2 || got.DeviceID != "" {
		t.Errorf("message options = %+v", got)
	}
	if !strings.Contains(got.TextMessage.Text, "1234") || !strings.Contains(got.TextMessage.Text, "5 menit") {
		t.Errorf("text = %q", got.TextMessage.Text)
	}
}

func TestSMSGateSenderDoesNotRetryFailures(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusServiceUnavailable} {
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"code":0,"message":"queue limits exceeded"}`))
		}))

		sender := NewSMSGateSender(SMSGateConfig{BaseURL: server.URL, Username: "u", Password: "p", CodeTTL: time.Minute})
		err := sender.Send(context.Background(), "081234567890", "1234")
		server.Close()
		if !errors.Is(err, ErrProviderRejected) {
			t.Errorf("status %d: Send err = %v, want ErrProviderRejected", status, err)
		}
		if calls.Load() != 1 {
			t.Errorf("status %d: calls = %d, want exactly 1 (no retry)", status, calls.Load())
		}
	}
}

func TestSMSTextStaysInOneGSM7Segment(t *testing.T) {
	text := SMSText("0123", 5*time.Minute)
	if len(text) > 160 {
		t.Errorf("len = %d, want <= 160", len(text))
	}
	for _, r := range text {
		if r > 0x7E || (r < 0x20 && r != '\n') {
			t.Fatalf("non GSM-7 friendly character %q in %q", r, text)
		}
	}
}

func TestInternationalPhone(t *testing.T) {
	if got := InternationalPhone("081234567890"); got != "+6281234567890" {
		t.Errorf("InternationalPhone(08...) = %q", got)
	}
}
