package email

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestNewSenderPicksTheImplementation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cases := []struct {
		name string
		cfg  SMTPConfig
		smtp bool
		err  bool
	}{
		{"no host logs", SMTPConfig{}, false, false},
		{"username without password logs", SMTPConfig{Host: "smtp-relay.brevo.com", Username: "login@smtp-brevo.com", From: "Sewa Kos <no-reply@sewakos.web.id>"}, false, false},
		{"missing from is an error", SMTPConfig{Host: "smtp-relay.brevo.com", Username: "login", Password: "key"}, false, true},
		{"full config sends", SMTPConfig{Host: "smtp-relay.brevo.com", Username: "login", Password: "key", From: "Sewa Kos <no-reply@sewakos.web.id>"}, true, false},
	}
	for _, tc := range cases {
		sender, err := NewSender(tc.cfg, logger)
		if (err != nil) != tc.err {
			t.Errorf("%s: err = %v", tc.name, err)
			continue
		}
		if _, isSMTP := sender.(*SMTPSender); err == nil && isSMTP != tc.smtp {
			t.Errorf("%s: SMTP sender = %v, want %v", tc.name, isSMTP, tc.smtp)
		}
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func TestSMTPSenderDeliversToSandbox(t *testing.T) {
	addr := envOr("TEST_SMTP_ADDR", "localhost:1025")
	api := envOr("TEST_MAILPIT_URL", "http://localhost:8025")
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Skipf("SMTP sandbox not reachable at %s: %v", addr, err)
	}
	conn.Close()
	host, portText, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(portText)

	sender, err := NewSender(SMTPConfig{Host: host, Port: port, From: "Sewa Kos <no-reply@sewakos.web.id>"}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	subject := fmt.Sprintf("Uji SMTP %d", time.Now().UnixNano())
	if err := sender.Send(context.Background(), "penyewa@example.com", subject, "Pembayaran berhasil."); err != nil {
		t.Fatalf("send: %v", err)
	}

	resp, err := http.Get(api + "/api/v1/search?query=" + url.QueryEscape(fmt.Sprintf("subject:%q", subject)))
	if err != nil {
		t.Fatalf("query sandbox: %v", err)
	}
	defer resp.Body.Close()
	var found struct {
		Messages []struct {
			Subject string
			From    struct{ Address string }
			To      []struct{ Address string }
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&found); err != nil {
		t.Fatal(err)
	}
	if len(found.Messages) != 1 || found.Messages[0].From.Address != "no-reply@sewakos.web.id" || found.Messages[0].To[0].Address != "penyewa@example.com" {
		t.Errorf("sandbox messages = %+v", found.Messages)
	}
}
