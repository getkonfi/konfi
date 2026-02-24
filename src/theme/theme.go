package theme

import (
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
)

// Theme holds the active palette and pre-computed lipgloss styles.
type Theme struct {
	Palette Palette

	// semantic styles
	Base    lipgloss.Style
	Surface lipgloss.Style
	Overlay lipgloss.Style

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
	Title     lipgloss.Style
	Statusbar lipgloss.Style
	Sidebar   lipgloss.Style
	Content   lipgloss.Style

	// icon rail + dashboard styles
	Badge        lipgloss.Style
	FieldLabel   lipgloss.Style
	FieldValue   lipgloss.Style
	FieldDefault lipgloss.Style
	Rail         lipgloss.Style
	PreviewHL    lipgloss.Style
	RowActive    lipgloss.Style
	RowActiveDim lipgloss.Style
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
	t.Surface = lipgloss.NewStyle().Background(p.Surface).Foreground(p.Text)
	t.Overlay = lipgloss.NewStyle().Background(p.Overlay).Foreground(p.Text)

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

	t.Title = lipgloss.NewStyle().
		Foreground(p.Primary).
		Bold(true)

	t.Statusbar = lipgloss.NewStyle().
		Foreground(p.Subtext).
		Padding(0, 1)

	t.Sidebar = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(p.Border).
		Padding(0, 1)

	t.Content = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(p.Border).
		Padding(0, 1)

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

	t.Rail = lipgloss.NewStyle().
		Background(p.Surface).
		Padding(1, 0)

	t.PreviewHL = lipgloss.NewStyle().
		Foreground(p.Accent).
		Bold(true)

	t.RowActive = lipgloss.NewStyle().
		Background(p.Overlay).
		Bold(true)

	t.RowActiveDim = lipgloss.NewStyle().
		Background(p.Surface)

	t.KeyCap = lipgloss.NewStyle().
		Background(p.Surface).
		Foreground(p.Text).
		Bold(true).
		Padding(0, 1)
}

// GlamourStyle returns a glamour renderer option using palette colors.
// dark-mode only with zero margins for embedding in the content pane.
func (t *Theme) GlamourStyle() glamour.TermRendererOption {
	p := t.Palette
	noMargin := uintPtr(0)
	// extract dark TrueColor from CompleteAdaptiveColor
	col := func(c lipgloss.CompleteAdaptiveColor) *string {
		s := c.Dark.TrueColor
		return &s
	}

	return glamour.WithStyles(ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: col(p.Muted)},
			Margin:         noMargin,
		},
		Paragraph: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: col(p.Muted)},
			Margin:         noMargin,
		},
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: col(p.Accent)},
		},
		Emph: ansi.StylePrimitive{
			Italic: boolPtr(true),
			Color:  col(p.Subtext),
		},
		Strong: ansi.StylePrimitive{
			Bold:  boolPtr(true),
			Color: col(p.Text),
		},
		Link: ansi.StylePrimitive{
			Color:     col(p.Secondary),
			Underline: boolPtr(true),
		},
		LinkText: ansi.StylePrimitive{
			Color: col(p.Primary),
		},
	})
}

func boolPtr(b bool) *bool { return &b }
func uintPtr(u uint) *uint { return &u }
