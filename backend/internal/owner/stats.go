package owner

import (
	"math"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/finance"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/response"
)

func occupancyRate(occupied, total int64) float64 {
	if total == 0 {
		return 0
	}
	return math.Round(float64(occupied)/float64(total)*100) / 100
}

func StatsHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from, toExclusive, errs := finance.ParseDateRange(r)
		if errs != nil {
			respondInvalidDateRange(w, r, errs)
			return
		}
		propertyID := optionalQueryID(r, "property_id")
		ctx := r.Context()

		views, err := deps.Store.GetOwnerStatsViews(ctx, dbgen.GetOwnerStatsViewsParams{
			OwnerID: user.UserID, PropertyID: propertyID, FromDate: toPgDate(from), ToDate: toPgDate(toExclusive.AddDate(0, 0, -1)),
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		counts, err := deps.Store.GetOwnerStatsBookingCounts(ctx, dbgen.GetOwnerStatsBookingCountsParams{
			FromTime: toPgTimestamptz(from), ToTime: toPgTimestamptz(toExclusive), OwnerID: user.UserID, PropertyID: propertyID,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		earnings, err := deps.Store.GetOwnerStatsEarnings(ctx, dbgen.GetOwnerStatsEarningsParams{
			OwnerID: pgtype.Int8{Int64: user.UserID, Valid: true}, FromTime: toPgTimestamptz(from), ToTime: toPgTimestamptz(toExclusive),
			PropertyID: propertyID,
		})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		occupancy, err := deps.Store.GetOwnerOccupancy(ctx, dbgen.GetOwnerOccupancyParams{OwnerID: user.UserID, PropertyID: propertyID})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		rating, err := deps.Store.GetOwnerRating(ctx, dbgen.GetOwnerRatingParams{OwnerID: user.UserID, PropertyID: propertyID})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		total := occupancy.Available + occupancy.Held + occupancy.Occupied
		response.Success(w, http.StatusOK, "", map[string]any{"stats": map[string]any{
			"from": from.Format("2006-01-02"), "to": toExclusive.AddDate(0, 0, -1).Format("2006-01-02"),
			"views": views, "bookings_paid": counts.BookingsPaid, "bookings_approved": counts.BookingsApproved,
			"bookings_rejected": counts.BookingsRejected,
			"earnings":          earnings, "earnings_label": money.FormatRupiah(earnings),
			"occupancy": map[string]any{
				"occupied": occupancy.Occupied, "total": total, "rate": occupancyRate(occupancy.Occupied, total),
			},
			"rating": map[string]any{"average": rating.Average, "count": rating.Count},
		}})
	}
}
