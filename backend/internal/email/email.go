// Package email defines the outbound mail interface (SPEC-GO.md §3, §17) and
// the dev/log-only implementation. A real SMTP sender is a later task
// (B9.3); nothing here depends on that yet.
package email

import (
	"context"
	"log/slog"
)

// Sender delivers one email; implementations must be safe to call from the
// worker's send_emails job (B4.8), including concurrently.
type Sender interface {
	Send(ctx context.Context, to, subject, body string) error
}

// LogSender writes the message to a structured logger instead of sending it,
// used whenever SMTP isn't configured (SPEC-GO.md §3: "Kosong = LogSender").
type LogSender struct {
	Logger *slog.Logger
}

func (s LogSender) Send(_ context.Context, to, subject, body string) error {
	s.Logger.Info("email", "to", to, "subject", subject, "body", body)
	return nil
}
