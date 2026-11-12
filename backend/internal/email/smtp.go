package email

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/wneessen/go-mail"
)

const implicitTLSPort = 465

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type SMTPSender struct {
	client *mail.Client
	from   string
}

func NewSMTPSender(cfg SMTPConfig) (*SMTPSender, error) {
	opts := []mail.Option{mail.WithPort(cfg.Port), mail.WithTimeout(30 * time.Second)}
	if cfg.Port == implicitTLSPort {
		opts = append(opts, mail.WithSSL())
	}
	// Credentials are only ever sent over TLS; without them (a local sandbox) STARTTLS is used when offered.
	if cfg.Username != "" {
		opts = append(opts, mail.WithTLSPolicy(mail.TLSMandatory), mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
			mail.WithUsername(cfg.Username), mail.WithPassword(cfg.Password))
	} else {
		opts = append(opts, mail.WithTLSPolicy(mail.TLSOpportunistic))
	}
	client, err := mail.NewClient(cfg.Host, opts...)
	if err != nil {
		return nil, fmt.Errorf("create smtp client for %s:%d: %w", cfg.Host, cfg.Port, err)
	}
	return &SMTPSender{client: client, from: cfg.From}, nil
}

func (s *SMTPSender) Send(ctx context.Context, to, subject, body string) error {
	msg := mail.NewMsg()
	if err := msg.From(s.from); err != nil {
		return fmt.Errorf("set from %q: %w", s.from, err)
	}
	if err := msg.To(to); err != nil {
		return fmt.Errorf("set recipient: %w", err)
	}
	msg.Subject(subject)
	msg.SetBodyString(mail.TypeTextPlain, body)
	if err := s.client.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	return nil
}

func NewSender(cfg SMTPConfig, logger *slog.Logger) (Sender, error) {
	if cfg.Host == "" {
		return LogSender{Logger: logger}, nil
	}
	if cfg.Username != "" && cfg.Password == "" {
		logger.Warn("SMTP_USERNAME is set without SMTP_PASSWORD; emails are logged instead of sent", "host", cfg.Host)
		return LogSender{Logger: logger}, nil
	}
	if cfg.From == "" {
		return nil, fmt.Errorf("EMAIL_FROM is required when SMTP_HOST is set")
	}
	if cfg.Port == 0 {
		cfg.Port = 587
	}
	return NewSMTPSender(cfg)
}
