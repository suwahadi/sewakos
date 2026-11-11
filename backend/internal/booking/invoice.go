package booking

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
)

func parseInvoiceID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "invoice"), 10, 64)
	return id, err == nil && id > 0
}

func InvoiceNumber(issuedAt time.Time, id int64) string {
	wib := issuedAt.In(money.WIB)
	return fmt.Sprintf("INV/%04d/%02d/%06d", wib.Year(), wib.Month(), id)
}

func InvoiceBody(id int64, number string, bookingID int64, invoiceType, status string, periodStart, periodEnd pgtype.Date,
	rentAmount, discount, deposit, adminFee, total int64, dueDate pgtype.Date, paidAt pgtype.Timestamptz, issuedAt pgtype.Timestamptz,
) map[string]any {
	return map[string]any{
		"id": id, "number": number, "booking_id": bookingID, "type": invoiceType, "status": status,
		"period_start": periodStart.Time.Format("2006-01-02"), "period_end": periodEnd.Time.Format("2006-01-02"),
		"rent_amount": rentAmount, "rent_amount_label": money.FormatRupiah(rentAmount),
		"discount": discount, "discount_label": money.FormatRupiah(discount),
		"deposit": deposit, "deposit_label": money.FormatRupiah(deposit),
		"admin_fee": adminFee, "admin_fee_label": money.FormatRupiah(adminFee),
		"total": total, "total_label": money.FormatRupiah(total),
		"due_date": dueDate.Time.Format("2006-01-02"), "paid_at": nullableTimestamp(paidAt),
		"issued_at": money.FormatExpiresAt(issuedAt.Time),
	}
}

func InvoiceListHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page := pageParam(r)
		limit := limitParam(r)

		var bookingIDFilter pgtype.Int8
		if v, ok := toInt64(parseQueryNumber(r, "booking_id")); ok {
			bookingIDFilter = pgtype.Int8{Int64: v, Valid: true}
		}
		var statusFilter pgtype.Text
		if v := r.URL.Query().Get("status"); v != "" {
			statusFilter = pgtype.Text{String: v, Valid: true}
		}

		rows, err := deps.Store.ListInvoicesForTenant(r.Context(), dbgen.ListInvoicesForTenantParams{
			TenantID: user.UserID, BookingID: bookingIDFilter, Status: statusFilter,
			ResultLimit: limit, ResultOffset: (page - 1) * limit,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		invoices := make([]map[string]any, len(rows))
		for i, row := range rows {
			invoices[i] = InvoiceBody(row.ID, row.Number, row.BookingID, row.Type, row.Status,
				row.PeriodStart, row.PeriodEnd, row.RentAmount, row.Discount, row.Deposit, row.AdminFee, row.Total,
				row.DueDate, row.PaidAt, row.IssuedAt)
		}

		response.Success(w, http.StatusOK, "", map[string]any{"invoices": invoices, "current_page": page})
	}
}

func parseQueryNumber(r *http.Request, key string) any {
	v := r.URL.Query().Get(key)
	if v == "" {
		return nil
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return nil
	}
	return n
}

func InvoiceDetailHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		invoiceID, ok := parseInvoiceID(r)
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		ctx := r.Context()

		invoice, err := deps.Store.GetInvoiceForTenant(ctx, invoiceID)
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFoundHandler(w, r)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if invoice.TenantID != user.UserID {
			response.Fail(w, http.StatusForbidden, response.MsgAccessDenied, nil)
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
			"invoice": InvoiceBody(invoice.ID, invoice.Number, invoice.BookingID, invoice.Type, invoice.Status,
				invoice.PeriodStart, invoice.PeriodEnd, invoice.RentAmount, invoice.Discount, invoice.Deposit, invoice.AdminFee,
				invoice.Total, invoice.DueDate, invoice.PaidAt, invoice.IssuedAt),
			"payment": paymentBody,
			"refund":  refundBody,
		})
	}
}
