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
	field   pkg.Field

	// sub-editing state
	editing  bool
	input    textinput.Model
	editIdx  int // index being edited, or len(items) for append

	// widget-aware sub-editor (e.g. fontEditor for widget: font)
	subEditor FieldEditor
}

func (e *listEditor) Init(field pkg.Field, currentValue string, th *theme.Theme) tea.Cmd {
	e.th = th
	e.field = field
	e.items = nil
	if currentValue != "" {
		// split on newline (from MultiValueParser) or comma (from display value)
		if strings.Contains(currentValue, "\n") {
			e.items = strings.Split(currentValue, "\n")
		} else {
			e.items = strings.Split(currentValue, ", ")
		}
		// trim + remove empties
		clean := e.items[:0]
		for _, item := range e.items {
			item = strings.TrimSpace(item)
			if item != "" {
				clean = append(clean, item)
			}
		}
		e.items = clean
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
	// widget sub-editor active (e.g. font picker)
	if e.subEditor != nil {
		cmd, done, canceled := e.subEditor.Update(msg)
		if done {
			if !canceled {
				val := strings.TrimSpace(e.subEditor.Value())
				if val != "" {
					if e.editIdx >= len(e.items) {
						e.items = append(e.items, val)
					} else {
						e.items[e.editIdx] = val
					}
				}
			}
			e.subEditor = nil
			e.editing = false
			if e.cursor >= len(e.items) && len(e.items) > 0 {
				e.cursor = len(e.items) - 1
			}
			return cmd, false, false
		}
		return cmd, false, false
	}

	// plain textinput sub-editing mode
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
			return e.startEdit(e.cursor, e.items[e.cursor])
		}
		// empty list: commit (returns empty)
		return nil, true, false
	case "a":
		// append new value
		return e.startEdit(len(e.items), "")
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
		return nil, true, false
	}
	return nil, false, false
}

// startEdit begins editing an item, using a widget-aware sub-editor when available.
func (e *listEditor) startEdit(idx int, value string) (tea.Cmd, bool, bool) {
	e.editing = true
	e.editIdx = idx

	// use widget-aware sub-editor for font, path, etc.
	switch e.field.Widget {
	case "font":
		sub := &fontEditor{}
		cmd := sub.Init(e.field, value, e.th)
		e.subEditor = sub
		return cmd, false, false
	case "path":
		sub := &pathEditor{}
		cmd := sub.Init(e.field, value, e.th)
		e.subEditor = sub
		return cmd, false, false
	}

	// default: plain textinput
	e.input.SetValue(value)
	e.input.CursorEnd()
	return e.input.Focus(), false, false
}

func (e *listEditor) View(width int) string {
	// widget sub-editor active — render it below the item list
	if e.subEditor != nil {
		var b strings.Builder
		for i, item := range e.items {
			if i == e.cursor {
				fmt.Fprintf(&b, "    %s %s", e.th.Primary.Render(">"), e.th.Text.Bold(true).Render(item))
			} else {
				fmt.Fprintf(&b, "      %s", e.th.Subtext.Render(item))
			}
			b.WriteByte('\n')
		}
		b.WriteString(e.subEditor.View(width))
		return b.String()
	}

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
	if e.subEditor != nil {
		return len(e.items) + e.subEditor.Height()
	}
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
