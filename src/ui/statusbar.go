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

	// left side: transient status only
	left := s.status

	// right side: key-cap hints + theme badge
	keyCap := lipgloss.NewStyle().
		Background(s.theme.Palette.Overlay).
		Foreground(s.theme.Palette.Text)
	label := s.theme.Muted

	var parts []string
	for _, h := range s.hints {
		k := keyCap.Render(" " + h.Key + " ")
		l := label.Render(h.Label)
		parts = append(parts, k+" "+l)
	}
	themeBadge := s.theme.Primary.Render(s.themeName)
	right := strings.Join(parts, "  ") + "  " + themeBadge

	// fill middle with spaces
	gap := s.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	line := left + strings.Repeat(" ", gap) + right
	return style.Render(line)
}
