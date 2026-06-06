package theme

import "charm.land/lipgloss/v2"

// truncate shortens s to fit within maxWidth, appending "…" if needed.
func Truncate(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	for i := range s {
		if lipgloss.Width(s[:i]) > maxWidth-1 {
			return s[:i] + "…"
		}
	}
	return s
}
