package admin

import (
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/booking"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
)

func bookingBody(row dbgen.AdminListBookingsRow) map[string]any {
	return map[string]any{
		"id": row.ID, "status": row.Status, "rent_type": row.RentType,
		"start_date": row.StartDate.Time.Format(time.DateOnly), "duration": row.Duration,
		"end_date": row.EndDate.Time.Format(time.DateOnly),
		"tenant":   map[string]any{"id": row.TenantID, "name": row.TenantName, "phone_number": row.TenantPhone},
		"property": map[string]any{"id": row.PropertyID, "name": row.PropertyName},
		"room":     map[string]any{"id": row.RoomID, "name": row.RoomName},
		"price":    row.Price, "price_label": money.FormatRupiah(row.Price),
		"deposit": row.Deposit, "deposit_label": money.FormatRupiah(row.Deposit),
		"admin_fee": row.AdminFee, "admin_fee_label": money.FormatRupiah(row.AdminFee),
		"payout_amount": row.PayoutAmount, "payout_amount_label": money.FormatRupiah(row.PayoutAmount),
		"paid_at": nullableTimestamp(row.PaidAt), "owner_deadline_at": nullableTimestamp(row.OwnerDeadlineAt),
		"reject_reason": nullableText(row.RejectReason), "deposit_returned_at": nullableTimestamp(row.DepositReturnedAt),
		"created_at": money.FormatTimestamp(row.CreatedAt.Time),
	}
}

func ListBookingsHandler(deps Deps, _ authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, limit := pageParam(r), limitParam(r)
		rows, err := deps.Store.AdminListBookings(r.Context(), dbgen.AdminListBookingsParams{
			Status: queryText(r, "status"), RowLimit: limit, RowOffset: (page - 1) * limit,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		bookings := make([]map[string]any, len(rows))
		for i, row := range rows {
			bookings[i] = bookingBody(row)
		}
		response.Success(w, http.StatusOK, "", map[string]any{"bookings": bookings, "current_page": page})
	}
}

func GetBookingHandler(deps Deps, _ authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bookingID, ok := parseID(r, "booking")
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		ctx := r.Context()
		rows, err := deps.Store.AdminListBookings(ctx, dbgen.AdminListBookingsParams{BookingID: pgtype.Int8{Int64: bookingID, Valid: true}, RowLimit: 1})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if len(rows) == 0 {
			response.NotFoundHandler(w, r)
			return
		}
		row := rows[0]

		var payment, refund, payout any
		if p, err := deps.Store.GetLatestPaymentForInvoice(ctx, row.InvoiceID); err == nil {
			payment = map[string]any{
				"order_id": p.OrderID, "status": p.Status, "snap_token": nullableText(p.SnapToken),
				"redirect_url": nullableText(p.RedirectUrl), "expires_at": money.FormatExpiresAt(p.ExpiresAt.Time),
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			response.InternalErrorHandler(w, r)
			return
		}
		if rf, err := deps.Store.GetRefundForInvoice(ctx, row.InvoiceID); err == nil {
			refund = map[string]any{
				"amount": rf.Amount, "amount_label": money.FormatRupiah(rf.Amount), "status": rf.Status,
				"completed_at": nullableTimestamp(rf.CompletedAt),
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			response.InternalErrorHandler(w, r)
			return
		}
		if po, err := deps.Store.GetPayoutForInvoice(ctx, row.InvoiceID); err == nil {
			payout = map[string]any{
				"id": po.ID, "booking_id": po.BookingID, "amount": po.Amount, "amount_label": money.FormatRupiah(po.Amount),
				"status": po.Status, "transfer_reference": nullableText(po.TransferReference),
				"completed_at": nullableTimestamp(po.CompletedAt), "created_at": money.FormatTimestamp(po.CreatedAt.Time),
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "", map[string]any{"booking": bookingBody(row), "payment": payment, "refund": refund, "payout": payout})
	}
}

func ListInvoicesHandler(deps Deps, _ authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, limit := pageParam(r), limitParam(r)
		rows, err := deps.Store.AdminListInvoices(r.Context(), dbgen.AdminListInvoicesParams{
			Status: queryText(r, "status"), Type: queryText(r, "type"), BookingID: queryID(r, "booking_id"),
			RowLimit: limit, RowOffset: (page - 1) * limit,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		invoices := make([]map[string]any, len(rows))
		for i, row := range rows {
			invoices[i] = booking.InvoiceBody(row.ID, row.Number, row.BookingID, row.Type, row.Status,
				row.PeriodStart, row.PeriodEnd, row.RentAmount, row.Discount, row.Deposit, row.AdminFee, row.Total,
				row.DueDate, row.PaidAt, row.IssuedAt)
		}
		response.Success(w, http.StatusOK, "", map[string]any{"invoices": invoices, "current_page": page})
	}
}
