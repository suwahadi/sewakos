package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/audit"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/notification"
	"github.com/suwahadi/sewakos/backend/internal/owner"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

func OwnerTimeout(ctx context.Context, deps Deps, logger *slog.Logger) {
	var ids []int64
	if err := inTx(ctx, deps, func(q *dbgen.Queries) (err error) {
		ids, err = q.SelectOwnerTimeoutBookingIDs(ctx)
		return err
	}); err != nil {
		logger.Error("owner_timeout: select", "error", err)
		return
	}
	for _, id := range ids {
		if err := inTx(ctx, deps, func(q *dbgen.Queries) error { return timeoutBooking(ctx, deps, q, id) }); err != nil {
			logger.Error("owner_timeout: process", "booking_id", id, "error", err)
		}
	}
}

func timeoutBooking(ctx context.Context, deps Deps, q *dbgen.Queries, id int64) error {
	booking, err := q.LockPaidBooking(ctx, dbgen.LockPaidBookingParams{ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if booking.Status != "paid" || time.Now().Before(booking.OwnerDeadlineAt.Time) {
		return nil
	}
	if err := owner.RejectPaidBooking(ctx, q, deps.Booking.AppPublicURL, booking, owner.Rejection{
		RejectedBy: "system", Reason: pgtype.Text{String: "Pemilik tidak merespons dalam 24 jam.", Valid: true},
		RefundReason: "owner_timeout", TenantTitle: "Booking ditolak otomatis",
		TenantBody: "Pemilik tidak merespons dalam 24 jam. Dana akan dikembalikan penuh. Silakan isi rekening pengembalian dana.",
	}); err != nil {
		return err
	}
	_, err = audit.Record(ctx, q, audit.Params{
		Action: "booking.owner_timeout", EntityType: "booking", EntityID: booking.ID, OwnerID: &booking.OwnerID,
		Before: map[string]any{"status": "paid"}, After: map[string]any{"status": "rejected", "rejected_by": "system"},
	})
	return err
}

func OwnerReminder(ctx context.Context, deps Deps, logger *slog.Logger) {
	var ids []int64
	if err := inTx(ctx, deps, func(q *dbgen.Queries) (err error) {
		ids, err = q.SelectOwnerReminderBookingIDs(ctx, int32(deps.OwnerReminderBeforeMinutes))
		return err
	}); err != nil {
		logger.Error("owner_reminder: select", "error", err)
		return
	}
	for _, id := range ids {
		if err := inTx(ctx, deps, func(q *dbgen.Queries) error { return remindOwner(ctx, deps, q, id) }); err != nil {
			logger.Error("owner_reminder: process", "booking_id", id, "error", err)
		}
	}
}

func remindOwner(ctx context.Context, deps Deps, q *dbgen.Queries, id int64) error {
	booking, err := q.MarkOwnerReminderSent(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return owner.NotifyUser(ctx, q, deps.Booking.AppPublicURL, booking.OwnerID, notification.CategoryBooking, "Segera respons booking",
		fmt.Sprintf("Booking #%d akan otomatis ditolak pada %s jika tidak direspons.", booking.ID, money.FormatHumanWIB(booking.OwnerDeadlineAt.Time)),
		fmt.Sprintf("/owner/booking/%d", booking.ID))
}
