// Package money formats Rupiah amounts and provides the WIB-based date/period
// helpers SPEC-GO.md §5.5 and §13.6 require. All amounts are int64 whole Rupiah.
package money

import (
	"strconv"
	"strings"
)

// FormatRupiah renders amount as "Rp" + thousands separated by dots, e.g.
// FormatRupiah(2250000) == "Rp2.250.000", FormatRupiah(0) == "Rp0".
func FormatRupiah(amount int64) string {
	negative := amount < 0
	if negative {
		amount = -amount
	}
	digits := strconv.FormatInt(amount, 10)

	var grouped strings.Builder
	for i, d := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			grouped.WriteByte('.')
		}
		grouped.WriteRune(d)
	}

	sign := ""
	if negative {
		sign = "-"
	}
	return sign + "Rp" + grouped.String()
}
