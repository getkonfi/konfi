package editors

import (
	"fmt"
	"strings"

	"github.com/getkonfi/konfi/pkg"
	"github.com/getkonfi/konfi/theme"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// structListEditor edits a list of structured items defined by item_schema.
// each item is composed of multiple parts joined by separator.
type structListEditor struct {
	items  [][]string // each item is a slice of part values
	schema []pkg.FieldPart
	sep    string
	cursor int
	th     *theme.Theme

	// sub-editing state
	editing  bool
	editIdx  int // index being edited, or len(items) for append
	editStep int // which FieldPart we're editing
	editBuf  []string

	// input for string/number parts
	input textinput.Model

	// selection state for enum parts
	enumOpts   []string
	enumCursor int
}

func (e *structListEditor) Init(field pkg.Field, currentValue string, th *theme.Theme) tea.Cmd {
	e.th = th
	e.schema = field.ItemSchema
	e.sep = field.Separator
	if e.sep == "" {
		e.sep = "="
	}
	e.items = nil

	if currentValue != "" && currentValue != "[]" {
		var lines []string
		if strings.Contains(currentValue, "\n") {
			lines = strings.Split(currentValue, "\n")
		} else {
			lines = []string{currentValue}
		}
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			e.items = append(e.items, e.parseLine(line))
		}
	}
	e.cursor = 0

	e.input = newFieldInput(th)
	return nil
}

// parseLine splits a raw value into parts using the separator.
func (e *structListEditor) parseLine(line string) []string {
	parts := strings.SplitN(line, e.sep, len(e.schema))
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	// pad to schema length
	for len(parts) < len(e.schema) {
		parts = append(parts, "")
	}
	return parts
}

