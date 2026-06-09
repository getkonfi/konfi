package editors

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/eminert/konfi/pkg"
	"github.com/eminert/konfi/theme"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// hookGroup represents a single hook group: a matcher + array of hooks.
type hookGroup struct {
	Matcher string     `json:"matcher"`
	Hooks   []hookItem `json:"hooks"`
}

// hookItem represents a single hook within a group.
type hookItem struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

// hookEditor provides a structured editor for hook arrays.
// implements FieldEditor with the same a/d/Enter/j/k UX as listEditor.
type hookEditor struct {
	groups []hookGroup
	cursor int
	th     *theme.Theme
	field  pkg.Field

	// sub-editing state
	editing  bool
	input    textinput.Model
	editIdx  int // index being edited, or len(groups) for append
	editStep int // 0=matcher, 1=command, 2=timeout
	editBuf  hookGroup

	// matcher completion overlay (editStep==0 only)
	matcherOptions     []string
	matcherFiltered    []string
	matcherCompIdx     int
	matcherCompVisible bool
}

func (e *hookEditor) Init(field pkg.Field, currentValue string, th *theme.Theme) tea.Cmd {
	e.th = th
	e.field = field
	e.groups = nil

	if currentValue != "" && currentValue != "[]" {
		var parsed []hookGroup
		if err := json.Unmarshal([]byte(currentValue), &parsed); err == nil {
			e.groups = parsed
		}
	}
	e.cursor = 0

	e.input = newFieldInput(th)

	if len(field.Options) > 0 {
		e.matcherOptions = make([]string, len(field.Options))
		copy(e.matcherOptions, field.Options)
	}
	return nil
}

