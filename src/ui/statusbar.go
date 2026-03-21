package ui

import (
	"fmt"
	"strings"

	"github.com/emin/konfigurator/theme"

	"charm.land/lipgloss/v2"
)

type keyHint struct {
	Key   string
	Label string
}

type statusbar struct {
	themeName   string
	status      string
	hints       []keyHint
	width       int
	theme       *theme.Theme
	mode        string // e.g. "NORMAL", "EDIT", "SEARCH"
	undoCount   int    // number of undoable operations
	changeCount int    // number of unsaved field changes
}

func newStatusbar(th *theme.Theme) statusbar {
	return statusbar{
		themeName: th.Palette.Name,
		theme:     th,
	}
}

func (s *statusbar) SetMode(mode string)     { s.mode = mode }
func (s *statusbar) SetUndoCount(count int)  { s.undoCount = count }

func (s *statusbar) View() string {
	style := s.theme.Statusbar.Width(s.width)

	// left side: mode badge + transient status
	var leftParts []string
	if s.mode != "" {
		badgeStyle := s.theme.KeyCap
		switch s.mode {
		case "EDIT":
			badgeStyle = lipgloss.NewStyle().
				Background(s.theme.Palette.Warning).
				Foreground(s.theme.Palette.Base).
				Bold(true).Padding(0, 1)
		case "SEARCH":
			badgeStyle = lipgloss.NewStyle().
				Background(s.theme.Palette.Secondary).
				Foreground(s.theme.Palette.Base).
				Bold(true).Padding(0, 1)
		}
		modeBadge := badgeStyle.Render("[" + s.mode + "]")
		leftParts = append(leftParts, modeBadge)
	}
	if s.status != "" {
		leftParts = append(leftParts, s.theme.Subtext.Render("status ")+s.theme.Text.Render(s.status))
	} else {
		leftParts = append(leftParts, s.theme.Muted.Render("ready"))
	}
	if s.undoCount > 0 {
		undoBadge := s.theme.Muted.Render(fmt.Sprintf("↩ %d", s.undoCount))
		leftParts = append(leftParts, undoBadge)
	}
	if s.changeCount > 0 {
		changeBadge := s.theme.Warning.Render(fmt.Sprintf("%d unsaved", s.changeCount))
		leftParts = append(leftParts, changeBadge)
	}
	left := strings.Join(leftParts, "  ")

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

	// drop hints from the end until the right side fits
	for len(hintParts) > 0 {
		candidate := strings.Join(hintParts, "  ") + "  " + themeBadge
		if lipgloss.Width(candidate) <= budget {
			break
		}
		hintParts = hintParts[:len(hintParts)-1]
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
