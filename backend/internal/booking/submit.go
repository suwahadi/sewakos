package booking

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/validation"
)

func toInt64(v any) (int64, bool) {
	n, ok := v.(float64)
	if !ok || n != float64(int64(n)) {
		return 0, false
	}
	return int64(n), true
}

func toPgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}

func toPgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func minInt(nums []int) int {
	m := nums[0]
	for _, n := range nums[1:] {
		if n < m {
			m = n
		}
	}
	return m
}

func SubmitHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		ctx := r.Context()

		roomID, roomIDOk := toInt64(body["room_id"])
		var room dbgen.GetRoomForSubmitRow
		var roomFound bool
		if roomIDOk {
			var err error
			room, err = deps.Store.GetRoomForSubmit(ctx, roomID)
			roomFound = err == nil
		}

		v := validation.New()
		v.Field("room_id", body["room_id"]).Required().Integer().Exists(roomFound)
		v.Field("rent_type", body["rent_type"]).Required().In("daily", "weekly", "monthly")
		_, startDate, startDateOk := v.Field("start_date", body["start_date"]).Required().Date()
		if startDateOk {
			v.Field("start_date", body["start_date"]).AfterOrEqualToday(startDate)
		}
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}

		rentType, _ := body["rent_type"].(string)

		var monthlyOptions []int
		_ = json.Unmarshal(room.MonthlyDurationOptions, &monthlyOptions)
		if len(monthlyOptions) == 0 {
			monthlyOptions = []int{1}
		}

		duration, durationOk := durationForSubmit(body["duration"], rentType, monthlyOptions)
		if !durationOk {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, map[string][]string{
				"duration": {"The selected duration is invalid."},
			})
			return
		}

		listed := room.IsActive && room.ModerationStatus == "approved" && !room.OwnerSuspended
		if !listed {
			response.Fail(w, http.StatusBadRequest, "Properti tidak aktif.", nil)
			return
		}
		if room.Available < 1 {
			response.Fail(w, http.StatusBadRequest, "Kamar tidak tersedia.", nil)
			return
		}

		periodPrice, priceOk := periodPriceForRentType(room, rentType)
		if !priceOk {
			response.Fail(w, http.StatusBadRequest, "Tipe sewa tidak tersedia.", nil)
			return
		}

		discount := int64(0)
		if room.IsFlashSale {
			discount = room.FlashSaleDiscount
		}
		price := periodPrice - discount
		if price < 0 {
			price = 0
		}

		adminFee, err := deps.AdminFee.Get(ctx)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		endDate := money.AddPeriod(startDate, rentType, duration)
		expiresAt := time.Now().Add(24 * time.Hour)

		var periods int
		var rentAmount int64
		var invoicePeriodEnd time.Time
		if rentType == "monthly" {
			periods = 1
			rentAmount = periodPrice
			invoicePeriodEnd = money.AddPeriod(startDate, rentType, 1)
		} else {
			periods = duration
			rentAmount = periodPrice * int64(duration)
			invoicePeriodEnd = endDate
		}
		invoiceTotal := rentAmount - discount + periodPrice + adminFee
		dueDate := time.Date(
			expiresAt.In(money.WIB).Year(), expiresAt.In(money.WIB).Month(), expiresAt.In(money.WIB).Day(),
			0, 0, 0, 0, money.WIB,
		)

		tx, err := deps.DB.Begin(ctx)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()
		queries := dbgen.New(tx)

		inserted, err := queries.InsertBooking(ctx, dbgen.InsertBookingParams{
			TenantID: user.UserID, RoomID: roomID, RentType: rentType,
			StartDate: toPgDate(startDate), Duration: int16(duration), EndDate: toPgDate(endDate),
			PeriodPrice: periodPrice, FlashSaleDiscount: discount, Price: price,
			Deposit: periodPrice, AdminFee: adminFee, ExpiresAt: toPgTimestamptz(expiresAt),
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		issuedAt := time.Now()
		invoiceID, err := queries.InsertInvoice(ctx, dbgen.InsertInvoiceParams{
			BookingID: inserted.ID, Type: "initial",
			PeriodStart: toPgDate(startDate), PeriodEnd: toPgDate(invoicePeriodEnd), Periods: int16(periods),
			RentAmount: rentAmount, Discount: discount, Deposit: periodPrice, AdminFee: adminFee,
			Total: invoiceTotal, DueDate: toPgDate(dueDate), IssuedAt: toPgTimestamptz(issuedAt),
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

		response.Success(w, http.StatusOK, "Pengajuan sewa dibuat. Silakan lakukan pembayaran.", map[string]any{
			"booking": map[string]any{
				"id": inserted.ID, "status": "pending",
				"room":       map[string]any{"id": roomID, "name": room.Name},
				"start_date": startDate.Format("2006-01-02"),
				"expires_at": money.FormatExpiresAt(expiresAt),
			},
		})
	}
}

func periodPriceForRentType(room dbgen.GetRoomForSubmitRow, rentType string) (int64, bool) {
	switch rentType {
	case "monthly":
		return room.PriceMonthly, true
	case "weekly":
		return room.PriceWeekly.Int64, room.PriceWeekly.Valid
	case "daily":
		return room.PriceDaily.Int64, room.PriceDaily.Valid
	default:
		return 0, false
	}
}

func durationForSubmit(raw any, rentType string, monthlyOptions []int) (int, bool) {
	switch rentType {
	case "monthly":
		if raw == nil {
			return minInt(monthlyOptions), true
		}
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
		return clampedOptionalDuration(raw, 1, 30)
	case "weekly":
		return clampedOptionalDuration(raw, 1, 12)
	default:
		return 0, false
	}
}

func clampedOptionalDuration(raw any, min, max int) (int, bool) {
	if raw == nil {
		return 1, true
	}
	n, ok := toInt64(raw)
	if !ok || n < int64(min) || n > int64(max) {
		return 0, false
	}
	return int(n), true
}
