package editors

import (
	"fmt"
	"strings"

	"github.com/getkonfi/konfi/pkg"
	"github.com/getkonfi/konfi/theme"

	tea "charm.land/bubbletea/v2"
)

type enumEditor struct {
	options []string
	cursor  int
	current int // index of the current value, -1 if not found
	val     string
	th      *theme.Theme
}

func (e *enumEditor) Init(field pkg.Field, currentValue string, th *theme.Theme) tea.Cmd {
	e.options = field.Options
	e.th = th
	e.current = -1
	for i, opt := range e.options {
		if opt == currentValue {
			e.cursor = i
			e.current = i
			break
		}
	}
	return nil
}

func (e *enumEditor) Update(msg tea.Msg) (tea.Cmd, bool, bool) {
	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil, false, false
	}
	switch km.String() {
	case "down":
		if e.cursor < len(e.options)-1 {
			e.cursor++
		}
	case "up":
		if e.cursor > 0 {
			e.cursor--
		}
	case "enter":
		e.val = e.options[e.cursor]
		return nil, true, false
	case "esc":
		return nil, true, true
	}
	return nil, false, false
}

func (e *enumEditor) View(width int) string {
	var b strings.Builder
	for i, opt := range e.options {
		var line string
		if i == e.cursor {
			line = fmt.Sprintf("    %s %s", e.th.Primary.Render(">"), e.th.Text.Bold(true).Render(opt))
		} else {
			line = fmt.Sprintf("      %s", e.th.Subtext.Render(opt))
		}
		if i == e.current && i != e.cursor {
			line = fmt.Sprintf("      %s", e.th.Accent.Render(opt))
		}
		b.WriteString(line)
		if i < len(e.options)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (e *enumEditor) Value() string                { return e.val }
func (e *enumEditor) Height() int                  { return len(e.options) }
func (e *enumEditor) Interaction() InteractionKind { return InteractionEnum }
