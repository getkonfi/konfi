package ui

import (
	"github.com/emin/konfigurator/pkg"
	"github.com/emin/konfigurator/theme"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type stringEditor struct {
	input textinput.Model
	val   string
}

func (e *stringEditor) Init(field pkg.Field, currentValue string, th *theme.Theme) tea.Cmd {
	e.input = textinput.New()
	e.input.Prompt = "┊ "
	e.input.PromptStyle = th.Muted
	e.input.TextStyle = th.Text
	e.input.SetValue(currentValue)
	e.input.CursorEnd()
	return e.input.Focus()
}

func (e *stringEditor) Update(msg tea.Msg) (tea.Cmd, bool, bool) {
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

func (e *stringEditor) View(width int) string {
	e.input.Width = width - 4
	return "    " + e.input.View()
}

func (e *stringEditor) Value() string { return e.val }
func (e *stringEditor) Height() int   { return 1 }
