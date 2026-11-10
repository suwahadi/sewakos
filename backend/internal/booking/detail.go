package booking

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
)

func parseBookingID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "booking"), 10, 64)
	return id, err == nil && id > 0
}

func nullableTimestamp(t pgtype.Timestamptz) any {
	if !t.Valid {
		return nil
	}
	return money.FormatExpiresAt(t.Time)
}

func nullableTextField(t pgtype.Text) any {
	if !t.Valid {
		return nil
	}
	return t.String
}

func DetailHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bookingID, ok := parseBookingID(r)
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		ctx := r.Context()

		b, err := deps.Store.GetBookingByID(ctx, dbgen.GetBookingByIDParams{
			ID: bookingID, Today: pgtype.Date{Time: money.TodayWIB(), Valid: true},
		})
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFoundHandler(w, r)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if b.TenantID != user.UserID {
			response.Fail(w, http.StatusForbidden, response.MsgAccessDenied, nil)
			return
		}

		invoice, err := deps.Store.GetInitialInvoiceForBooking(ctx, bookingID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		var paymentBody any
		payment, err := deps.Store.GetLatestPaymentForInvoice(ctx, invoice.ID)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			paymentBody = nil
		case err != nil:
			response.InternalErrorHandler(w, r)
			return
		default:
			paymentBody = map[string]any{
				"order_id": payment.OrderID, "status": payment.Status,
				"snap_token":   nullableTextField(payment.SnapToken),
				"redirect_url": nullableTextField(payment.RedirectUrl),
				"expires_at":   money.FormatExpiresAt(payment.ExpiresAt.Time),
			}
		}

		var refundBody any
		refund, err := deps.Store.GetRefundForInvoice(ctx, invoice.ID)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			refundBody = nil
		case err != nil:
			response.InternalErrorHandler(w, r)
			return
		default:
			refundBody = map[string]any{
				"amount": refund.Amount, "amount_label": money.FormatRupiah(refund.Amount),
				"status": refund.Status, "completed_at": nullableTimestamp(refund.CompletedAt),
			}
		}

		response.Success(w, http.StatusOK, "", map[string]any{
			"booking": map[string]any{
				"id": b.ID, "status": b.Status, "rent_type": b.RentType,
				"room":       map[string]any{"id": b.RoomID, "name": b.RoomName},
				"property":   map[string]any{"id": b.PropertyID, "name": b.PropertyName},
				"start_date": b.StartDate.Time.Format("2006-01-02"),
				"duration":   b.Duration,
				"end_date":   b.EndDate.Time.Format("2006-01-02"),
				"price":      b.Price, "price_label": money.FormatRupiah(b.Price),
				"deposit": b.Deposit, "deposit_label": money.FormatRupiah(b.Deposit),
				"admin_fee": b.AdminFee, "admin_fee_label": money.FormatRupiah(b.AdminFee),
				"total_payment": invoice.Total, "total_payment_label": money.FormatRupiah(invoice.Total),
				"expires_at":          money.FormatExpiresAt(b.ExpiresAt.Time),
				"paid_at":             nullableTimestamp(b.PaidAt),
				"owner_deadline_at":   nullableTimestamp(b.OwnerDeadlineAt),
				"reject_reason":       nullableTextField(b.RejectReason),
				"deposit_returned_at": nullableTimestamp(b.DepositReturnedAt),
				"can_extend":          b.CanExtend, "can_review": b.CanReview,
				"created_at": money.FormatTimestamp(b.CreatedAt.Time),
			},
			"payment": paymentBody,
			"refund":  refundBody,
		})
	}
}
