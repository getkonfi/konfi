package theme

import (
	"fmt"
	"strconv"
	"strings"
)

// FormatNum renders f as an integer when it has no fractional part, else with
// one decimal place.
func FormatNum(f float64) string {
	if f == float64(int(f)) {
		return strconv.Itoa(int(f))
	}
	return fmt.Sprintf("%.1f", f)
}

// FormatCount renders a "(cur of total)" position label, "(0)" when total is 0.
func FormatCount(cur, total int) string {
	if total == 0 {
		return "(0)"
	}
	return strings.Join([]string{"(", strings.TrimSpace(FormatNum(float64(cur))), " of ", strings.TrimSpace(FormatNum(float64(total))), ")"}, "")
}
