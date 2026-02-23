package ui

import (
	"strings"

	"github.com/emin/konfigurator/theme"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// sidebarItem holds the icon glyph, name, and install status for a panel entry.
type sidebarItem struct {
	icon      string
	name      string
	installed bool
	system    bool // system items render in bottom section
}

type sidebar struct {
	items     []sidebarItem
	cursor    int // index into filtered
	filtered  []int
	focused   bool
	searching bool
	search    textinput.Model
	width     int
	height    int
	theme     *theme.Theme
}

func newSidebar(items []sidebarItem, th *theme.Theme) sidebar {
	ti := textinput.New()
	ti.Placeholder = "filter..."
	ti.CharLimit = 32
	ti.Prompt = ""

	s := sidebar{
		items:  items,
		cursor: 0,
		search: ti,
		theme:  th,
	}
	s.refilter()
	return s
}

func (s *sidebar) refilter() {
	query := strings.ToLower(s.search.Value())
	s.filtered = s.filtered[:0]
	for i, item := range s.items {
		if query == "" || strings.Contains(strings.ToLower(item.name), query) {
			s.filtered = append(s.filtered, i)
		}
	}
	if s.cursor >= len(s.filtered) {
		s.cursor = len(s.filtered) - 1
	}
	if s.cursor < 0 {
		s.cursor = 0
	}
}

func (s sidebar) Update(msg tea.Msg) (sidebar, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !s.focused {
			return s, nil
		}

		if s.searching {
			return s.updateSearching(msg)
		}

		// normal mode
		switch msg.String() {
		case "/":
			s.searching = true
			s.search.Focus()
			return s, textinput.Blink
		case "j", "down":
			return s.moveDown()
		case "k", "up":
			return s.moveUp()
		case "enter", " ":
			return s.selectCurrent(true)
		case "home":
			return s.moveTo(0)
		case "end":
			return s.moveTo(len(s.filtered) - 1)
		}

	case ThemeChangedMsg:
		s.theme = msg.Theme
	}

	return s, nil
}

func (s sidebar) updateSearching(msg tea.KeyMsg) (sidebar, tea.Cmd) {
	switch msg.String() {
	case "esc":
		s.searching = false
		s.search.SetValue("")
		s.search.Blur()
		s.refilter()
		return s, nil
	case "enter":
		s.searching = false
		s.search.Blur()
		return s.selectCurrent(true)
	case "j", "down":
		return s.moveDown()
	case "k", "up":
		return s.moveUp()
	default:
		var cmd tea.Cmd
		s.search, cmd = s.search.Update(msg)
		prev := len(s.filtered)
		s.refilter()
		// emit selection change if filter changed what's under cursor
		if len(s.filtered) > 0 && (prev != len(s.filtered)) {
			idx := s.filtered[s.cursor]
			return s, tea.Batch(cmd, func() tea.Msg {
				return AppSelectedMsg{Index: idx, Name: s.items[idx].name}
			})
		}
		return s, cmd
	}
}

func (s sidebar) moveDown() (sidebar, tea.Cmd) {
	if len(s.filtered) == 0 {
		return s, nil
	}
	prev := s.cursor
	if s.cursor < len(s.filtered)-1 {
		s.cursor++
	}
	if s.cursor == prev {
		return s, nil
	}
	idx := s.filtered[s.cursor]
	return s, func() tea.Msg {
		return AppSelectedMsg{Index: idx, Name: s.items[idx].name}
	}
}

func (s sidebar) moveUp() (sidebar, tea.Cmd) {
	if len(s.filtered) == 0 {
		return s, nil
	}
	prev := s.cursor
	if s.cursor > 0 {
		s.cursor--
	}
	if s.cursor == prev {
		return s, nil
	}
	idx := s.filtered[s.cursor]
	return s, func() tea.Msg {
		return AppSelectedMsg{Index: idx, Name: s.items[idx].name}
	}
}

func (s sidebar) moveTo(pos int) (sidebar, tea.Cmd) {
	if len(s.filtered) == 0 {
		return s, nil
	}
	if pos < 0 {
		pos = 0
	}
	if pos >= len(s.filtered) {
		pos = len(s.filtered) - 1
	}
	if s.cursor == pos {
		return s, nil
	}
	s.cursor = pos
	idx := s.filtered[s.cursor]
	return s, func() tea.Msg {
		return AppSelectedMsg{Index: idx, Name: s.items[idx].name}
	}
}

func (s sidebar) selectCurrent(confirmed bool) (sidebar, tea.Cmd) {
	if len(s.filtered) == 0 {
		return s, nil
	}
	idx := s.filtered[s.cursor]
	return s, func() tea.Msg {
		return AppSelectedMsg{Index: idx, Name: s.items[idx].name, Confirmed: confirmed}
	}
}

func (s sidebar) View() string {
	var b strings.Builder

	// search box
	if s.searching {
		prompt := s.theme.Primary.Render("/ ")
		b.WriteString(prompt + s.search.View())
	} else {
		b.WriteString(s.theme.Muted.Render("/ filter..."))
	}
	b.WriteByte('\n')

	// separator
	innerW := s.width - 2 - 2 // border + padding
	if innerW < 4 {
		innerW = 4
	}
	b.WriteString(s.theme.Muted.Render(strings.Repeat("─", innerW)))
	b.WriteByte('\n')

	if len(s.filtered) == 0 {
		b.WriteString(s.theme.Muted.Render("no matches"))
	} else {
		drawnSystemSep := false
		for fi, origIdx := range s.filtered {
			item := s.items[origIdx]

			// visual separator before first system item
			if item.system && !drawnSystemSep {
				if fi > 0 {
					b.WriteByte('\n')
				}
				b.WriteString(s.theme.Muted.Render(strings.Repeat("─", innerW)))
				drawnSystemSep = true
			}

			if fi > 0 || drawnSystemSep {
				b.WriteByte('\n')
			}
			isCursor := fi == s.cursor

			var indicator, icon, name string
			if isCursor {
				indicator = " "
				icon = " " + item.icon + " "
				if item.installed {
					icon = s.theme.Primary.Render(icon)
					name = s.theme.Text.Render(item.name)
				} else {
					icon = s.theme.Muted.Render(icon)
					name = s.theme.Muted.Render(item.name)
				}
			} else {
				indicator = " "
				if item.installed {
					icon = s.theme.Subtext.Render(" " + item.icon + " ")
					name = s.theme.Subtext.Render(item.name)
				} else {
					icon = s.theme.Muted.Render(" " + item.icon + " ")
					name = s.theme.Muted.Render(item.name)
				}
			}
			b.WriteString(indicator + icon + name)
		}
	}

	return s.renderPanel(b.String())
}

// renderPanel wraps content in the sidebar bordered style.
func (s sidebar) renderPanel(content string) string {
	style := s.theme.Sidebar.
		Width(s.width - 2). // subtract border
		Height(s.height - 2).
		Align(lipgloss.Left, lipgloss.Top)

	return style.Render(content)
}
