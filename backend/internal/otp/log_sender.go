package otp

import (
	"context"
	"log/slog"
)

type LogSender struct {
	Logger *slog.Logger
}

func (s LogSender) Send(_ context.Context, destination, code string) error {
	s.Logger.Info("otp", "channel", "whatsapp", "destination", destination, "message", "Kode verifikasi Sewa Kos kamu: "+code)
	return nil
}
