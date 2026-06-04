package ui

import (
	"github.com/eminert/konfi/theme"

	"charm.land/lipgloss/v2"
)

// breadcrumb renders a navigation path like "ghostty > appearance > font-size"
type breadcrumb struct {
	app     string
	section string
	field   string
	width   int
	theme   *theme.Theme
}

func newBreadcrumb(th *theme.Theme) breadcrumb {
	return breadcrumb{theme: th}
}

func (b *breadcrumb) SetPath(app, section, field string) {
	b.app = app
	b.section = section
	b.field = field
}

func (b *breadcrumb) SetWidth(w int) {
	b.width = w
}

func (b *breadcrumb) View() string {
	if b.app == "" {
		return ""
	}

	sep := b.theme.Muted.Render(" > ")

	appStr := b.theme.Primary.Render(b.app)
	secStr := b.theme.Text.Render(b.section)
	fieldStr := ""
	if b.field != "" {
		fieldStr = b.theme.Accent.Render(b.field)
	}

	// build full breadcrumb
	parts := []string{appStr}
	if b.section != "" {
		parts = append(parts, secStr)
	}
	if fieldStr != "" {
		parts = append(parts, fieldStr)
	}

	line := joinWith(parts, sep)

	// truncate from the left if too wide
	if b.width > 0 && lipgloss.Width(line) > b.width {
		ellipsis := b.theme.Muted.Render("…")

		// try dropping leading segments until it fits
		for len(parts) > 1 {
			parts = parts[1:]
			candidate := ellipsis + joinWith(parts, sep)
			if lipgloss.Width(candidate) <= b.width {
				line = candidate
				break
			}
		}
		// last resort: single segment with ellipsis
		if len(parts) == 1 && lipgloss.Width(ellipsis+parts[0]) > b.width {
			line = ellipsis + parts[0]
		}
	}

	return line
}

func joinWith(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for _, p := range parts[1:] {
		out += sep + p
	}
	return out
}
