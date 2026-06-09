package ui

import (
	"image/color"
	"strconv"
	"strings"

	"github.com/eminert/konfi/theme"

	"charm.land/lipgloss/v2"
)

type keyHint struct {
	Key   string
	Label string
}

type statusTone int

const (
	statusQuiet statusTone = iota
	statusDirty
	statusPreview
	statusSaving
	statusSaved
	statusError
)

type statusbar struct {
	themeName   string
	status      string
	hints       []keyHint
	width       int
	theme       *theme.Theme
	mode        string // e.g. "NORMAL", "EDIT", "SEARCH"
	undoCount   int    // number of undoable operations
	changeCount int    // number of unsaved field changes

	// cached badge styles
	editBadge   lipgloss.Style
	searchBadge lipgloss.Style
}

func newStatusbar(th *theme.Theme) statusbar {
	s := statusbar{
		themeName: th.Palette.Name,
		theme:     th,
	}
	s.refreshStyles()
	return s
}

func (s *statusbar) SetMode(mode string)    { s.mode = mode }
func (s *statusbar) SetUndoCount(count int) { s.undoCount = count }

func (s *statusbar) refreshStyles() {
	s.editBadge = lipgloss.NewStyle().
		Background(s.theme.Palette.Warning).
		Foreground(s.readableOn(s.theme.Palette.Warning, s.theme.Palette.Base, s.theme.Palette.Text)).
		Bold(true).
		Padding(0, 1)
	s.searchBadge = lipgloss.NewStyle().
		Background(s.theme.Palette.Secondary).
		Foreground(s.readableOn(s.theme.Palette.Secondary, s.theme.Palette.Base, s.theme.Palette.Text)).
		Bold(true).
		Padding(0, 1)
}

