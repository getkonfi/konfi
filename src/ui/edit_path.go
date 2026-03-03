package ui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/emin/konfigurator/pkg"
	"github.com/emin/konfigurator/theme"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type pathEditor struct {
	input       textinput.Model
	completions []string
	compCursor  int
	showComp    bool
	th          *theme.Theme
	val         string
}

func (e *pathEditor) Init(field pkg.Field, currentValue string, th *theme.Theme) tea.Cmd {
	e.th = th
	e.input = textinput.New()
	e.input.Prompt = "┊ "
	s := textinput.DefaultDarkStyles()
	s.Focused.Prompt = th.Muted
	s.Focused.Text = th.Text
	e.input.SetStyles(s)
	e.input.SetValue(currentValue)
	e.input.CursorEnd()
	return e.input.Focus()
}

func (e *pathEditor) Update(msg tea.Msg) (tea.Cmd, bool, bool) {
	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		var cmd tea.Cmd
		e.input, cmd = e.input.Update(msg)
		return cmd, false, false
	}

	if e.showComp {
		return e.updateCompletion(km)
	}

	switch km.String() {
	case "enter":
		e.val = e.input.Value()
		return nil, true, false
	case "esc":
		return nil, true, true
	case "tab":
		e.complete()
		return nil, false, false
	}

	var cmd tea.Cmd
	e.input, cmd = e.input.Update(km)
	return cmd, false, false
}

func (e *pathEditor) updateCompletion(km tea.KeyPressMsg) (tea.Cmd, bool, bool) {
	switch km.String() {
	case "j", "down":
		if e.compCursor < len(e.completions)-1 {
			e.compCursor++
		}
	case "k", "up":
		if e.compCursor > 0 {
			e.compCursor--
		}
	case "enter":
		e.applyCompletion(e.completions[e.compCursor])
		return nil, false, false
	case "tab":
		// cycle through completions
		e.compCursor = (e.compCursor + 1) % len(e.completions)
	case "esc":
		e.showComp = false
		e.completions = nil
	default:
		// any other key closes completions and forwards to input
		e.showComp = false
		e.completions = nil
		var cmd tea.Cmd
		e.input, cmd = e.input.Update(km)
		return cmd, false, false
	}
	return nil, false, false
}

func (e *pathEditor) complete() {
	raw := e.input.Value()

	// expand ~ to home directory
	if strings.HasPrefix(raw, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			raw = home + raw[1:]
			e.input.SetValue(raw)
			e.input.CursorEnd()
		}
	}

	dir := filepath.Dir(raw)
	prefix := filepath.Base(raw)

	// if raw ends with / treat it as a directory listing
	if strings.HasSuffix(raw, "/") || strings.HasSuffix(raw, string(filepath.Separator)) {
		dir = raw
		prefix = ""
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var matches []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(prefix, ".") {
			continue // skip hidden unless prefix starts with .
		}
		if prefix == "" || strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
			full := filepath.Join(dir, name)
			if entry.IsDir() {
				full += "/"
			}
			matches = append(matches, full)
		}
	}

	if len(matches) == 0 {
		return
	}

	if len(matches) == 1 {
		e.input.SetValue(matches[0])
		e.input.CursorEnd()
		e.showComp = false
		return
	}

	e.completions = matches
	e.compCursor = 0
	e.showComp = true
}

func (e *pathEditor) applyCompletion(path string) {
	e.input.SetValue(path)
	e.input.CursorEnd()
	e.showComp = false
	e.completions = nil

	// if it's a directory, immediately list contents
	if strings.HasSuffix(path, "/") {
		e.complete()
	}
}

func (e *pathEditor) View(width int) string {
	var b strings.Builder

	e.input.SetWidth(width - 4)
	b.WriteString("    " + e.input.View())

	if e.showComp && len(e.completions) > 0 {
		maxShow := 6
		end := min(len(e.completions), maxShow)
		for i := 0; i < end; i++ {
			b.WriteByte('\n')
			entry := e.completions[i]
			display := entry
			maxW := width - 6
			if maxW > 0 && len(display) > maxW {
				display = display[:maxW-1] + "…"
			}
			if i == e.compCursor {
				b.WriteString("    " + e.th.Primary.Render("> ") + e.th.Text.Bold(true).Render(display))
			} else {
				b.WriteString("      " + e.th.Subtext.Render(display))
			}
		}
	}

	b.WriteByte('\n')
	b.WriteString("    " + e.th.Muted.Render("tab:complete  ⏎:commit  esc:cancel"))

	return b.String()
}

func (e *pathEditor) Value() string { return e.val }

func (e *pathEditor) Height() int {
	h := 2 // input + help
	if e.showComp && len(e.completions) > 0 {
		n := min(len(e.completions), 6)
		h += n
	}
	return h
}
