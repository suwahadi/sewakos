package finance

import (
	"net/http"
	"time"

	"github.com/suwahadi/sewakos/backend/internal/money"
)

const maxRangeDays = 366

// ParseDateRange returns a half-open WIB range [from, toExclusive) so callers
// can filter with occurred_at >= from AND occurred_at < toExclusive without
// re-deriving the end-of-day boundary themselves.
func ParseDateRange(r *http.Request) (from, toExclusive time.Time, errs map[string][]string) {
	today := money.TodayWIB()
	from = time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, money.WIB)
	to := from.AddDate(0, 1, 0).AddDate(0, 0, -1)

	if v := r.URL.Query().Get("from"); v != "" {
		parsed, err := time.ParseInLocation(time.DateOnly, v, money.WIB)
		if err != nil {
			return time.Time{}, time.Time{}, map[string][]string{"from": {"The from field format is invalid."}}
		}
		from = parsed
	}
	if v := r.URL.Query().Get("to"); v != "" {
		parsed, err := time.ParseInLocation(time.DateOnly, v, money.WIB)
		if err != nil {
			return time.Time{}, time.Time{}, map[string][]string{"to": {"The to field format is invalid."}}
		}
		to = parsed
	}
	if to.Before(from) {
		return time.Time{}, time.Time{}, map[string][]string{"to": {"The to field must be a date after or equal to from."}}
	}
	if to.Sub(from) > (maxRangeDays-1)*24*time.Hour {
		return time.Time{}, time.Time{}, map[string][]string{"to": {"The date range must not exceed 366 days."}}
	}
	return from, to.AddDate(0, 0, 1), nil
}
