package worker

import (
	"context"
	"log/slog"

	"github.com/suwahadi/sewakos/backend/internal/booking"
	"github.com/suwahadi/sewakos/backend/internal/midtrans"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

func ExpirePayments(ctx context.Context, deps Deps, logger *slog.Logger) {
	orderIDs, err := selectExpiredPendingPaymentOrderIDs(ctx, deps)
	if err != nil {
		logger.Error("expire_payments: select", "error", err)
		return
	}
	for _, orderID := range orderIDs {
		if err := expireOnePayment(ctx, deps, orderID); err != nil {
			logger.Error("expire_payments: process", "order_id", orderID, "error", err)
		}
	}
}

func selectExpiredPendingPaymentOrderIDs(ctx context.Context, deps Deps) ([]string, error) {
	tx, err := deps.Booking.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	return dbgen.New(tx).SelectExpiredPendingPaymentOrderIDs(ctx)
}

func expireOnePayment(ctx context.Context, deps Deps, orderID string) error {
	status, err := deps.Booking.Midtrans.GetStatus(ctx, orderID)
	if err != nil {
		return err
	}
	if status.TransactionStatus != "pending" {
		return booking.ApplyPaymentStatus(ctx, deps.Booking, orderID, status, nil)
	}

	_ = deps.Booking.Midtrans.ExpireTransaction(ctx, orderID)
	return booking.ApplyPaymentStatus(ctx, deps.Booking, orderID, midtrans.StatusResponse{
		OrderID: orderID, TransactionStatus: "expire",
	}, nil)
}
