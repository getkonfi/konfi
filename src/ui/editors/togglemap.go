package editors

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/eminert/konfi/pkg"
	"github.com/eminert/konfi/theme"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type toggleMapEntry struct {
	key     string
	enabled bool
}

type toggleMapEditor struct {
	entries []toggleMapEntry
	cursor  int
	th      *theme.Theme

	// adding new entry
	adding bool
	input  textinput.Model

	// track which keys existed in the original value and which the user changed,
	// so Value() doesn't materialize absent schema options as explicit false
	origKeys   map[string]bool
	userEdited map[string]bool
}

func (e *toggleMapEditor) Init(field pkg.Field, currentValue string, th *theme.Theme) tea.Cmd {
	e.th = th
	e.entries = nil
	e.userEdited = make(map[string]bool)

	// parse JSON object value
	parsed := make(map[string]bool)
	if currentValue != "" {
		_ = json.Unmarshal([]byte(currentValue), &parsed)
	}

	// record which keys were originally present
	e.origKeys = make(map[string]bool, len(parsed))
	for k := range parsed {
		e.origKeys[k] = true
	}

	// build entries from parsed map
	for k, v := range parsed {
		e.entries = append(e.entries, toggleMapEntry{key: k, enabled: v})
	}
	sort.Slice(e.entries, func(i, j int) bool {
		return e.entries[i].key < e.entries[j].key
	})

	// add known options from schema that aren't already present
	existing := make(map[string]bool, len(e.entries))
	for _, ent := range e.entries {
		existing[ent.key] = true
	}
	for _, opt := range field.Options {
		if !existing[opt] {
			e.entries = append(e.entries, toggleMapEntry{key: opt, enabled: false})
		}
	}

	e.input = newFieldInput(th)
	e.input.Placeholder = "key name"
	return nil
}

func (e *toggleMapEditor) Update(msg tea.Msg) (tea.Cmd, bool, bool) {
	if e.adding {
		if km, ok := msg.(tea.KeyPressMsg); ok {
			switch km.String() {
			case "enter":
				val := strings.TrimSpace(e.input.Value())
				if val != "" {
					e.entries = append(e.entries, toggleMapEntry{key: val, enabled: true})
					e.cursor = len(e.entries) - 1
					e.userEdited[val] = true
				}
				e.adding = false
				e.input.Blur()
				return nil, false, false
			case "esc":
				e.adding = false
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
	case "down":
		if e.cursor < len(e.entries)-1 {
			e.cursor++
		}
	case "up":
		if e.cursor > 0 {
			e.cursor--
		}
	case "space":
		if len(e.entries) > 0 && e.cursor < len(e.entries) {
			e.entries[e.cursor].enabled = !e.entries[e.cursor].enabled
			e.userEdited[e.entries[e.cursor].key] = true
		}
	case "a":
		e.adding = true
		e.input.SetValue("")
		return e.input.Focus(), false, false
	case "d":
		if len(e.entries) > 0 && e.cursor < len(e.entries) {
			deleted := e.entries[e.cursor].key
			e.entries = append(e.entries[:e.cursor], e.entries[e.cursor+1:]...)
			if e.cursor >= len(e.entries) && e.cursor > 0 {
				e.cursor--
			}
			e.userEdited[deleted] = true
		}
	case "enter", "ctrl+s":
		return nil, true, false
	case "esc":
		return nil, true, true
	}
	return nil, false, false
}

func (e *toggleMapEditor) View(width int) string {
	var b strings.Builder

	if len(e.entries) == 0 && !e.adding {
		b.WriteString("    " + e.th.Muted.Render("(empty) press a to add"))
		return b.String()
	}

	for i, ent := range e.entries {
		var check, label string
		if ent.enabled {
			check = e.th.Success.Render("[x]")
		} else {
			check = e.th.Muted.Render("[ ]")
		}

		if i == e.cursor {
			label = fmt.Sprintf("    %s %s %s",
				e.th.Primary.Render(">"),
				check,
				e.th.Text.Bold(true).Render(ent.key))
		} else {
			label = fmt.Sprintf("      %s %s", check, e.th.Subtext.Render(ent.key))
		}
		b.WriteString(label)
		if i < len(e.entries)-1 || e.adding {
			b.WriteByte('\n')
		}
	}

	if e.adding {
		e.input.SetWidth(width - 8)
		b.WriteString("    " + e.th.Primary.Render("+ ") + e.input.View())
	}

	if !e.adding {
		b.WriteByte('\n')
		b.WriteString("    " + e.th.Muted.Render("a:add  d:delete  ␣:toggle  ⏎:done  esc:cancel"))
	}

	return b.String()
}

func (e *toggleMapEditor) Value() string {
	m := make(map[string]bool, len(e.entries))
	for _, ent := range e.entries {
		if e.origKeys[ent.key] || e.userEdited[ent.key] {
			m[ent.key] = ent.enabled
		}
	}
	data, _ := json.Marshal(m)
	return string(data)
}

func (e *toggleMapEditor) Interaction() InteractionKind { return InteractionToggleMap }

func (e *toggleMapEditor) Height() int {
	h := len(e.entries)
	if e.adding {
		h++
	}
	if !e.adding {
		h++ // help line
	}
	if h < 1 {
		h = 1
	}
	return h
}