func (s *statusbar) View() string {
	tone := s.tone()
	style := s.bandStyle(tone)

	// left side: mode badge + transient status
	var leftParts []string
	if s.mode != "" {
		badgeStyle := s.theme.KeyCap
		switch s.mode {
		case "EDIT":
			badgeStyle = s.editBadge
		case "SEARCH":
			badgeStyle = s.searchBadge
		}
		modeBadge := badgeStyle.Render("[" + s.mode + "]")
		leftParts = append(leftParts, modeBadge)
	}
	if signal := s.signal(tone); signal != "" {
		leftParts = append(leftParts, s.signalStyle(tone).Render(signal))
	}
	statusText := s.statusText(tone)
	if statusText != "" {
		if tone == statusQuiet {
			leftParts = append(leftParts, s.quietLabelStyle().Render("status ")+s.quietTextStyle().Render(statusText))
		} else {
			leftParts = append(leftParts, s.bandTextStyle(tone).Render(statusText))
		}
	} else {
		leftParts = append(leftParts, s.quietLabelStyle().Render("ready"))
	}
	if s.undoCount > 0 {
		undoStyle := s.quietLabelStyle()
		if tone != statusQuiet {
			undoStyle = s.bandTextStyle(tone)
		}
		undoBadge := undoStyle.Render("↩ " + strconv.Itoa(s.undoCount))
		leftParts = append(leftParts, undoBadge)
	}
	if s.changeCount > 0 {
		changeStyle := s.theme.Warning
		if tone != statusQuiet {
			changeStyle = s.bandTextStyle(tone)
		}
		changeBadge := changeStyle.Render(s.changeText())
		leftParts = append(leftParts, changeBadge)
	}
	left := strings.Join(leftParts, "  ")

	// right side: key-cap hints + theme badge
	// build hint parts, then trim from the left until they fit
	var hintParts []string
	keyStyle, hintStyle, themeStyle := s.hintStyles(tone)
	for _, h := range s.hints {
		k := keyStyle.Render(h.Key)
		l := hintStyle.Render(h.Label)
		hintParts = append(hintParts, k+" "+l)
	}
	themeKey := keyStyle.Render("t")
	themeName := themeStyle.Render(s.themeName)
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

func (s *statusbar) tone() statusTone {
	text := strings.ToLower(s.status)
	switch {
	case strings.HasPrefix(text, "error:") || strings.Contains(text, "failed"):
		return statusError
	case strings.HasPrefix(text, "no unsaved"):
		return statusQuiet
	case strings.Contains(text, "preview"):
		return statusPreview
	case strings.Contains(text, "saving") || strings.Contains(text, "reverting") || strings.Contains(text, "reloading"):
		return statusSaving
	case s.changeCount > 0 || strings.Contains(text, "unsaved"):
		return statusDirty
	case strings.Contains(text, "saved") || strings.Contains(text, "reverted"):
		return statusSaved
	default:
		return statusQuiet
	}
}

func (s *statusbar) bandStyle(tone statusTone) lipgloss.Style {
	style := s.theme.Statusbar.Width(s.width)
	switch tone {
	case statusDirty:
		return s.toneBandStyle(style, s.theme.Palette.Warning)
	case statusPreview:
		return s.toneBandStyle(style, s.theme.Palette.Secondary)
	case statusSaving:
		return s.toneBandStyle(style, s.theme.Palette.Primary)
	case statusSaved:
		return s.toneBandStyle(style, s.theme.Palette.Success)
	case statusError:
		return s.toneBandStyle(style, s.theme.Palette.Error)
	default:
		return style
	}
}

func (s *statusbar) toneBandStyle(style lipgloss.Style, bg color.Color) lipgloss.Style {
	return style.
		Background(bg).
		Foreground(s.readableOn(bg, s.theme.Palette.Base, s.theme.Palette.Text)).
		Bold(true)
}

func (s *statusbar) signal(tone statusTone) string {
	text := strings.ToLower(s.status)
	switch tone {
	case statusDirty:
		return "UNSAVED"
	case statusPreview:
		return "PREVIEW"
	case statusSaving:
		switch {
		case strings.Contains(text, "revert"):
			return "REVERTING"
		case strings.Contains(text, "reload"):
			return "RELOADING"
		default:
			return "SAVING"
		}
	case statusSaved:
		return "SAVED"
	case statusError:
		return "ERROR"
	default:
		return ""
	}
}

func (s *statusbar) signalStyle(tone statusTone) lipgloss.Style {
	bg := s.theme.Palette.Base
	return lipgloss.NewStyle().
		Background(bg).
		Foreground(s.readableOn(bg, s.toneColor(tone), s.theme.Palette.Text, s.theme.Palette.Subtext)).
		Bold(true).
		Padding(0, 1)
}

func (s *statusbar) statusText(tone statusTone) string {
	if s.status != "" {
		return s.status
	}
	if tone == statusDirty {
		return "unsaved changes"
	}
	return ""
}

func (s *statusbar) changeText() string {
	if s.changeCount == 1 {
		return "1 unsaved change"
	}
	return strconv.Itoa(s.changeCount) + " unsaved changes"
}

func (s *statusbar) bandTextStyle(tone statusTone) lipgloss.Style {
	bg := s.toneColor(tone)
	return lipgloss.NewStyle().
		Background(bg).
		Foreground(s.readableOn(bg, s.theme.Palette.Base, s.theme.Palette.Text)).
		Bold(true)
}

func (s *statusbar) hintStyles(tone statusTone) (keyStyle, labelStyle, themeStyle lipgloss.Style) {
	if tone == statusQuiet {
		return s.keyCapStyle(), s.quietLabelStyle(), s.quietThemeStyle()
	}
	keyBg := s.theme.Palette.Base
	keyStyle = lipgloss.NewStyle().
		Background(keyBg).
		Foreground(s.readableOn(keyBg, s.toneColor(tone), s.theme.Palette.Text, s.theme.Palette.Subtext)).
		Bold(true).
		Padding(0, 1)
	textStyle := s.bandTextStyle(tone)
	return keyStyle, textStyle, textStyle
}

func (s *statusbar) keyCapStyle() lipgloss.Style {
	bg := s.theme.Palette.Surface
	return s.theme.KeyCap.
		Background(bg).
		Foreground(s.readableOn(bg, s.theme.Palette.Text, s.theme.Palette.Subtext))
}

func (s *statusbar) quietLabelStyle() lipgloss.Style {
	bg := s.theme.Palette.Base
	return lipgloss.NewStyle().
		Background(bg).
		Foreground(s.readableOn(bg, s.theme.Palette.Subtext, s.theme.Palette.Text))
}

func (s *statusbar) quietTextStyle() lipgloss.Style {
	bg := s.theme.Palette.Base
	return lipgloss.NewStyle().
		Background(bg).
		Foreground(s.readableOn(bg, s.theme.Palette.Text, s.theme.Palette.Subtext))
}

func (s *statusbar) quietThemeStyle() lipgloss.Style {
	bg := s.theme.Palette.Base
	return lipgloss.NewStyle().
		Background(bg).
		Foreground(s.readableOn(bg, s.theme.Palette.Primary, s.theme.Palette.Text)).
		Bold(true)
}

func (s *statusbar) readableOn(bg color.Color, preferred ...color.Color) color.Color {
	return theme.ReadableColor(bg, preferred...)
}

func (s *statusbar) toneColor(tone statusTone) color.Color {
	switch tone {
	case statusDirty:
		return s.theme.Palette.Warning
	case statusPreview:
		return s.theme.Palette.Secondary
	case statusSaving:
		return s.theme.Palette.Primary
	case statusSaved:
		return s.theme.Palette.Success
	case statusError:
		return s.theme.Palette.Error
	default:
		return s.theme.Palette.Primary
	}
}
