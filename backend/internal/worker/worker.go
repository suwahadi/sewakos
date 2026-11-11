package worker

import (
	"context"
	"log/slog"

	"github.com/suwahadi/sewakos/backend/internal/booking"
	"github.com/suwahadi/sewakos/backend/internal/email"
	"github.com/suwahadi/sewakos/backend/internal/money"
)

type Deps struct {
	Booking booking.Deps
	Email   email.Sender
	Storage ObjectStore

	OwnerReminderBeforeMinutes int
}

func RunJobs(ctx context.Context, deps Deps, logger *slog.Logger) {
	today := money.TodayWIB()
	ExpirePayments(ctx, deps, logger)
	ExpireBookings(ctx, deps, logger)
	OwnerTimeout(ctx, deps, logger)
	OwnerReminder(ctx, deps, logger)
	IssueRenewals(ctx, deps, logger, today)
	RemindRenewalsDue(ctx, deps, logger, today)
	TerminateOverdue(ctx, deps, logger, today)
	CompleteTenancies(ctx, deps, logger, today)
	PurgeStorage(ctx, deps, logger)
	SendEmails(ctx, deps, logger)
}
