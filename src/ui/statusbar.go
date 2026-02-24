package ui

import (
	"strings"

	"github.com/emin/konfigurator/theme"

	"github.com/charmbracelet/lipgloss"
)

type keyHint struct {
	Key   string
	Label string
}

type statusbar struct {
	themeName string
	status    string
	hints     []keyHint
	width     int
	theme     *theme.Theme
}

func newStatusbar(th *theme.Theme) statusbar {
	return statusbar{
		themeName: th.Palette.Name,
		theme:     th,
	}
}

func (s *statusbar) View() string {
	style := s.theme.Statusbar.Width(s.width)

	// left side: transient status
	left := s.theme.Muted.Render("ready")
	if s.status != "" {
		left = s.theme.Subtext.Render("status ") + s.theme.Text.Render(s.status)
	}

	// right side: key-cap hints + theme badge
	// build hint parts, then trim from the left until they fit
	var hintParts []string
	for _, h := range s.hints {
		k := s.theme.KeyCap.Render(h.Key)
		l := s.theme.Muted.Render(h.Label)
		hintParts = append(hintParts, k+" "+l)
	}
	themeKey := s.theme.KeyCap.Render("theme")
	themeName := s.theme.Primary.Bold(true).Render(s.themeName)
	themeBadge := themeKey + " " + themeName

	// available space: total width minus left, padding (2), and minimum gap (2)
	budget := s.width - lipgloss.Width(left) - 4
	if budget < lipgloss.Width(themeBadge) {
		budget = lipgloss.Width(themeBadge)
	}

	// drop hints from the beginning until the right side fits
	for len(hintParts) > 0 {
		candidate := strings.Join(hintParts, "  ") + "  " + themeBadge
		if lipgloss.Width(candidate) <= budget {
			break
		}
		hintParts = hintParts[1:]
	}

	var right string
	if len(hintParts) > 0 {
		right = strings.Join(hintParts, "  ") + "  " + themeBadge
	} else {
		right = themeBadge
	}

	// fill middle with spaces
	gap := s.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	line := left + strings.Repeat(" ", gap) + right
	return style.Render(line)
}
