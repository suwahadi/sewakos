package booking

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
)

const msgBookingCannotBeExtended = "Sewa tidak dapat diperpanjang."

func ExtendHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bookingID, ok := parseBookingID(r)
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		ctx := r.Context()

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		tx, err := deps.DB.Begin(ctx)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback(ctx)
			}
		}()
		queries := dbgen.New(tx)

		row, err := queries.LockBookingForExtend(ctx, bookingID)
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

		var monthlyOptions []int
		_ = json.Unmarshal(row.MonthlyDurationOptions, &monthlyOptions)
		if len(monthlyOptions) == 0 {
			monthlyOptions = []int{1}
		}
		duration, durationOk := durationForExtend(body["duration"], row.RentType, monthlyOptions)
		if !durationOk {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, map[string][]string{
				"duration": {"The selected duration is invalid."},
			})
			return
		}

		today := money.TodayWIB()
		if row.Status != "approved" || !row.EndDate.Time.After(today) {
			response.Fail(w, http.StatusBadRequest, msgBookingCannotBeExtended, nil)
			return
		}
		unpaid, err := queries.ExistsUnpaidExtensionInvoice(ctx, bookingID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if unpaid {
			response.Fail(w, http.StatusBadRequest, "Masih ada perpanjangan yang belum dibayar.", nil)
			return
		}

		fee, err := deps.AdminFee.Get(ctx)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		periodStart := row.EndDate.Time
		periodEnd := money.AddPeriod(row.StartDate.Time, row.RentType, int(row.Duration)+duration)
		dueDate := periodStart.AddDate(0, 0, -1)
		var rentAmount int64
		if row.RentType == "monthly" {
			rentAmount = row.PeriodPrice
		} else {
			rentAmount = row.PeriodPrice * int64(duration)
		}
		total := rentAmount + fee

		issuedAt := time.Now()
		invoiceID, err := queries.InsertExtensionInvoice(ctx, dbgen.InsertExtensionInvoiceParams{
			BookingID: bookingID, PeriodStart: toPgDate(periodStart), PeriodEnd: toPgDate(periodEnd),
			Periods: int16(duration), RentAmount: rentAmount, AdminFee: fee, Total: total,
			DueDate: toPgDate(dueDate), IssuedAt: toPgTimestamptz(issuedAt),
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		number := InvoiceNumber(issuedAt, invoiceID)
		if err := queries.SetInvoiceNumber(ctx, dbgen.SetInvoiceNumberParams{ID: invoiceID, Number: number}); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		committed = true

		response.Success(w, http.StatusOK, "Tagihan perpanjangan dibuat. Silakan lakukan pembayaran.", map[string]any{
			"invoice": InvoiceBody(invoiceID, number, bookingID, "extension", "unpaid",
				toPgDate(periodStart), toPgDate(periodEnd), rentAmount, 0, 0, fee, total,
				toPgDate(dueDate), pgtype.Timestamptz{}, toPgTimestamptz(issuedAt)),
		})
	}
}

func durationForExtend(raw any, rentType string, monthlyOptions []int) (int, bool) {
	switch rentType {
	case "monthly":
		n, ok := toInt64(raw)
		if !ok {
			return 0, false
		}
		for _, opt := range monthlyOptions {
			if int64(opt) == n {
				return int(n), true
			}
		}
		return 0, false
	case "daily":
		return clampedDuration(raw, 1, 30)
	case "weekly":
		return clampedDuration(raw, 1, 12)
	default:
		return 0, false
	}
}

func clampedDuration(raw any, min, max int) (int, bool) {
	n, ok := toInt64(raw)
	if !ok || n < int64(min) || n > int64(max) {
		return 0, false
	}
	return int(n), true
}
