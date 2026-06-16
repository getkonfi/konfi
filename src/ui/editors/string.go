package editors

import (
	"github.com/getkonfi/konfi/pkg"
	"github.com/getkonfi/konfi/theme"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type stringEditor struct {
	input textinput.Model
	val   string
}

func (e *stringEditor) Init(field pkg.Field, currentValue string, th *theme.Theme) tea.Cmd {
	e.input = newFieldInput(th)
	e.input.SetValue(currentValue)
	e.input.CursorEnd()
	return e.input.Focus()
}

func (e *stringEditor) Update(msg tea.Msg) (tea.Cmd, bool, bool) {
	if km, ok := msg.(tea.KeyPressMsg); ok {
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
	e.input.SetWidth(width - 4)
	return "    " + e.input.View()
}

func (e *stringEditor) InlineView(width int) string {
	e.input.SetWidth(width)
	return e.input.View()
}

func (e *stringEditor) Value() string { return e.val }
func (e *stringEditor) Height() int   { return 0 }
