package otp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestFazpassSenderSendsOwnCodeOnce(t *testing.T) {
	var calls atomic.Int32
	var got fazpassSendRequest
	var auth, path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		path = r.URL.Path
		auth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"status":true,"message":"Success","data":{"id":"abc"}}`))
	}))
	defer server.Close()

	sender := NewFazpassSender(server.URL+"/", "merchant-key", "gateway-key")
	if err := sender.Send(context.Background(), "081234567890", "1234"); err != nil {
		t.Fatalf("Send: %v", err)
	}

	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1", calls.Load())
	}
	if path != "/v1/otp/send" {
		t.Errorf("path = %q, want /v1/otp/send", path)
	}
	if auth != "Bearer merchant-key" {
		t.Errorf("Authorization = %q", auth)
	}
	if got.Phone != "6281234567890" || got.OTP != "1234" || got.GatewayKey != "gateway-key" {
		t.Errorf("payload = %+v", got)
	}
}

func TestFazpassSenderDoesNotRetryFailures(t *testing.T) {
	cases := map[string]struct {
		status int
		body   string
	}{
		"http error":    {http.StatusInternalServerError, `{"status":false,"message":"boom"}`},
		"status false":  {http.StatusOK, `{"status":false,"message":"Gateway not found"}`},
		"not json body": {http.StatusOK, `<html>bad gateway</html>`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			sender := NewFazpassSender(server.URL, "merchant-key", "gateway-key")
			if err := sender.Send(context.Background(), "081234567890", "1234"); err == nil {
				t.Fatal("Send succeeded, want error")
			}
			if calls.Load() != 1 {
				t.Errorf("calls = %d, want exactly 1 (no retry)", calls.Load())
			}
		})
	}
}

func TestFazpassPhone(t *testing.T) {
	if got := FazpassPhone("081234567890"); got != "6281234567890" {
		t.Errorf("FazpassPhone(08...) = %q", got)
	}
	if got := FazpassPhone("6281234567890"); got != "6281234567890" {
		t.Errorf("FazpassPhone(62...) = %q", got)
	}
}
