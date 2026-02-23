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

	// derive key/label styles that carry the statusbar background
	// so inner Render resets don't strip it
	bg := s.theme.Palette.Surface
	keyStyle := lipgloss.NewStyle().Foreground(s.theme.Palette.Subtext).Background(bg)
	labelStyle := lipgloss.NewStyle().Foreground(s.theme.Palette.Muted).Background(bg)

	// left: transient status or keyboard hints
	var left string
	if s.status != "" {
		left = labelStyle.Render(s.status)
	} else {
		var parts []string
		for _, h := range s.hints {
			parts = append(parts, keyStyle.Render(h.Key)+" "+labelStyle.Render(h.Label))
		}
		left = strings.Join(parts, labelStyle.Render(" · "))
	}

	// right: theme name
	right := labelStyle.Render(s.themeName)

	// fill middle with spaces
	gap := s.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	line := left + strings.Repeat(" ", gap) + right
	return style.Render(line)
}
