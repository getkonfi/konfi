package ui

import (
	"fmt"
	"strings"

	"github.com/emin/konfigurator/theme"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// sidebarItem holds the icon glyph, name, and install status for a panel entry.
type sidebarItem struct {
	icon      string
	name      string
	installed bool
	system    bool // system items render in bottom section
	home      bool // home/dashboard item (always first)
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
	dirtyApps map[string]bool // apps with unsaved changes
	newCounts map[string]int  // per-app count of "new" fields
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

	// partition: home items first, then regular apps, then system items
	for i, item := range s.items {
		if item.home && (query == "" || strings.Contains(strings.ToLower(item.name), query)) {
			s.filtered = append(s.filtered, i)
		}
	}
	for i, item := range s.items {
		if !item.home && !item.system && (query == "" || strings.Contains(strings.ToLower(item.name), query)) {
			s.filtered = append(s.filtered, i)
		}
	}
	for i, item := range s.items {
		if item.system && (query == "" || strings.Contains(strings.ToLower(item.name), query)) {
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
	case tea.KeyPressMsg:
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
		case "enter", "space":
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

func (s sidebar) updateSearching(msg tea.KeyPressMsg) (sidebar, tea.Cmd) {
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
	case "down":
		return s.moveDown()
	case "up":
		return s.moveUp()
	default:
		var cmd tea.Cmd
		s.search, cmd = s.search.Update(msg)
		prev := len(s.filtered)
		s.refilter()
		// emit selection change if filter changed what's under cursor
		if len(s.filtered) > 0 && (prev != len(s.filtered)) {
			return s, tea.Batch(cmd, s.emitSelection(s.filtered[s.cursor], false))
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
	return s, s.emitSelection(s.filtered[s.cursor], false)
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
	return s, s.emitSelection(s.filtered[s.cursor], false)
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
	return s, s.emitSelection(s.filtered[s.cursor], false)
}

// emitSelection builds an AppSelectedMsg for the item at the given sidebar item index.
// home items get Index -1; regular items get their konfable index (item index minus home count).
func (s sidebar) emitSelection(itemIdx int, confirmed bool) tea.Cmd {
	item := s.items[itemIdx]
	if item.home {
		return func() tea.Msg {
			return AppSelectedMsg{Index: -1, Name: item.name, Confirmed: confirmed}
		}
	}
	// konfable index = item index minus number of home items before it
	ki := itemIdx
	for i := 0; i < itemIdx; i++ {
		if s.items[i].home {
			ki--
		}
	}
	return func() tea.Msg {
		return AppSelectedMsg{Index: ki, Name: item.name, Confirmed: confirmed}
	}
}

func (s sidebar) selectCurrent(confirmed bool) (sidebar, tea.Cmd) {
	if len(s.filtered) == 0 {
		return s, nil
	}
	return s, s.emitSelection(s.filtered[s.cursor], confirmed)
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
	innerH := s.height
	if innerH < 1 {
		innerH = 1
	}

	line := 0
	for fi, origIdx := range s.filtered {
		item := s.items[origIdx]
		if item.system {
			continue
		}
		if line > 0 {
			b.WriteByte('\n')
		}
		isCursor := fi == s.cursor

		var glyph string
		if item.icon != "" {
			glyph = item.icon
		} else {
			glyph = string([]rune(item.name)[0])
		}
		if s.dirtyApps[item.name] {
			glyph = s.theme.Warning.Render("●")
		}

		var styled string
		switch {
		case isCursor:
			styled = s.theme.Primary.Render(glyph)
		case !item.installed:
			styled = s.theme.Muted.Render(glyph)
		default:
			styled = s.theme.Subtext.Render(glyph)
		}
		b.WriteString(styled)
		line++
	}

	// system items at the bottom
	var sysLabels []string
	for fi, origIdx := range s.filtered {
		if !s.items[origIdx].system {
			continue
		}
		item := s.items[origIdx]
		glyph := string([]rune(item.name)[0])
		if item.icon != "" {
			glyph = item.icon
		}
		isCursor := fi == s.cursor
		if isCursor {
			sysLabels = append(sysLabels, s.theme.Primary.Render(glyph))
		} else {
			sysLabels = append(sysLabels, s.theme.Muted.Render(glyph))
		}
	}

	topStr := b.String()
	topLines := strings.Count(topStr, "\n") + 1
	if len(sysLabels) > 0 {
		botStr := strings.Join(sysLabels, "\n")
		botLines := len(sysLabels)
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
		afterHome := false
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
				if item.home {
					top.WriteByte('\n')
					top.WriteString(line)
					afterHome = true
				} else {
					if afterHome {
						top.WriteByte('\n')
						top.WriteString(s.theme.Muted.Render(strings.Repeat("─", innerW)))
						afterHome = false
					}
					top.WriteByte('\n')
					top.WriteString(line)
				}
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
	name := item.name

	// dirty indicator: themed dot instead of plain *
	dirty := ""
	if s.dirtyApps[item.name] {
		dirty = " " + s.theme.Warning.Render("●")
	}

	// "what's new" badge
	badge := ""
	if n := s.newCounts[item.name]; n > 0 {
		badge = fmt.Sprintf(" %d new", n)
	}

	// icon glyph (shown before name)
	icon := ""
	if item.icon != "" {
		icon = item.icon + " "
	}

	// when sidebar is unfocused, dim all items but keep cursor
	if !s.focused {
		nameStyle := s.theme.Muted.Faint(true)
		prefix := "  "
		if isCursor {
			prefix = nameStyle.Render("▎ ")
		}
		body := nameStyle.Render(icon+name) + dirty
		if badge != "" {
			body += nameStyle.Render(badge)
		}
		return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(prefix + body)
	}

	var nameStyle lipgloss.Style
	if item.installed {
		nameStyle = s.theme.Text
	} else {
		nameStyle = s.theme.Muted
	}

	iconStyle := nameStyle
	prefix := "  "
	if isCursor {
		prefix = s.theme.Primary.Render("▎ ")
		if item.installed {
			nameStyle = s.theme.Primary
			iconStyle = s.theme.Primary
		} else {
			nameStyle = s.theme.Muted
			iconStyle = s.theme.Muted
		}
	}

	rendered := prefix + iconStyle.Render(icon) + nameStyle.Render(name) + dirty
	if badge != "" {
		rendered += s.theme.Muted.Render(badge)
	}
	return lipgloss.NewStyle().Width(width).MaxWidth(width).Render(rendered)
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
