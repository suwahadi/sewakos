package worker

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"

	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

func ExpireBookings(ctx context.Context, deps Deps, logger *slog.Logger) {
	ids, err := selectExpiredBookingIDs(ctx, deps)
	if err != nil {
		logger.Error("expire_bookings: select", "error", err)
		return
	}
	for _, id := range ids {
		if err := expireOneBooking(ctx, deps, id); err != nil {
			logger.Error("expire_bookings: process", "booking_id", id, "error", err)
		}
	}
}

func selectExpiredBookingIDs(ctx context.Context, deps Deps) ([]int64, error) {
	tx, err := deps.Booking.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	return dbgen.New(tx).SelectExpiredBookingIDsWithoutPendingPayment(ctx)
}

func expireOneBooking(ctx context.Context, deps Deps, bookingID int64) error {
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

	row, err := queries.SelectBookingForCancel(ctx, bookingID)
	if err != nil {
		return err
	}
	if row.Status != "pending" {
		committed = true
		return tx.Commit(ctx)
	}
	_, pendingErr := queries.GetPendingPaymentOrderID(ctx, row.InvoiceID)
	if pendingErr == nil {
		committed = true
		return tx.Commit(ctx)
	}
	if !errors.Is(pendingErr, pgx.ErrNoRows) {
		return pendingErr
	}

	if err := queries.CancelBookingBySystem(ctx, bookingID); err != nil {
		return err
	}
	if row.StockHeld {
		if err := queries.ReleaseRoomLock(ctx, row.RoomID); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}
