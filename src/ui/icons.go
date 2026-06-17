package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func appIconWidth(nerd bool) int {
	if nerd {
		return 1
	}
	return 2
}

func fieldIconWidth(nerd bool) int {
	if nerd {
		return 1
	}
	return 3
}

func plainAppIcon(icon string) string {
	return strings.ReplaceAll(icon, "\ufe0f", "")
}

func iconCell(icon string, width int) string {
	if width <= 0 {
		return icon
	}
	if lipgloss.Width(icon) > width {
		icon = truncateCells(icon, width)
	}
	w := lipgloss.Width(icon)
	if w == width {
		return icon
	}
	return icon + strings.Repeat(" ", width-w)
}

func appInitials(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			continue
		}
		b.WriteRune(r)
		if b.Len() >= 2 {
			break
		}
	}
	if b.Len() == 0 {
		return "??"
	}
	return b.String()
}

func truncateCells(s string, width int) string {
	if width <= 0 {
		return ""
	}
	var b strings.Builder
	used := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if used+rw > width {
			break
		}
		b.WriteRune(r)
		used += rw
	}
	return b.String()
}
