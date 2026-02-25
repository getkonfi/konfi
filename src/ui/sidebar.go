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
	ti.Placeholder = "filter"
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

// collapsed returns true when the sidebar is in icon-rail mode.
func (s sidebar) collapsed() bool {
	return s.width <= 6
}

func (s sidebar) View() string {
	if s.collapsed() {
		return s.viewCollapsed()
	}
	return s.viewExpanded()
}

func (s sidebar) viewCollapsed() string {
	var b strings.Builder
	innerH := s.height // no vertical border or padding
	if innerH < 1 {
		innerH = 1
	}

	line := 0
	for fi, origIdx := range s.filtered {
		if s.items[origIdx].system {
			continue
		}
		if line > 0 {
			b.WriteByte('\n')
		}
		item := s.items[origIdx]
		isCursor := fi == s.cursor
		icon := item.icon
		if icon == "" {
			icon = "•"
		}

		var styled string
		switch {
		case isCursor:
			styled = s.theme.Primary.Render(icon)
		case !item.installed:
			styled = s.theme.Muted.Render(icon)
		default:
			styled = s.theme.Subtext.Render(icon)
		}
		b.WriteString(styled)
		line++
	}

	// system items at the bottom
	var sysIcons []string
	for fi, origIdx := range s.filtered {
		if !s.items[origIdx].system {
			continue
		}
		item := s.items[origIdx]
		icon := item.icon
		if icon == "" {
			icon = "•"
		}
		isCursor := fi == s.cursor
		if isCursor {
			sysIcons = append(sysIcons, s.theme.Primary.Render(icon))
		} else {
			sysIcons = append(sysIcons, s.theme.Muted.Render(icon))
		}
	}

	topStr := b.String()
	topLines := strings.Count(topStr, "\n") + 1
	if len(sysIcons) > 0 {
		botStr := strings.Join(sysIcons, "\n")
		botLines := len(sysIcons)
		gap := innerH - topLines - botLines
		if gap < 1 {
			gap = 1
		}
		topStr = topStr + strings.Repeat("\n", gap) + botStr
	}

	style := lipgloss.NewStyle().
		Padding(0, 1).
		Width(s.width).
		Height(s.height).
		Align(lipgloss.Center, lipgloss.Top)
	return style.Render(topStr)
}

func (s sidebar) viewExpanded() string {
	var top, bot strings.Builder
	innerW := s.width - 2 - 2 // border + padding
	if innerW < 6 {
		innerW = 6
	}

	// search box (only when active)
	if s.searching {
		prompt := s.theme.Primary.Render("/ ")
		top.WriteString(prompt + s.search.View())
		top.WriteByte('\n')
		top.WriteString(s.theme.Muted.Render(strings.Repeat("─", innerW)))
	}

	if len(s.filtered) == 0 {
		top.WriteByte('\n')
		top.WriteString(s.theme.Muted.Render("no matches"))
	} else {
		for fi, origIdx := range s.filtered {
			item := s.items[origIdx]
			isCursor := fi == s.cursor
			line := s.renderItem(item, isCursor, innerW)

			if item.system {
				if bot.Len() > 0 {
					bot.WriteByte('\n')
				}
				bot.WriteString(line)
			} else {
				top.WriteByte('\n')
				top.WriteString(line)
			}
		}
	}

	// pin system items to bottom with gap
	topStr := top.String()
	topLines := strings.Count(topStr, "\n") + 1
	innerH := s.height // right-only border and horizontal-only padding add no height
	if innerH < 1 {
		innerH = 1
	}

	botStr := bot.String()
	if botStr != "" {
		sep := s.theme.Muted.Render(strings.Repeat("─", innerW))
		title := s.theme.Muted.Bold(true).Render("SYSTEM")
		botStr = sep + "\n" + title + "\n" + botStr
		botLines := strings.Count(botStr, "\n") + 1
		gap := innerH - topLines - botLines
		if gap < 1 {
			gap = 1
		}
		return s.renderPanel(topStr + strings.Repeat("\n", gap) + botStr)
	}

	return s.renderPanel(topStr)
}

func (s sidebar) renderItem(item sidebarItem, isCursor bool, width int) string {
	iconGlyph := item.icon
	if iconGlyph == "" {
		iconGlyph = "•"
	}

	// when sidebar is unfocused, dim all items
	if !s.focused {
		iconStyle := s.theme.Muted.Faint(true)
		nameStyle := s.theme.Muted.Faint(true)
		body := iconStyle.Render(iconGlyph) + " " + nameStyle.Render(item.name)
		return lipgloss.NewStyle().Width(width).MaxWidth(width).Render("  " + body)
	}

	var iconStyle, nameStyle lipgloss.Style
	if item.installed {
		iconStyle = s.theme.Subtext
		nameStyle = s.theme.Text
	} else {
		iconStyle = s.theme.Muted
		nameStyle = s.theme.Muted
	}

	if isCursor {
		iconStyle = s.theme.Primary
		if item.installed {
			nameStyle = s.theme.Primary
		}
	}

	body := iconStyle.Render(iconGlyph) + " " + nameStyle.Render(item.name)
	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render("  " + body)
}

// renderPanel wraps content in the sidebar style with right-edge border.
func (s sidebar) renderPanel(content string) string {
	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.Border{Right: "│"}).
		BorderRight(true).
		BorderForeground(s.theme.Palette.Border).
		Padding(0, 1).
		Width(s.width - 1). // subtract border char
		Height(s.height).
		Align(lipgloss.Left, lipgloss.Top)

	if s.focused {
		style = style.BorderForeground(s.theme.Palette.BorderFocus)
	}

	return style.Render(content)
}