func (e *hookEditor) Update(msg tea.Msg) (tea.Cmd, bool, bool) {
	if e.editing {
		if km, ok := msg.(tea.KeyPressMsg); ok {
			// matcher completion overlay navigation (step 0 only)
			if e.editStep == 0 && e.matcherCompVisible && len(e.matcherFiltered) > 0 {
				switch km.String() {
				case "up":
					if e.matcherCompIdx > 0 {
						e.matcherCompIdx--
					}
					return nil, false, false
				case "down":
					if e.matcherCompIdx < len(e.matcherFiltered)-1 {
						e.matcherCompIdx++
					}
					return nil, false, false
				case "tab", "right":
					e.input.SetValue(e.matcherFiltered[e.matcherCompIdx])
					e.input.CursorEnd()
					e.matcherCompVisible = false
					return nil, false, false
				case "esc":
					e.matcherCompVisible = false
					return nil, false, false
				}
			}

			switch km.String() {
			case "enter":
				e.matcherCompVisible = false
				return e.advanceStep()
			case "esc":
				e.editing = false
				e.matcherCompVisible = false
				e.input.Blur()
				return nil, false, false
			}
		}
		var cmd tea.Cmd
		e.input, cmd = e.input.Update(msg)
		if e.editStep == 0 {
			e.filterMatcherCompletions()
		}
		return cmd, false, false
	}

	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil, false, false
	}

	switch km.String() {
	case "j", "down":
		if e.cursor < len(e.groups)-1 {
			e.cursor++
		}
	case "k", "up":
		if e.cursor > 0 {
			e.cursor--
		}
	case "enter":
		if len(e.groups) > 0 && e.cursor < len(e.groups) {
			return e.startEdit(e.cursor)
		}
		return nil, true, false
	case "a":
		return e.startAdd()
	case "d":
		if len(e.groups) > 0 && e.cursor < len(e.groups) {
			e.groups = append(e.groups[:e.cursor], e.groups[e.cursor+1:]...)
			if e.cursor >= len(e.groups) && e.cursor > 0 {
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

// startAdd begins adding a new hook group.
func (e *hookEditor) startAdd() (tea.Cmd, bool, bool) {
	e.editing = true
	e.editIdx = len(e.groups)
	e.editStep = 0
	e.editBuf = hookGroup{Hooks: []hookItem{{Type: "command"}}}
	e.input.Placeholder = "matcher (e.g. Bash, * for all)"
	e.input.SetValue("")
	e.input.CursorEnd()
	return e.input.Focus(), false, false
}

// startEdit begins editing an existing hook group.
func (e *hookEditor) startEdit(idx int) (tea.Cmd, bool, bool) {
	e.editing = true
	e.editIdx = idx
	e.editStep = 0
	e.editBuf = e.groups[idx]
	// deep-copy the hooks slice so edits don't alias the original backing array
	e.editBuf.Hooks = append([]hookItem(nil), e.editBuf.Hooks...)
	if len(e.editBuf.Hooks) == 0 {
		e.editBuf.Hooks = []hookItem{{Type: "command"}}
	}
	e.input.Placeholder = "matcher"
	e.input.SetValue(e.editBuf.Matcher)
	e.input.CursorEnd()
	return e.input.Focus(), false, false
}

// advanceStep moves to the next input in the sequential add/edit flow.
func (e *hookEditor) advanceStep() (tea.Cmd, bool, bool) {
	val := strings.TrimSpace(e.input.Value())

	switch e.editStep {
	case 0: // matcher
		e.editBuf.Matcher = val
		e.editStep = 1
		e.input.Placeholder = "command"
		if len(e.editBuf.Hooks) > 0 {
			e.input.SetValue(e.editBuf.Hooks[0].Command)
		} else {
			e.input.SetValue("")
		}
		e.input.CursorEnd()
		return nil, false, false
	case 1: // command
		if val == "" {
			// command is required — stay on this step
			return nil, false, false
		}
		if len(e.editBuf.Hooks) == 0 {
			e.editBuf.Hooks = []hookItem{{Type: "command"}}
		}
		e.editBuf.Hooks[0].Command = val
		e.editStep = 2
		e.input.Placeholder = "timeout seconds (0 = none)"
		if e.editBuf.Hooks[0].Timeout > 0 {
			e.input.SetValue(strconv.Itoa(e.editBuf.Hooks[0].Timeout))
		} else {
			e.input.SetValue("")
		}
		e.input.CursorEnd()
		return nil, false, false
	case 2: // timeout
		timeout := 0
		if val != "" {
			if t, err := strconv.Atoi(val); err == nil && t > 0 {
				timeout = t
			}
		}
		e.editBuf.Hooks[0].Timeout = timeout
		// commit the group
		if e.editIdx >= len(e.groups) {
			e.groups = append(e.groups, e.editBuf)
		} else {
			e.groups[e.editIdx] = e.editBuf
		}
		e.editing = false
		e.input.Blur()
		if e.cursor >= len(e.groups) && len(e.groups) > 0 {
			e.cursor = len(e.groups) - 1
		}
		return nil, false, false
	}
	return nil, false, false
}

func (e *hookEditor) View(width int) string {
	var b strings.Builder

	if len(e.groups) == 0 && !e.editing {
		b.WriteString("    " + e.th.Muted.Render("(no hooks) press a to add"))
		return b.String()
	}

	for i, g := range e.groups {
		summary := groupSummary(g)
		switch {
		case e.editing && i == e.editIdx:
			fmt.Fprintf(&b, "    %s %s", e.th.Primary.Render(">"), e.th.Text.Bold(true).Render(summary))
			b.WriteByte('\n')
			e.input.SetWidth(width - 8)
			label := [3]string{"matcher", "command", "timeout"}[e.editStep]
			b.WriteString("    " + e.th.Muted.Render(label+": ") + e.input.View())
		case i == e.cursor:
			fmt.Fprintf(&b, "    %s %s", e.th.Primary.Render(">"), e.th.Text.Bold(true).Render(summary))
		default:
			fmt.Fprintf(&b, "      %s", e.th.Subtext.Render(summary))
		}
		if i < len(e.groups)-1 || (e.editing && e.editIdx >= len(e.groups)) {
			b.WriteByte('\n')
		}
	}

	// new group input at the end
	if e.editing && e.editIdx >= len(e.groups) {
		e.input.SetWidth(width - 8)
		label := [3]string{"matcher", "command", "timeout"}[e.editStep]
		b.WriteString("    " + e.th.Primary.Render("+ ") + e.th.Muted.Render(label+": ") + e.input.View())
	}

	// matcher completion overlay
	if e.editing && e.editStep == 0 && e.matcherCompVisible && len(e.matcherFiltered) > 0 {
		maxShow := 6
		end := min(len(e.matcherFiltered), maxShow)
		for i := 0; i < end; i++ {
			b.WriteByte('\n')
			display := e.matcherFiltered[i]
			maxW := width - 8
			if maxW > 0 && len(display) > maxW {
				display = theme.Truncate(display, maxW)
			}
			if i == e.matcherCompIdx {
				b.WriteString("      " + e.th.Primary.Render("> ") + e.th.Text.Bold(true).Render(display))
			} else {
				b.WriteString("        " + e.th.Subtext.Render(display))
			}
		}
		if len(e.matcherFiltered) > maxShow {
			b.WriteByte('\n')
			b.WriteString("        " + e.th.Muted.Render(fmt.Sprintf("… %d more", len(e.matcherFiltered)-maxShow)))
		}
	}

	if !e.editing {
		b.WriteByte('\n')
		b.WriteString("    " + e.th.Muted.Render("a:add  d:delete  ⏎:edit  ^S:done  esc:cancel"))
	}

	return b.String()
}

// CursorOffset returns the line offset of the active cursor for scroll tracking.
func (e *hookEditor) CursorOffset() int {
	if e.editing && e.editIdx < len(e.groups) {
		// editing existing: item line + input line below
		return e.editIdx + 1
	}
	if e.editing && e.editIdx >= len(e.groups) {
		// adding new: input renders after all existing groups
		return len(e.groups)
	}
	return e.cursor
}

func (e *hookEditor) Value() string {
	if len(e.groups) == 0 {
		return "[]"
	}
	data, err := json.Marshal(e.groups)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func (e *hookEditor) Interaction() InteractionKind { return InteractionList }

func (e *hookEditor) Height() int {
	h := len(e.groups)
	if e.editing {
		h++ // input line
		if e.editIdx >= len(e.groups) {
			h++ // new item placeholder
		}
		// completion overlay lines
		if e.editStep == 0 && e.matcherCompVisible && len(e.matcherFiltered) > 0 {
			shown := min(len(e.matcherFiltered), 6)
			h += shown
			if len(e.matcherFiltered) > 6 {
				h++ // "… N more" line
			}
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

func (e *hookEditor) filterMatcherCompletions() {
	if len(e.matcherOptions) == 0 {
		return
	}
	query := strings.ToLower(strings.TrimSpace(e.input.Value()))
	if query == "" {
		e.matcherFiltered = e.matcherOptions
	} else {
		var filtered []string
		for _, opt := range e.matcherOptions {
			if strings.Contains(strings.ToLower(opt), query) {
				filtered = append(filtered, opt)
			}
		}
		e.matcherFiltered = filtered
	}
	e.matcherCompVisible = len(e.matcherFiltered) > 0
	if e.matcherCompIdx >= len(e.matcherFiltered) {
		e.matcherCompIdx = 0
	}
}

// groupSummary returns a one-line display string for a hook group.
func groupSummary(g hookGroup) string {
	matcher := g.Matcher
	if matcher == "" {
		matcher = "*"
	}
	if len(g.Hooks) == 0 {
		return matcher + " → (empty)"
	}
	h := g.Hooks[0]
	cmd := h.Command
	// show just the basename for readability
	if base := filepath.Base(cmd); base != "." && base != "/" {
		cmd = base
	}
	if h.Timeout > 0 {
		return fmt.Sprintf("%s → %s (%ds)", matcher, cmd, h.Timeout)
	}
	return fmt.Sprintf("%s → %s", matcher, cmd)
}
