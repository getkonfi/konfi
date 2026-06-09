package theme

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

// Theme holds the active palette and pre-computed lipgloss styles.
type Theme struct {
	Palette Palette

	// semantic styles
	Base lipgloss.Style

	Text        lipgloss.Style
	Subtext     lipgloss.Style
	Muted       lipgloss.Style
	InsightText lipgloss.Style

	Primary   lipgloss.Style
	Secondary lipgloss.Style
	Accent    lipgloss.Style

	Success lipgloss.Style
	Warning lipgloss.Style
	Error   lipgloss.Style

	// borders
	Border      lipgloss.Style
	BorderFocus lipgloss.Style

	// composite styles
	Statusbar lipgloss.Style
	Content   lipgloss.Style

	// detail panel
	Detail         lipgloss.Style
	FaintSeparator lipgloss.Style

	// icon rail + dashboard styles
	Badge        lipgloss.Style
	FieldLabel   lipgloss.Style
	FieldValue   lipgloss.Style
	FieldDefault lipgloss.Style
	FieldChanged lipgloss.Style
	FieldNew     lipgloss.Style
	FieldWarn    lipgloss.Style
	FieldStale   lipgloss.Style
	FieldDocLink lipgloss.Style
	FieldMatch   lipgloss.Style
	PreviewHL    lipgloss.Style
	KeyCap       lipgloss.Style
}

var fieldChangedOrange = cac(cc("#c2410c", "166", "3"), cc("#fb923c", "215", "3"))

// NewTheme creates a fully initialized Theme from a Palette.
func NewTheme(p *Palette) *Theme {
	t := &Theme{Palette: *p}
	t.recompute()
	return t
}

// SetPalette switches the active palette and recomputes all styles.
func (t *Theme) SetPalette(p *Palette) {
	t.Palette = *p
	t.recompute()
}

func (t *Theme) recompute() {
	p := t.Palette

	t.Base = lipgloss.NewStyle().Background(p.Base).Foreground(p.Text)

	t.Text = lipgloss.NewStyle().Foreground(p.Text)
	t.Subtext = lipgloss.NewStyle().Foreground(p.Subtext)
	t.Muted = lipgloss.NewStyle().Foreground(p.Muted)
	t.InsightText = lipgloss.NewStyle().Foreground(p.Muted).Italic(true)

	t.Primary = lipgloss.NewStyle().Foreground(p.Primary)
	t.Secondary = lipgloss.NewStyle().Foreground(p.Secondary)
	t.Accent = lipgloss.NewStyle().Foreground(p.Accent)

	t.Success = lipgloss.NewStyle().Foreground(p.Success)
	t.Warning = lipgloss.NewStyle().Foreground(p.Warning)
	t.Error = lipgloss.NewStyle().Foreground(p.Error)

	t.Border = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(p.Border)

	t.BorderFocus = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(p.BorderFocus)

	t.Statusbar = lipgloss.NewStyle().
		Background(p.Base).
		Foreground(ReadableColor(p.Base, p.Subtext, p.Text)).
		Padding(0, 1)

	t.Content = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(p.Border).
		Padding(0, 1)

	t.Detail = lipgloss.NewStyle().
		BorderStyle(lipgloss.Border{Left: "┃"}).
		BorderLeft(true).
		BorderForeground(p.Border).
		Padding(0, 1)

	t.FaintSeparator = lipgloss.NewStyle().
		Foreground(p.Muted)

	t.Badge = lipgloss.NewStyle().
		Background(p.Surface).
		Foreground(p.Text).
		Bold(true).
		Padding(0, 1)

	t.FieldLabel = lipgloss.NewStyle().
		Foreground(p.Subtext)

	t.FieldValue = lipgloss.NewStyle().
		Foreground(p.Accent)

	t.FieldDefault = lipgloss.NewStyle().
		Foreground(p.Muted)

	t.FieldChanged = lipgloss.NewStyle().
		Foreground(fieldChangedOrange).
		Bold(true)

	t.FieldNew = lipgloss.NewStyle().
		Foreground(p.Success).
		Underline(true)

	t.FieldWarn = lipgloss.NewStyle().
		Foreground(p.Warning).
		Underline(true)

	t.FieldStale = lipgloss.NewStyle().
		Foreground(p.Error).
		Faint(true).
		Strikethrough(true)

	t.FieldDocLink = lipgloss.NewStyle().
		Foreground(p.Secondary).
		Underline(true)

	t.FieldMatch = lipgloss.NewStyle().
		Foreground(p.Primary).
		Underline(true)

	t.PreviewHL = lipgloss.NewStyle().
		Foreground(p.Accent).
		Bold(true)

	t.KeyCap = lipgloss.NewStyle().
		Background(p.Surface).
		Foreground(ReadableColor(p.Surface, p.Text, p.Subtext)).
		Bold(true).
		Padding(0, 1)
}

// Truncate shortens s to fit within maxWidth, appending "…" if needed.
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

// FormatNum renders f as an integer when it has no fractional part, else with
// one decimal place.
func FormatNum(f float64) string {
	if f == float64(int(f)) {
		return strconv.Itoa(int(f))
	}
	return fmt.Sprintf("%.1f", f)
}

// FormatCount renders a "(cur of total)" position label, "(0)" when total is 0.
func FormatCount(cur, total int) string {
	if total == 0 {
		return "(0)"
	}
	return strings.Join([]string{"(", strings.TrimSpace(FormatNum(float64(cur))), " of ", strings.TrimSpace(FormatNum(float64(total))), ")"}, "")
}
