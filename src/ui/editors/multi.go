package editors

import (
	"fmt"
	"strings"

	"github.com/eminert/konfi/pkg"
	"github.com/eminert/konfi/theme"

	tea "charm.land/bubbletea/v2"
)

type multiEditor struct {
	options  []string
	selected map[int]bool
	cursor   int
	th       *theme.Theme
}

func (e *multiEditor) Init(field pkg.Field, currentValue string, th *theme.Theme) tea.Cmd {
	e.options = field.Options
	e.th = th
	e.selected = make(map[int]bool)

	// parse current comma-separated value
	if currentValue != "" {
		parts := strings.Split(currentValue, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			for i, opt := range e.options {
				if opt == p {
					e.selected[i] = true
				}
			}
		}
	}
	return nil
}

func (e *multiEditor) Update(msg tea.Msg) (tea.Cmd, bool, bool) {
	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil, false, false
	}
	switch km.String() {
	case "j", "down":
		if e.cursor < len(e.options)-1 {
			e.cursor++
		}
	case "k", "up":
		if e.cursor > 0 {
			e.cursor--
		}
	case "space":
		e.selected[e.cursor] = !e.selected[e.cursor]
	case "enter":
		return nil, true, false
	case "esc":
		return nil, true, true
	}
	return nil, false, false
}

func (e *multiEditor) View(width int) string {
	var b strings.Builder
	for i, opt := range e.options {
		var check, label string
		if e.selected[i] {
			check = e.th.Success.Render("[x]")
		} else {
			check = e.th.Muted.Render("[ ]")
		}

		if i == e.cursor {
			label = fmt.Sprintf("    %s %s %s",
				e.th.Primary.Render(">"),
				check,
				e.th.Text.Bold(true).Render(opt))
		} else {
			label = fmt.Sprintf("      %s %s", check, e.th.Subtext.Render(opt))
		}
		b.WriteString(label)
		if i < len(e.options)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (e *multiEditor) Value() string {
	var selected []string
	for i, opt := range e.options {
		if e.selected[i] {
			selected = append(selected, opt)
		}
	}
	return strings.Join(selected, ",")
}

func (e *multiEditor) Height() int {
	return len(e.options)
}
