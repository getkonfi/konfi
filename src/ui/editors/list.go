package editors

import (
	"fmt"
	"strings"

	"github.com/eminert/konfi/pkg"
	"github.com/eminert/konfi/theme"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type listEditor struct {
	items  []string
	cursor int
	th     *theme.Theme
	field  pkg.Field

	// sub-editing state
	editing bool
	input   textinput.Model
	editIdx int // index being edited, or len(items) for append

	// widget-aware sub-editor (e.g. fontEditor for widget: font)
	subEditor FieldEditor

	// patternlist completion overlay
	completionOptions  []string
	completionFiltered []string
	completionIdx      int
	completionVisible  bool
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

	e.input = newFieldInput(th)

	if field.Widget == "patternlist" && len(field.Options) > 0 {
		e.completionOptions = make([]string, len(field.Options))
		copy(e.completionOptions, field.Options)
	}
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
			// completion overlay navigation
			if e.completionVisible && len(e.completionFiltered) > 0 {
				switch km.String() {
				case "up":
					if e.completionIdx > 0 {
						e.completionIdx--
					}
					return nil, false, false
				case "down":
					if e.completionIdx < len(e.completionFiltered)-1 {
						e.completionIdx++
					}
					return nil, false, false
				case "tab", "right":
					e.input.SetValue(e.completionFiltered[e.completionIdx])
					e.input.CursorEnd()
					e.completionVisible = false
					return nil, false, false
				case "esc":
					e.completionVisible = false
					return nil, false, false
				}
			}

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
				e.completionVisible = false
				e.input.Blur()
				if e.cursor >= len(e.items) && len(e.items) > 0 {
					e.cursor = len(e.items) - 1
				}
				return nil, false, false
			case "esc":
				e.editing = false
				e.completionVisible = false
				e.input.Blur()
				return nil, false, false
			}
		}
		var cmd tea.Cmd
		e.input, cmd = e.input.Update(msg)
		e.filterCompletions()
		return cmd, false, false
	}

	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil, false, false
	}

	switch km.String() {
	case "down":
		if e.cursor < len(e.items)-1 {
			e.cursor++
		}
	case "up":
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
	e.filterCompletions()
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

	// patternlist completion overlay
	if e.editing && e.completionVisible && len(e.completionFiltered) > 0 {
		maxShow := 6
		end := min(len(e.completionFiltered), maxShow)
		for i := 0; i < end; i++ {
			b.WriteByte('\n')
			display := e.completionFiltered[i]
			maxW := width - 8
			if maxW > 0 && len(display) > maxW {
				display = theme.Truncate(display, maxW)
			}
			if i == e.completionIdx {
				b.WriteString("      " + e.th.Primary.Render("> ") + e.th.Text.Bold(true).Render(display))
			} else {
				b.WriteString("        " + e.th.Subtext.Render(display))
			}
		}
		if len(e.completionFiltered) > maxShow {
			b.WriteByte('\n')
			b.WriteString("        " + e.th.Muted.Render(fmt.Sprintf("… %d more", len(e.completionFiltered)-maxShow)))
		}
	}

	if !e.editing {
		b.WriteByte('\n')
		b.WriteString("    " + e.th.Muted.Render("a:add  d:delete  ⏎:edit  ^S:done  esc:cancel"))
	}

	return b.String()
}

func (e *listEditor) filterCompletions() {
	if len(e.completionOptions) == 0 {
		return
	}
	query := strings.ToLower(strings.TrimSpace(e.input.Value()))
	if query == "" {
		e.completionFiltered = e.completionOptions
	} else {
		var filtered []string
		for _, opt := range e.completionOptions {
			if strings.Contains(strings.ToLower(opt), query) {
				filtered = append(filtered, opt)
			}
		}
		e.completionFiltered = filtered
	}
	e.completionVisible = len(e.completionFiltered) > 0
	if e.completionIdx >= len(e.completionFiltered) {
		e.completionIdx = 0
	}
}

// CursorOffset returns the line offset of the active cursor within the editor output.
// the content uses this to scroll the viewport to keep the active item visible.
func (e *listEditor) CursorOffset() int {
	return e.cursor
}

func (e *listEditor) Value() string {
	return strings.Join(e.items, "\n")
}

func (e *listEditor) Interaction() InteractionKind { return InteractionList }

func (e *listEditor) AcceptsMultiValue() bool { return true }

func (e *listEditor) Height() int {
	if e.subEditor != nil {
		return len(e.items) + e.subEditor.Height()
	}
	h := len(e.items)
	if e.editing && e.editIdx >= len(e.items) {
		h++ // new item line
	}
	if e.editing && e.completionVisible && len(e.completionFiltered) > 0 {
		n := min(len(e.completionFiltered), 6)
		h += n
		if len(e.completionFiltered) > 6 {
			h++ // "… N more" line
		}
	}
	if !e.editing {
		h++ // help line
	}
	if h < 1 {
		h = 1
	}
	return h
}
