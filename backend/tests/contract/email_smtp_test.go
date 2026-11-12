package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/suwahadi/sewakos/backend/internal/booking"
	"github.com/suwahadi/sewakos/backend/internal/email"
	"github.com/suwahadi/sewakos/backend/internal/worker"
)

func TestSendEmailsDeliversThroughSMTPSandbox(t *testing.T) {
	addr := envOr("TEST_SMTP_ADDR", "localhost:1025")
	api := envOr("TEST_MAILPIT_URL", "http://localhost:8025")
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Skipf("SMTP sandbox not reachable at %s: %v", addr, err)
	}
	conn.Close()
	_, pool := newTestServerAndPool(t)
	host, portText, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(portText)
	sender, err := email.NewSMTPSender(email.SMTPConfig{Host: host, Port: port, From: "Sewa Kos <no-reply@sewakos.web.id>"})
	if err != nil {
		t.Fatal(err)
	}

	subject := fmt.Sprintf("Pembayaran berhasil %d", time.Now().UnixNano())
	var emailID int64
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO email_outbox (to_email, subject, body_text) VALUES ($1, $2, $3) RETURNING id`,
		"penyewa@example.test", subject, "Pembayaran tagihanmu sudah kami terima.",
	).Scan(&emailID); err != nil {
		t.Fatal(err)
	}

	worker.SendEmails(context.Background(), worker.Deps{Booking: booking.Deps{DB: pool}, Email: sender}, discardLogger())

	var sent bool
	if err := pool.QueryRow(context.Background(), `SELECT sent_at IS NOT NULL FROM email_outbox WHERE id = $1`, emailID).Scan(&sent); err != nil || !sent {
		t.Fatalf("outbox row sent = %v (err %v)", sent, err)
	}
	resp, err := http.Get(api + "/api/v1/search?query=" + url.QueryEscape(fmt.Sprintf("subject:%q", subject)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var found struct {
		Messages []struct {
			Subject string
			To      []struct{ Address string }
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&found); err != nil {
		t.Fatal(err)
	}
	if len(found.Messages) != 1 || found.Messages[0].To[0].Address != "penyewa@example.test" {
		t.Errorf("sandbox messages = %+v", found.Messages)
	}
}
