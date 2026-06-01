package theme

import "charm.land/lipgloss/v2"

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
	PreviewHL    lipgloss.Style
	KeyCap       lipgloss.Style
}

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
		Foreground(p.Subtext).
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

	t.PreviewHL = lipgloss.NewStyle().
		Foreground(p.Accent).
		Bold(true)

	t.KeyCap = lipgloss.NewStyle().
		Background(p.Surface).
		Foreground(p.Text).
		Bold(true).
		Padding(0, 1)
}

