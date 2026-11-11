package worker

import (
	"context"
	"log/slog"

	"github.com/suwahadi/sewakos/backend/internal/booking"
	"github.com/suwahadi/sewakos/backend/internal/email"
)

type Deps struct {
	Booking booking.Deps
	Email   email.Sender
	Storage ObjectStore
}

func RunJobs(ctx context.Context, deps Deps, logger *slog.Logger) {
	ExpirePayments(ctx, deps, logger)
	ExpireBookings(ctx, deps, logger)
	SendEmails(ctx, deps, logger)
	PurgeStorage(ctx, deps, logger)
}