func (e *structListEditor) Update(msg tea.Msg) (tea.Cmd, bool, bool) {
	if e.editing {
		return e.updateEditing(msg)
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
		if len(e.items) > 0 && e.cursor < len(e.items) {
			return e.startEdit(e.cursor)
		}
		return nil, true, false
	case "a":
		return e.startEdit(len(e.items))
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

func (e *structListEditor) updateEditing(msg tea.Msg) (tea.Cmd, bool, bool) {
	part := e.schema[e.editStep]

	if part.Type == "enum" && len(part.Options) > 0 {
		return e.updateEnum(msg)
	}

	// text/number input
	if km, ok := msg.(tea.KeyPressMsg); ok {
		switch km.String() {
		case "enter":
			return e.advanceStep()
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

func (e *structListEditor) updateEnum(msg tea.Msg) (tea.Cmd, bool, bool) {
	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil, false, false
	}
	switch km.String() {
	case "down":
		if e.enumCursor < len(e.enumOpts)-1 {
			e.enumCursor++
		}
	case "up":
		if e.enumCursor > 0 {
			e.enumCursor--
		}
	case "enter":
		e.editBuf[e.editStep] = e.enumOpts[e.enumCursor]
		return e.advanceToNext()
	case "esc":
		e.editing = false
		return nil, false, false
	}
	return nil, false, false
}

// startEdit begins editing an existing or new item.
func (e *structListEditor) startEdit(idx int) (tea.Cmd, bool, bool) {
	e.editing = true
	e.editIdx = idx
	e.editStep = 0

	if idx < len(e.items) {
		e.editBuf = make([]string, len(e.schema))
		copy(e.editBuf, e.items[idx])
	} else {
		e.editBuf = make([]string, len(e.schema))
		for i, p := range e.schema {
			e.editBuf[i] = p.Default
		}
	}

	return e.setupStep()
}

// advanceStep captures current input value and moves to next step.
func (e *structListEditor) advanceStep() (tea.Cmd, bool, bool) {
	val := strings.TrimSpace(e.input.Value())
	part := e.schema[e.editStep]
	if val == "" && part.Required {
		return nil, false, false
	}
	e.editBuf[e.editStep] = val
	return e.advanceToNext()
}

// advanceToNext moves to the next step or commits the item.
func (e *structListEditor) advanceToNext() (tea.Cmd, bool, bool) {
	e.editStep++
	if e.editStep >= len(e.schema) {
		// commit
		if e.editIdx >= len(e.items) {
			e.items = append(e.items, e.editBuf)
		} else {
			e.items[e.editIdx] = e.editBuf
		}
		e.editing = false
		e.input.Blur()
		if e.cursor >= len(e.items) && len(e.items) > 0 {
			e.cursor = len(e.items) - 1
		}
		return nil, false, false
	}
	return e.setupStep()
}

// setupStep configures the input widget for the current edit step.
func (e *structListEditor) setupStep() (tea.Cmd, bool, bool) {
	if len(e.schema) == 0 {
		e.editing = false
		return nil, false, false
	}
	part := e.schema[e.editStep]

	if part.Type == "enum" && len(part.Options) > 0 {
		e.enumOpts = part.Options
		e.enumCursor = 0
		for i, o := range part.Options {
			if o == e.editBuf[e.editStep] {
				e.enumCursor = i
				break
			}
		}
		return nil, false, false
	}

	// text/number input
	e.input.Placeholder = part.Placeholder
	if e.input.Placeholder == "" {
		label := part.Label
		if label == "" {
			label = part.Name
		}
		e.input.Placeholder = label
	}
	e.input.SetValue(e.editBuf[e.editStep])
	e.input.CursorEnd()
	return e.input.Focus(), false, false
}

// joinParts renders an item as a single display string.
func (e *structListEditor) joinParts(parts []string) string {
	return strings.Join(parts, e.sep)
}

func (e *structListEditor) View(width int) string {
	var b strings.Builder

	if len(e.items) == 0 && !e.editing {
		b.WriteString("    " + e.th.Muted.Render("(empty) press a to add"))
		return b.String()
	}

	for i, item := range e.items {
		display := e.joinParts(item)
		switch {
		case e.editing && i == e.editIdx:
			fmt.Fprintf(&b, "    %s %s", e.th.Primary.Render(">"), e.th.Text.Bold(true).Render(display))
			b.WriteByte('\n')
			b.WriteString(e.renderEditForm(width))
		case i == e.cursor:
			fmt.Fprintf(&b, "    %s %s", e.th.Primary.Render(">"), e.th.Text.Bold(true).Render(display))
		default:
			fmt.Fprintf(&b, "      %s", e.th.Subtext.Render(display))
		}
		if i < len(e.items)-1 || (e.editing && e.editIdx >= len(e.items)) {
			b.WriteByte('\n')
		}
	}

	// new item input at the end
	if e.editing && e.editIdx >= len(e.items) {
		b.WriteString("    " + e.th.Primary.Render("+ "))
		b.WriteString(e.renderEditForm(width))
	}

	if !e.editing {
		b.WriteByte('\n')
		b.WriteString("    " + e.th.Muted.Render("a:add  d:delete  ⏎:edit  ^S:done  esc:cancel"))
	}

	return b.String()
}

// renderEditForm renders the current step's input widget.
func (e *structListEditor) renderEditForm(width int) string {
	part := e.schema[e.editStep]
	label := part.Label
	if label == "" {
		label = part.Name
	}

	if part.Type == "enum" && len(part.Options) > 0 {
		var b strings.Builder
		b.WriteString("    " + e.th.Muted.Render(label+": "))
		b.WriteByte('\n')
		for i, opt := range e.enumOpts {
			if i == e.enumCursor {
				fmt.Fprintf(&b, "      %s %s", e.th.Primary.Render(">"), e.th.Text.Bold(true).Render(opt))
			} else {
				fmt.Fprintf(&b, "        %s", e.th.Subtext.Render(opt))
			}
			if i < len(e.enumOpts)-1 {
				b.WriteByte('\n')
			}
		}
		return b.String()
	}

	e.input.SetWidth(width - 8)
	return e.th.Muted.Render(label+": ") + e.input.View()
}

func (e *structListEditor) CursorOffset() int {
	if e.editing && e.editIdx < len(e.items) {
		offset := e.editIdx + 1 // item line + edit form below
		if e.schema[e.editStep].Type == "enum" && len(e.schema[e.editStep].Options) > 0 {
			offset += e.enumCursor + 1 // label line + cursor position in enum list
		}
		return offset
	}
	if e.editing && e.editIdx >= len(e.items) {
		return len(e.items)
	}
	return e.cursor
}

func (e *structListEditor) Value() string {
	flat := make([]string, len(e.items))
	for i, item := range e.items {
		flat[i] = e.joinParts(item)
	}
	return strings.Join(flat, "\n")
}

func (e *structListEditor) Interaction() InteractionKind { return InteractionList }

func (e *structListEditor) AcceptsMultiValue() bool { return true }

func (e *structListEditor) Height() int {
	h := len(e.items)
	if e.editing {
		h++ // edit form line
		if e.editIdx >= len(e.items) {
			h++ // new item placeholder
		}
		// enum part adds extra lines
		if e.schema[e.editStep].Type == "enum" && len(e.schema[e.editStep].Options) > 0 {
			h += len(e.schema[e.editStep].Options)
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
