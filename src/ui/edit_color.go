package ui

import (
	"strings"

	"github.com/emin/konfigurator/pkg"
	"github.com/emin/konfigurator/theme"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type colorEditor struct {
	input   textinput.Model
	val     string
	oldHex  string
	th      *theme.Theme
}

func (e *colorEditor) Init(field pkg.Field, currentValue string, th *theme.Theme) tea.Cmd {
	e.th = th
	e.oldHex = normalizeHex(currentValue)

	e.input = textinput.New()
	e.input.Prompt = "┊ "
	e.input.PromptStyle = th.Muted
	e.input.TextStyle = th.Text
	e.input.SetValue(currentValue)
	e.input.CursorEnd()

	return e.input.Focus()
}

func (e *colorEditor) Update(msg tea.Msg) (tea.Cmd, bool, bool) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "enter":
			e.val = e.input.Value()
			return nil, true, false
		case "esc":
			return nil, true, true
		}
	}
	var cmd tea.Cmd
	e.input, cmd = e.input.Update(msg)
	return cmd, false, false
}

func (e *colorEditor) View(width int) string {
	e.input.Width = width - 4

	inputLine := "    " + e.input.View()

	newHex := normalizeHex(e.input.Value())
	swatch := func(hex string) string {
		return lipgloss.NewStyle().
			Background(lipgloss.Color(hex)).
			Render("      ")
	}
	swatchLine := "    " + swatch(e.oldHex) +
		e.th.Muted.Render(" old → ") +
		swatch(newHex) +
		e.th.Muted.Render(" new")

	return inputLine + "\n" + swatchLine
}

func (e *colorEditor) Value() string { return e.val }
func (e *colorEditor) Height() int   { return 2 }

// normalizeHex prepends # if missing so lipgloss can render it.
func normalizeHex(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "#000000"
	}
	if s[0] != '#' {
		return "#" + s
	}
	return s
}
