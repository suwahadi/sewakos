package booking

import (
	"net/http"
	"strconv"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
)

const (
	defaultPage  = 1
	defaultLimit = 20
	maxLimit     = 50
)

func pageParam(r *http.Request) int32 {
	n, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || n < 1 {
		return defaultPage
	}
	return int32(n)
}

func limitParam(r *http.Request) int32 {
	n, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || n < 1 {
		return defaultLimit
	}
	if n > maxLimit {
		return maxLimit
	}
	return int32(n)
}

func ListHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page := pageParam(r)
		limit := limitParam(r)

		rows, err := deps.Store.ListBookingsForUser(r.Context(), dbgen.ListBookingsForUserParams{
			TenantID: user.UserID, ResultLimit: limit, ResultOffset: (page - 1) * limit,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		bookings := make([]map[string]any, len(rows))
		for i, row := range rows {
			bookings[i] = map[string]any{
				"id": row.ID, "status": row.Status, "rent_type": row.RentType,
				"room":       map[string]any{"id": row.RoomID, "name": row.RoomName},
				"start_date": row.StartDate.Time.Format("2006-01-02"),
				"duration":   row.Duration,
				"end_date":   row.EndDate.Time.Format("2006-01-02"),
				"price":      row.Price, "price_label": money.FormatRupiah(row.Price),
				"expires_at": money.FormatExpiresAt(row.ExpiresAt.Time),
				"created_at": money.FormatTimestamp(row.CreatedAt.Time),
			}
		}

		response.Success(w, http.StatusOK, "", map[string]any{"bookings": bookings, "current_page": page})
	}
}
