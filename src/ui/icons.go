package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func iconCell(icon string, width int) string {
	if width <= 0 {
		return icon
	}
	w := lipgloss.Width(icon)
	if w >= width {
		return icon
	}
	return icon + strings.Repeat(" ", width-w)
}
