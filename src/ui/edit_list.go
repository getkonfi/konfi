package ui

import (
	"fmt"
	"strings"

	"github.com/emin/konfigurator/pkg"
	"github.com/emin/konfigurator/theme"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type listEditor struct {
	items   []string
	cursor  int
	th      *theme.Theme

	// sub-editing state
	editing  bool
	input    textinput.Model
	editIdx  int // index being edited, or len(items) for append
}

func (e *listEditor) Init(field pkg.Field, currentValue string, th *theme.Theme) tea.Cmd {
	e.th = th
	e.items = nil
	if currentValue != "" {
		e.items = strings.Split(currentValue, "\n")
	}
	e.cursor = 0

	e.input = textinput.New()
	e.input.Prompt = "┊ "
	s := textinput.DefaultDarkStyles()
	s.Focused.Prompt = th.Muted
	s.Focused.Text = th.Text
	e.input.SetStyles(s)
	return nil
}

func (e *listEditor) Update(msg tea.Msg) (tea.Cmd, bool, bool) {
	// sub-editing mode: forward to textinput
	if e.editing {
		if km, ok := msg.(tea.KeyPressMsg); ok {
			switch km.String() {
			case "enter":
				val := strings.TrimSpace(e.input.Value())
				if val != "" {
					if e.editIdx >= len(e.items) {
						e.items = append(e.items, val)
					} else {
						e.items[e.editIdx] = val
					}
				}
				e.editing = false
				e.input.Blur()
				// clamp cursor
				if e.cursor >= len(e.items) && len(e.items) > 0 {
					e.cursor = len(e.items) - 1
				}
				return nil, false, false
			case "esc":
				e.editing = false
				e.input.Blur()
				return nil, false, false
			}
		}
		var cmd tea.Cmd
		e.input, cmd = e.input.Update(msg)
		return cmd, false, false
	}

	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil, false, false
	}

	switch km.String() {
	case "j", "down":
		if e.cursor < len(e.items)-1 {
			e.cursor++
		}
	case "k", "up":
		if e.cursor > 0 {
			e.cursor--
		}
	case "enter":
		// edit current item
		if len(e.items) > 0 && e.cursor < len(e.items) {
			e.editing = true
			e.editIdx = e.cursor
			e.input.SetValue(e.items[e.cursor])
			e.input.CursorEnd()
			return e.input.Focus(), false, false
		}
		// empty list: commit (returns empty)
		return nil, true, false
	case "a":
		// append new value
		e.editing = true
		e.editIdx = len(e.items)
		e.input.SetValue("")
		return e.input.Focus(), false, false
	case "d":
		if len(e.items) > 0 && e.cursor < len(e.items) {
			e.items = append(e.items[:e.cursor], e.items[e.cursor+1:]...)
			if e.cursor >= len(e.items) && e.cursor > 0 {
				e.cursor--
			}
		}
	case "esc":
		return nil, true, true
	case "ctrl+s":
		// commit all values
		return nil, true, false
	}
	return nil, false, false
}

func (e *listEditor) View(width int) string {
	var b strings.Builder

	if len(e.items) == 0 && !e.editing {
		b.WriteString("    " + e.th.Muted.Render("(empty) press a to add"))
		return b.String()
	}

	for i, item := range e.items {
		switch {
		case e.editing && i == e.editIdx:
			e.input.SetWidth(width - 8)
			b.WriteString("    " + e.th.Primary.Render("> ") + e.input.View())
		case i == e.cursor:
			fmt.Fprintf(&b, "    %s %s", e.th.Primary.Render(">"), e.th.Text.Bold(true).Render(item))
		default:
			fmt.Fprintf(&b, "      %s", e.th.Subtext.Render(item))
		}
		if i < len(e.items)-1 || (e.editing && e.editIdx >= len(e.items)) {
			b.WriteByte('\n')
		}
	}

	// new item input at the end
	if e.editing && e.editIdx >= len(e.items) {
		e.input.SetWidth(width - 8)
		b.WriteString("    " + e.th.Primary.Render("+ ") + e.input.View())
	}

	if !e.editing {
		b.WriteByte('\n')
		b.WriteString("    " + e.th.Muted.Render("a:add  d:delete  ⏎:edit  ^S:done  esc:cancel"))
	}

	return b.String()
}

func (e *listEditor) Value() string {
	return strings.Join(e.items, "\n")
}

func (e *listEditor) Height() int {
	h := len(e.items)
	if e.editing && e.editIdx >= len(e.items) {
		h++ // new item line
	}
	if !e.editing {
		h++ // help line
	}
	if h < 1 {
		h = 1
	}
	return h
}
