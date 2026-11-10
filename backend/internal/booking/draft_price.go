package booking

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/response"
)

func parseRoomID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "room"), 10, 64)
	return id, err == nil && id > 0
}

type draftPriceOption struct {
	sort  int
	label string
	typ   string
	price int64
	has   bool
}

func clamp(n, min, max int) int {
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

func DraftPriceHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomID, ok := parseRoomID(r)
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		ctx := r.Context()

		room, err := deps.Store.GetRoomForDraftPrice(ctx, roomID)
		if err != nil {
			response.NotFoundHandler(w, r)
			return
		}

		adminFee, err := deps.AdminFee.Get(ctx)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		discount := int64(0)
		if room.IsFlashSale && r.URL.Query().Get("is_flash_sale") == "true" {
			discount = room.FlashSaleDiscount
		}

		duration := 1
		if raw := r.URL.Query().Get("duration"); raw != "" {
			if n, err := strconv.Atoi(raw); err == nil {
				duration = n
			}
		}
		dailyDuration := clamp(duration, 1, 30)
		weeklyDuration := clamp(duration, 1, 12)

		options := []draftPriceOption{
			{1, "Per Bulan", "monthly", room.PriceMonthly, true},
			{2, "Per Minggu", "weekly", room.PriceWeekly.Int64, room.PriceWeekly.Valid},
			{3, "Per Hari", "daily", room.PriceDaily.Int64, room.PriceDaily.Valid},
		}

		data := make([]map[string]any, 0, len(options))
		for _, o := range options {
			if !o.has {
				continue
			}
			periods := 1
			switch o.typ {
			case "weekly":
				periods = weeklyDuration
			case "daily":
				periods = dailyDuration
			}

			rentForPeriods := o.price * int64(periods)
			deposit := o.price
			totalOffFirstPayment := rentForPeriods + deposit + adminFee - discount
			remainingPayment := rentForPeriods + deposit + adminFee

			data = append(data, map[string]any{
				"sort": o.sort, "label": o.label, "type": o.typ,
				"price": o.price, "price_label": money.FormatRupiah(o.price),
				"deposit": deposit, "deposit_label": money.FormatRupiah(deposit),
				"admin_fee": adminFee, "admin_fee_label": money.FormatRupiah(adminFee),
				"flash_sale_discount": discount, "flash_sale_discount_label": money.FormatRupiah(discount),
				"total_off_first_payment":       totalOffFirstPayment,
				"total_off_first_payment_label": money.FormatRupiah(totalOffFirstPayment),
				"remaining_payment":             remainingPayment,
				"remaining_payment_label":       money.FormatRupiah(remainingPayment),
			})
		}

		response.Success(w, http.StatusOK, "", map[string]any{"data": data})
	}
}
