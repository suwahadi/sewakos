package money

import "time"

// WIB is Indonesia Western Time (UTC+7). Indonesia has never observed daylight
// saving, so a fixed offset is used instead of time.LoadLocation("Asia/Jakarta") —
// this keeps WIB calculations correct even on a minimal runtime image with no
// tzdata installed, and avoids a silent fallback to UTC if the zone is missing.
var WIB = time.FixedZone("WIB", 7*60*60)

// NowWIB returns the current instant in the WIB zone.
func NowWIB() time.Time {
	return time.Now().In(WIB)
}

// TodayWIB returns today's calendar date at midnight WIB.
func TodayWIB() time.Time {
	now := NowWIB()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, WIB)
}

// AddPeriod returns the date `n` periods of rentType after start (SPEC-GO.md
// §13.6): daily +1 day, weekly +7 days, monthly +1 calendar month with
// end-of-month clamping. The monthly case always clamps from start's original
// day-of-month, not from the previous period's (already clamped) result —
// e.g. starting 31 Jan, period 1 ends 28/29 Feb but period 2 ends 31 Mar, not
// 28 Mar.
func AddPeriod(start time.Time, rentType string, n int) time.Time {
	switch rentType {
	case "daily":
		return start.AddDate(0, 0, n)
	case "weekly":
		return start.AddDate(0, 0, 7*n)
	case "monthly":
		return addMonthsClamped(start, n)
	default:
		panic("money: unknown rent_type " + rentType)
	}
}

func addMonthsClamped(start time.Time, n int) time.Time {
	day := start.Day()
	firstOfTargetMonth := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, start.Location()).AddDate(0, n, 0)
	if lastDay := daysInMonth(firstOfTargetMonth.Year(), firstOfTargetMonth.Month()); day > lastDay {
		day = lastDay
	}
	return time.Date(firstOfTargetMonth.Year(), firstOfTargetMonth.Month(), day, 0, 0, 0, 0, start.Location())
}

func daysInMonth(year int, month time.Month) int {
	// Day 0 of the following month is the last day of this one.
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// FormatTimestamp renders t in UTC at microsecond precision
// ("2006-01-02T15:04:05.000000Z"), the format every created_at/read_at field
// uses. The trailing "Z" is a literal in this layout string, not a Go time
// zone verb, which is correct here only because t is explicitly in UTC.
func FormatTimestamp(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000000Z")
}
