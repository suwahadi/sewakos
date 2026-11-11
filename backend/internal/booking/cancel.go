package booking

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/finance"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/notification"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
)

const msgBookingCannotBeCancelled = "Booking tidak dapat dibatalkan."

var errBookingStateChanged = errors.New("booking state changed concurrently")

func CancelHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bookingID, ok := parseBookingID(r)
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		ctx := r.Context()

		row, err := lockBookingForCancel(ctx, deps, bookingID)
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFoundHandler(w, r)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if row.TenantID != user.UserID {
			response.Fail(w, http.StatusForbidden, response.MsgAccessDenied, nil)
			return
		}

		var result cancelResult
		switch row.Status {
		case "pending":
			result, err = cancelPendingBooking(ctx, deps, row)
		case "paid":
			result, err = cancelPaidBooking(ctx, deps, row.ID)
		default:
			response.Fail(w, http.StatusBadRequest, msgBookingCannotBeCancelled, nil)
			return
		}
		if err != nil {
			if errors.Is(err, errBookingStateChanged) {
				response.Fail(w, http.StatusBadRequest, msgBookingCannotBeCancelled, nil)
				return
			}
			response.InternalErrorHandler(w, r)
			return
		}

		if !result.hasRefund {
			response.Success(w, http.StatusOK, "Booking dibatalkan.", map[string]any{
				"booking": map[string]any{"id": row.ID, "status": "cancelled"},
				"refund":  nil,
			})
			return
		}
		response.Success(w, http.StatusOK, "Booking dibatalkan. Dana akan dikembalikan setelah dikurangi biaya admin.", map[string]any{
			"booking": map[string]any{"id": row.ID, "status": "cancelled"},
			"refund": map[string]any{
				"amount": result.refundAmount, "amount_label": money.FormatRupiah(result.refundAmount),
				"status": "awaiting_account", "completed_at": nil, "id": result.refundID,
			},
		})
	}
}

func lockBookingForCancel(ctx context.Context, deps Deps, bookingID int64) (dbgen.SelectBookingForCancelRow, error) {
	tx, err := deps.DB.Begin(ctx)
	if err != nil {
		return dbgen.SelectBookingForCancelRow{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	return dbgen.New(tx).SelectBookingForCancel(ctx, bookingID)
}

func lockPendingPayment(ctx context.Context, deps Deps, invoiceID int64) (dbgen.GetPendingPaymentOrderIDRow, error) {
	tx, err := deps.DB.Begin(ctx)
	if err != nil {
		return dbgen.GetPendingPaymentOrderIDRow{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	return dbgen.New(tx).GetPendingPaymentOrderID(ctx, invoiceID)
}

type cancelResult struct {
	hasRefund    bool
	refundID     int64
	refundAmount int64
}

// cancelPendingBooking's Midtrans calls run against a stale pre-read of the
// booking, same as checkout's Midtrans call; the final write below re-locks
// and re-checks the booking's status fresh, inside the transaction that
// mutates it, so a concurrent change (e.g. a webhook settling the payment
// first) is caught instead of silently overwritten.
func cancelPendingBooking(ctx context.Context, deps Deps, preRead dbgen.SelectBookingForCancelRow) (cancelResult, error) {
	pending, err := lockPendingPayment(ctx, deps, preRead.InvoiceID)
	if err == nil {
		status, statusErr := deps.Midtrans.GetStatus(ctx, pending.OrderID)
		if statusErr == nil && status.TransactionStatus == "settlement" {
			if err := ApplyPaymentStatus(ctx, deps, pending.OrderID, status, nil); err != nil {
				return cancelResult{}, err
			}
			return cancelPaidBooking(ctx, deps, preRead.ID)
		}
		_ = deps.Midtrans.ExpireTransaction(ctx, pending.OrderID)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return cancelResult{}, err
	}

	tx, err := deps.DB.Begin(ctx)
	if err != nil {
		return cancelResult{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	queries := dbgen.New(tx)

	row, err := queries.SelectBookingForCancel(ctx, preRead.ID)
	if err != nil {
		return cancelResult{}, err
	}
	if row.Status != "pending" {
		return cancelResult{}, errBookingStateChanged
	}

	if pending.ID != 0 {
		if err := queries.MarkPaymentExpired(ctx, pending.ID); err != nil {
			return cancelResult{}, err
		}
	}
	if err := queries.CancelBooking(ctx, row.ID); err != nil {
		return cancelResult{}, err
	}
	if row.StockHeld {
		if err := queries.ReleaseRoomLock(ctx, row.RoomID); err != nil {
			return cancelResult{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return cancelResult{}, err
	}
	committed = true
	return cancelResult{}, nil
}

func cancelPaidBooking(ctx context.Context, deps Deps, bookingID int64) (cancelResult, error) {
	tx, err := deps.DB.Begin(ctx)
	if err != nil {
		return cancelResult{}, err
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
		return cancelResult{}, err
	}
	if row.Status != "paid" {
		return cancelResult{}, errBookingStateChanged
	}

	if err := queries.CancelBooking(ctx, row.ID); err != nil {
		return cancelResult{}, err
	}
	if row.StockHeld {
		if err := queries.ReleaseRoomLock(ctx, row.RoomID); err != nil {
			return cancelResult{}, err
		}
	}

	refundAmount := row.InvoiceTotal - row.InvoiceAdminFee
	refundID, err := queries.InsertRefund(ctx, dbgen.InsertRefundParams{
		BookingID: row.ID, InvoiceID: row.InvoiceID, Amount: refundAmount, Reason: "tenant_cancelled",
	})
	if err != nil {
		return cancelResult{}, err
	}
	if _, err := finance.Record(ctx, queries, finance.Entry{
		Type: finance.TypeRefundDue, Amount: refundAmount, TenantID: &row.TenantID,
		BookingID: &row.ID, InvoiceID: &row.InvoiceID, RefundID: &refundID,
		Description: "Refund due for cancelled booking",
	}); err != nil {
		return cancelResult{}, err
	}

	tenant, err := queries.GetUserByID(ctx, row.TenantID)
	if err != nil {
		return cancelResult{}, err
	}
	owner, err := queries.GetUserByID(ctx, row.OwnerID)
	if err != nil {
		return cancelResult{}, err
	}
	link := fmt.Sprintf("/owner/booking/%d", row.ID)
	if _, err := notification.Create(ctx, queries, deps.AppPublicURL, notification.Params{
		UserID: row.OwnerID, RecipientEmail: textFieldOrNil(owner.Email), Category: notification.CategoryBooking,
		Title: "Booking dibatalkan penyewa",
		Body:  tenant.Name + " membatalkan booking " + row.RoomName + " di " + row.PropertyName + ". Kamar kembali tersedia.",
		Link:  &link,
	}); err != nil {
		return cancelResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return cancelResult{}, err
	}
	committed = true
	return cancelResult{hasRefund: true, refundID: refundID, refundAmount: refundAmount}, nil
}
