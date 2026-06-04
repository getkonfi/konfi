package ui

import (
	"strings"

	"github.com/eminert/konfi/theme"

	"charm.land/lipgloss/v2"
)

type helpBinding struct {
	Key  string
	Desc string
}

type helpGroup struct {
	Title    string
	Bindings []helpBinding
}

var helpCardStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	Padding(1, 2).
	Align(lipgloss.Left, lipgloss.Top)

var helpGlobal = helpGroup{
	Title: "Global",
	Bindings: []helpBinding{
		{"ctrl+c", "quit"},
		{"ctrl+s", "save / keep preview"},
		{"ctrl+k", "command palette"},
		{"ctrl+z / ctrl+y", "undo / redo"},
		{"ctrl+o / ctrl+]", "nav back / forward"},
		{"ctrl+n / ctrl+p", "next / prev app"},
		{".", "configured only"},
		{"tab", "changed only"},
		{"← →", "switch pane"},
		{"t", "cycle theme"},
		{"?", "toggle help"},
		{"q", "quit"},
	},
}

var helpSidebar = helpGroup{
	Title: "Sidebar",
	Bindings: []helpBinding{
		{"j/k ↑↓", "navigate"},
		{"⏎ space", "open app"},
		{"/", "search"},
		{"esc", "clear search"},
		{"home/end", "jump"},
	},
}

var helpContent = helpGroup{
	Title: "Field List",
	Bindings: []helpBinding{
		{"j/k ↑↓", "navigate fields"},
		{"⏎", "edit in detail panel"},
		{"space", "collapse/expand section"},
		{"[ / ]", "prev / next section"},
		{"/", "search fields"},
		{"n / N", "next / prev match"},
		{"f", "toggle configured"},
		{"g", "effective config"},
		{"w", "what's new filter"},
		{"tab", "changed only"},
		{"m", "bookmark field"},
		{"b", "show bookmarks"},
		{"c / p", "copy / paste value"},
		{"d / ⌫", "delete field"},
		{"J / K", "scroll detail"},
		{"pgup / pgdn", "page up / down"},
		{"home / end", "first / last"},
		{"o", "open docs"},
		{"e", "open in $EDITOR"},
		{"esc", "back to sidebar"},
	},
}

var helpEditor = helpGroup{
	Title: "Editor (Detail Panel)",
	Bindings: []helpBinding{
		{"⏎ / ctrl+s", "confirm edit"},
		{"esc", "cancel edit"},
		{"↑↓ / j/k", "navigate options"},
		{"tab", "switch mode"},
		{"h/l / ←→", "adjust (slider, style)"},
		{"␣", "toggle (bool, togglemap)"},
		{"a", "add item (list, map)"},
		{"d", "delete item (list, map)"},
	},
}

// helpContext returns the help groups with the active group index based on state.
func helpContext(focus pane, editing bool) (groups []helpGroup, active int) {
	groups = []helpGroup{helpGlobal, helpSidebar, helpContent, helpEditor}
	switch {
	case editing:
		active = 3
	case focus == paneContent:
		active = 2
	case focus == paneSidebar:
		active = 1
	default:
		active = 0
	}
	return groups, active
}

// renderHelpCard renders a centered, bordered help card.
func renderHelpCard(width, height int, focus pane, editing bool, th *theme.Theme) string {
	groups, active := helpContext(focus, editing)

	cardW := width * 60 / 100
	if cardW < 40 {
		cardW = 40
	}
	if cardW > width-4 {
		cardW = width - 4
	}
	cardH := height * 70 / 100
	if cardH < 15 {
		cardH = 15
	}
	if cardH > height-4 {
		cardH = height - 4
	}

	innerW := cardW - 4 // border + padding

	var b strings.Builder

	title := th.Primary.Bold(true).Render("  Keybindings")
	b.WriteString(title)
	b.WriteString("\n\n")

	for gi, g := range groups {
		isActive := gi == active

		var header string
		if isActive {
			header = th.Primary.Bold(true).Render("▸ " + g.Title)
		} else {
			header = th.Muted.Bold(true).Render("  " + g.Title)
		}
		b.WriteString(header)
		b.WriteByte('\n')

		// compute max key width for this group
		keyW := 0
		for _, bind := range g.Bindings {
			if len(bind.Key) > keyW {
				keyW = len(bind.Key)
			}
		}
		keyW += 2 // padding after key

		for _, bind := range g.Bindings {
			key := bind.Key

			var line string
			if isActive {
				k := th.Accent.Render(padRight(key, keyW))
				d := th.Text.Render(bind.Desc)
				line = "    " + k + d
			} else {
				k := th.Muted.Render(padRight(key, keyW))
				d := th.Muted.Render(bind.Desc)
				line = "    " + k + d
			}

			// truncate to innerW
			if lipgloss.Width(line) > innerW {
				line = line[:innerW]
			}
			b.WriteString(line)
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}

	b.WriteString(th.Muted.Italic(true).Render("  press ? or esc to close"))

	card := helpCardStyle.
		BorderForeground(th.Palette.Primary).
		Width(cardW).
		Height(cardH).
		Render(b.String())

	return lipgloss.Place(width, height,
		lipgloss.Center, lipgloss.Center,
		card,
		lipgloss.WithWhitespaceChars(" "),
	)
}

func padRight(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}
