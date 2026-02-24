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
	var parts []string
	for _, h := range s.hints {
		k := s.theme.KeyCap.Render(h.Key)
		l := s.theme.Muted.Render(h.Label)
		parts = append(parts, k+" "+l)
	}
	themeKey := s.theme.KeyCap.Render("theme")
	themeName := s.theme.Primary.Bold(true).Render(s.themeName)
	parts = append(parts, themeKey+" "+themeName)
	right := strings.Join(parts, "  ")

	// fill middle with spaces
	gap := s.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	line := left + strings.Repeat(" ", gap) + right
	return style.Render(line)
}
