package ui

import (
	"os/exec"
	"sort"
	"strings"
	"sync"

	"github.com/eminert/konfi/pkg"
	"github.com/eminert/konfi/theme"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// font cache shared across editor instances
var (
	fontCacheMu   sync.Mutex
	fontCache     []string
	fontCacheErr  error
	fontCacheDone bool
)

func loadFontsCmd() tea.Cmd {
	return func() tea.Msg {
		fontCacheMu.Lock()
		defer fontCacheMu.Unlock()
		if fontCacheDone && fontCacheErr == nil {
			return FontsLoadedMsg{Fonts: fontCache}
		}
		out, err := exec.Command("fc-list", "--format=%{family[0]}\n").Output()
		if err != nil {
			fontCacheDone = true
			fontCacheErr = err
			return FontsLoadedMsg{Err: err}
		}
		seen := make(map[string]bool)
		var families []string
		for line := range strings.SplitSeq(string(out), "\n") {
			f := strings.TrimSpace(line)
			if f != "" && !seen[f] {
				seen[f] = true
				families = append(families, f)
			}
		}
		sort.Strings(families)
		fontCache = families
		fontCacheErr = nil
		fontCacheDone = true
		return FontsLoadedMsg{Fonts: families}
	}
}

type fontEditor struct {
	filter     textinput.Model
	all        []string // full sorted font list
	filtered   []string // subset matching filter
	cursor     int
	viewOffset int
	val        string
	loading    bool
	freetext   bool // tab toggles to raw textinput mode
	th         *theme.Theme
	field      pkg.Field
}

func (e *fontEditor) Init(field pkg.Field, currentValue string, th *theme.Theme) tea.Cmd {
	e.field = field
	e.th = th
	e.loading = true
	e.val = currentValue

	e.filter = newFieldInput(th)
	e.filter.Placeholder = "filter fonts..."
	// don't pre-fill filter — show full list, cursor finds current font after load

	return tea.Batch(e.filter.Focus(), loadFontsCmd())
}

func (e *fontEditor) Update(msg tea.Msg) (tea.Cmd, bool, bool) {
	switch msg := msg.(type) {
	case FontsLoadedMsg:
		e.loading = false
		if msg.Err != nil || len(msg.Fonts) == 0 {
			e.freetext = true
			return nil, false, false
		}
		e.all = msg.Fonts
		e.refilter()
		// position cursor on the current font
		if e.val != "" {
			for i, name := range e.filtered {
				if strings.EqualFold(name, e.val) {
					e.cursor = i
					e.scrollToCursor()
					break
				}
			}
		}
		return nil, false, false

	case tea.KeyPressMsg:
		if e.loading {
			if msg.String() == "esc" {
				return nil, true, true
			}
			return nil, false, false
		}
		if e.freetext {
			return e.updateFreetext(msg)
		}
		return e.updatePicker(msg)
	}

	// forward non-key messages to filter input
	var cmd tea.Cmd
	e.filter, cmd = e.filter.Update(msg)
	return cmd, false, false
}

func (e *fontEditor) updatePicker(msg tea.KeyPressMsg) (tea.Cmd, bool, bool) {
	switch msg.String() {
	case "j", "down":
		if e.cursor < len(e.filtered)-1 {
			e.cursor++
			e.scrollToCursor()
		}
	case "k", "up":
		if e.cursor > 0 {
			e.cursor--
			e.scrollToCursor()
		}
	case "enter":
		if len(e.filtered) > 0 && e.cursor < len(e.filtered) {
			e.val = e.filtered[e.cursor]
		} else {
			e.val = e.filter.Value()
		}
		return nil, true, false
	case "esc":
		return nil, true, true
	case "tab":
		e.freetext = true
		e.filter.SetValue(e.filter.Value()) // seed with current filter text
		e.filter.CursorEnd()
		return e.filter.Focus(), false, false
	default:
		// forward to filter input for typing
		prev := e.filter.Value()
		var cmd tea.Cmd
		e.filter, cmd = e.filter.Update(msg)
		if e.filter.Value() != prev {
			e.refilter()
		}
		return cmd, false, false
	}
	return nil, false, false
}

func (e *fontEditor) updateFreetext(msg tea.KeyPressMsg) (tea.Cmd, bool, bool) {
	switch msg.String() {
	case "enter":
		e.val = e.filter.Value()
		return nil, true, false
	case "esc":
		return nil, true, true
	case "tab":
		e.freetext = false
		e.refilter()
		return nil, false, false
	}
	var cmd tea.Cmd
	e.filter, cmd = e.filter.Update(msg)
	return cmd, false, false
}

func (e *fontEditor) refilter() {
	query := strings.ToLower(e.filter.Value())
	e.filtered = e.filtered[:0]
	for _, name := range e.all {
		if query == "" || strings.Contains(strings.ToLower(name), query) {
			e.filtered = append(e.filtered, name)
		}
	}
	e.cursor = 0
	e.viewOffset = 0
}

func (e *fontEditor) scrollToCursor() {
	maxVisible := 8
	if e.cursor < e.viewOffset {
		e.viewOffset = e.cursor
	}
	if e.cursor >= e.viewOffset+maxVisible {
		e.viewOffset = e.cursor - maxVisible + 1
	}
}

func (e *fontEditor) View(width int) string {
	if e.loading {
		return "    " + e.th.Muted.Render("loading fonts...")
	}
	if e.freetext {
		e.filter.SetWidth(width - 4)
		return "    " + e.filter.View() + "\n    " + e.th.Muted.Render("tab:picker  ⏎:commit  esc:cancel")
	}

	var b strings.Builder

	// filter input
	e.filter.SetWidth(width - 4)
	b.WriteString("    " + e.filter.View())
	b.WriteByte('\n')

	// font list
	maxVisible := 8
	end := min(e.viewOffset+maxVisible, len(e.filtered))

	for i := e.viewOffset; i < end; i++ {
		name := e.filtered[i]
		display := name
		maxW := width - 6
		if maxW > 0 && len(display) > maxW {
			display = truncate(display, maxW)
		}

		if i == e.cursor {
			b.WriteString("    " + e.th.Primary.Render("> ") + e.th.Text.Bold(true).Render(display))
		} else {
			b.WriteString("      " + e.th.Subtext.Render(display))
		}
		b.WriteByte('\n')
	}

	// count + help
	count := e.th.Muted.Render(
		formatCount(e.cursor+1, len(e.filtered)) + "  " + "j/k:nav  ⏎:select  tab:freetext  esc:cancel",
	)
	b.WriteString("    " + count)

	return b.String()
}

func (e *fontEditor) Value() string { return e.val }

func (e *fontEditor) Height() int {
	if e.loading {
		return 1
	}
	if e.freetext {
		return 2
	}
	rows := min(len(e.filtered), 8)
	return 1 + rows + 1 // filter + list + help
}

func formatCount(cur, total int) string {
	if total == 0 {
		return "(0)"
	}
	return strings.Join([]string{"(", strings.TrimSpace(formatNum(float64(cur))), " of ", strings.TrimSpace(formatNum(float64(total))), ")"}, "")
}
