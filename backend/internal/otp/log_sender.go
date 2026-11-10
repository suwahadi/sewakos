package otp

import (
	"context"
	"log/slog"
)

// LogSender writes the OTP message to a structured logger instead of
// delivering it, used until a real SMS/WhatsApp provider is chosen
// (PRD.md §8 #1; SPEC-GO.md §3: "Pengiriman OTP hanya log").
type LogSender struct {
	Logger *slog.Logger
}

func (s LogSender) Send(_ context.Context, channel Channel, destination, message string) error {
	s.Logger.Info("otp", "channel", channel, "destination", destination, "message", message)
	return nil
}
