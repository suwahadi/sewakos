package worker

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

func SendEmails(ctx context.Context, deps Deps, logger *slog.Logger) {
	rows, err := selectDueEmailOutbox(ctx, deps)
	if err != nil {
		logger.Error("send_emails: select", "error", err)
		return
	}
	for _, row := range rows {
		if err := sendOneEmail(ctx, deps, row); err != nil {
			logger.Error("send_emails: process", "email_id", row.ID, "error", err)
		}
	}
}

func selectDueEmailOutbox(ctx context.Context, deps Deps) ([]dbgen.SelectDueEmailOutboxRow, error) {
	tx, err := deps.Booking.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	return dbgen.New(tx).SelectDueEmailOutbox(ctx)
}

func sendOneEmail(ctx context.Context, deps Deps, row dbgen.SelectDueEmailOutboxRow) error {
	sendErr := deps.Email.Send(ctx, row.ToEmail, row.Subject, row.BodyText)

	tx, err := deps.Booking.DB.Begin(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	queries := dbgen.New(tx)

	if sendErr == nil {
		if err := queries.MarkEmailSent(ctx, row.ID); err != nil {
			return err
		}
	} else {
		backoff := time.Duration(math.Pow(2, float64(row.Attempts+1))) * time.Minute
		if err := queries.MarkEmailAttemptFailed(ctx, dbgen.MarkEmailAttemptFailedParams{
			ID: row.ID, NextAttemptAt: pgtype.Timestamptz{Time: time.Now().Add(backoff), Valid: true},
			LastError: pgtype.Text{String: sendErr.Error(), Valid: true},
		}); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}
