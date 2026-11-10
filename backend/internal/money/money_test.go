package money

import (
	"testing"
	"time"
)

func TestFormatRupiah(t *testing.T) {
	cases := map[int64]string{
		0:       "Rp0",
		2250000: "Rp2.250.000",
		100:     "Rp100",
		1000:    "Rp1.000",
		1000000: "Rp1.000.000",
		-250000: "-Rp250.000",
	}
	for amount, want := range cases {
		if got := FormatRupiah(amount); got != want {
			t.Errorf("FormatRupiah(%d) = %q, want %q", amount, got, want)
		}
	}
}

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, WIB)
}

func TestAddPeriodDailyWeekly(t *testing.T) {
	start := date(2026, time.November, 10)
	if got := AddPeriod(start, "daily", 3); !got.Equal(date(2026, time.November, 13)) {
		t.Errorf("daily +3 = %v", got)
	}
	if got := AddPeriod(start, "weekly", 2); !got.Equal(date(2026, time.November, 24)) {
		t.Errorf("weekly +2 = %v", got)
	}
}

func TestAddPeriodMonthlyEndOfMonthClamp(t *testing.T) {
	start := date(2026, time.January, 31) // 2026 is not a leap year: Feb has 28 days.

	if got := AddPeriod(start, "monthly", 1); !got.Equal(date(2026, time.February, 28)) {
		t.Errorf("31 Jan + 1 month = %v, want 28 Feb 2026", got)
	}
	// Period 2 must clamp from the ORIGINAL 31, not cascade from period 1's 28 Feb
	// (28 Feb + 1 month would wrongly give 28 Mar instead of 31 Mar).
	if got := AddPeriod(start, "monthly", 2); !got.Equal(date(2026, time.March, 31)) {
		t.Errorf("31 Jan + 2 months = %v, want 31 Mar 2026", got)
	}
}

func TestAddPeriodMonthlyLeapYear(t *testing.T) {
	start := date(2028, time.January, 31) // 2028 is a leap year: Feb has 29 days.
	if got := AddPeriod(start, "monthly", 1); !got.Equal(date(2028, time.February, 29)) {
		t.Errorf("31 Jan 2028 + 1 month = %v, want 29 Feb 2028", got)
	}
}

func TestAddPeriodMonthlyYearTransition(t *testing.T) {
	start := date(2026, time.December, 31)
	if got := AddPeriod(start, "monthly", 1); !got.Equal(date(2027, time.January, 31)) {
		t.Errorf("31 Dec 2026 + 1 month = %v, want 31 Jan 2027", got)
	}
}

func TestTodayWIBMatchesCurrentWIBDate(t *testing.T) {
	want := time.Now().In(WIB)
	got := TodayWIB()
	if got.Year() != want.Year() || got.Month() != want.Month() || got.Day() != want.Day() {
		t.Errorf("TodayWIB() = %v, want same date as %v", got, want)
	}
	if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 {
		t.Errorf("TodayWIB() = %v, want midnight", got)
	}
}

func TestFormatTimestamp(t *testing.T) {
	in := time.Date(2026, time.November, 10, 17, 36, 44, 123000, WIB)
	want := "2026-11-10T10:36:44.000123Z"
	if got := FormatTimestamp(in); got != want {
		t.Errorf("FormatTimestamp(%v) = %q, want %q", in, got, want)
	}
}
