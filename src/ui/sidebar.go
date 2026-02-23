package ui

import (
	"strings"

	"github.com/emin/konfigurator/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// sidebarItem holds the icon glyph and name for a rail entry.
type sidebarItem struct {
	icon string
	name string
}

type sidebar struct {
	items   []sidebarItem
	cursor  int
	focused bool
	width   int
	height  int
	theme   *theme.Theme
}

func newSidebar(items []sidebarItem, th *theme.Theme) sidebar {
	return sidebar{
		items:   items,
		cursor:  0,
		focused: true,
		theme:   th,
	}
}

func (s sidebar) Update(msg tea.Msg) (sidebar, tea.Cmd) {
	if !s.focused {
		return s, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if len(s.items) == 0 {
			return s, nil
		}

		switch msg.String() {
		case "j", "down":
			prev := s.cursor
			if s.cursor < len(s.items)-1 {
				s.cursor++
			}
			if s.cursor == prev {
				return s, nil
			}
			return s, func() tea.Msg {
				return AppSelectedMsg{Index: s.cursor, Name: s.items[s.cursor].name}
			}
		case "k", "up":
			prev := s.cursor
			if s.cursor > 0 {
				s.cursor--
			}
			if s.cursor == prev {
				return s, nil
			}
			return s, func() tea.Msg {
				return AppSelectedMsg{Index: s.cursor, Name: s.items[s.cursor].name}
			}
		case "enter", " ":
			return s, func() tea.Msg {
				return AppSelectedMsg{Index: s.cursor, Name: s.items[s.cursor].name, Confirmed: true}
			}
		case "home":
			if s.cursor == 0 {
				return s, nil
			}
			s.cursor = 0
			return s, func() tea.Msg {
				return AppSelectedMsg{Index: s.cursor, Name: s.items[s.cursor].name}
			}
		case "end":
			last := len(s.items) - 1
			if s.cursor == last {
				return s, nil
			}
			s.cursor = last
			return s, func() tea.Msg {
				return AppSelectedMsg{Index: s.cursor, Name: s.items[s.cursor].name}
			}
		}

	case ThemeChangedMsg:
		s.theme = msg.Theme
	}

	return s, nil
}

func (s sidebar) View() string {
	if len(s.items) == 0 {
		return s.renderRail("")
	}

	var b strings.Builder
	for i, item := range s.items {
		if i > 0 {
			b.WriteByte('\n')
		}

		var indicator, icon string
		if i == s.cursor {
			indicator = s.theme.Primary.Render("▎")
			icon = s.theme.Primary.Render(" " + item.icon + " ")
		} else {
			indicator = " "
			icon = s.theme.Muted.Render(" " + item.icon + " ")
		}
		b.WriteString(indicator + icon)
	}

	return s.renderRail(b.String())
}

// renderRail wraps content in the rail background style.
func (s sidebar) renderRail(content string) string {
	style := s.theme.Rail.
		Width(s.width).
		Height(s.height).
		Align(lipgloss.Center, lipgloss.Top)

	return style.Render(content)
}
