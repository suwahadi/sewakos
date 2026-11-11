package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/booking"
	"github.com/suwahadi/sewakos/backend/internal/midtrans"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
)

const reconcilePaymentsLockKey = int64(7303)

func ReconcilePayments(ctx context.Context, deps Deps, logger *slog.Logger, now time.Time) {
	var payments []dbgen.SelectPaymentsForReconciliationRow
	if err := inTx(ctx, deps, func(q *dbgen.Queries) (err error) {
		payments, err = q.SelectPaymentsForReconciliation(ctx, pgtype.Timestamptz{Time: now.Add(-3 * 24 * time.Hour), Valid: true})
		return err
	}); err != nil {
		logger.Error("reconcile_payments: select", "error", err)
		return
	}
	for _, payment := range payments {
		reconcileOnePayment(ctx, deps, logger, payment)
	}
}

func expectedLocalStatus(status midtrans.StatusResponse) string {
	switch {
	case status.TransactionStatus == "settlement" || (status.TransactionStatus == "capture" && status.FraudStatus == "accept"):
		return "settlement"
	case status.TransactionStatus == "expire" || status.TransactionStatus == "cancel" || status.TransactionStatus == "deny":
		return status.TransactionStatus
	default:
		return "pending"
	}
}

func reconcileOnePayment(ctx context.Context, deps Deps, logger *slog.Logger, payment dbgen.SelectPaymentsForReconciliationRow) {
	status, err := deps.Booking.Midtrans.GetStatus(ctx, payment.OrderID)
	if err != nil {
		if midtrans.IsNotFound(err) {
			recordReconciliationIssue(ctx, deps, logger, payment, "not_found_at_midtrans", pgtype.Text{}, pgtype.Int8{})
			return
		}
		logger.Error("reconcile_payments: get status", "order_id", payment.OrderID, "error", err)
		return
	}

	expected := expectedLocalStatus(status)
	midtransStatus := pgtype.Text{String: status.TransactionStatus, Valid: true}
	midtransAmount, amountKnown := booking.ParseMidtransGrossAmount(status.GrossAmount)
	var midtransAmountArg pgtype.Int8
	if amountKnown {
		midtransAmountArg = pgtype.Int8{Int64: midtransAmount, Valid: true}
	}

	if expected != payment.Status {
		recordReconciliationIssue(ctx, deps, logger, payment, "status_mismatch", midtransStatus, midtransAmountArg)
		if expected == "settlement" {
			if err := booking.ApplyPaymentStatus(ctx, deps.Booking, payment.OrderID, status, nil); err != nil {
				logger.Error("reconcile_payments: apply status", "order_id", payment.OrderID, "error", err)
			}
		}
	}
	if amountKnown && midtransAmount != payment.GrossAmount {
		recordReconciliationIssue(ctx, deps, logger, payment, "amount_mismatch", midtransStatus, midtransAmountArg)
	}
}

func recordReconciliationIssue(ctx context.Context, deps Deps, logger *slog.Logger, payment dbgen.SelectPaymentsForReconciliationRow, kind string, midtransStatus pgtype.Text, midtransAmount pgtype.Int8) {
	if err := inTx(ctx, deps, func(q *dbgen.Queries) error {
		return q.InsertReconciliationIssue(ctx, dbgen.InsertReconciliationIssueParams{
			PaymentID: payment.ID, Kind: kind, LocalStatus: payment.Status,
			MidtransStatus: midtransStatus, LocalAmount: payment.GrossAmount, MidtransAmount: midtransAmount,
		})
	}); err != nil {
		logger.Error("reconcile_payments: insert issue", "order_id", payment.OrderID, "kind", kind, "error", err)
	}
}
