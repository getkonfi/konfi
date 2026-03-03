package ui

import (
	"strings"

	"github.com/emin/konfigurator/theme"

	"charm.land/lipgloss/v2"
)

// maxChangebarLines caps the change bar height.
const maxChangebarLines = 3

// renderChangebar renders pending changes grouped by section.
// returns empty string when there are no changes.
func renderChangebar(changes []pendingChange, width int, th *theme.Theme) string {
	if len(changes) == 0 {
		return ""
	}

	// group by section
	type group struct {
		name    string
		entries []string
	}
	var groups []group
	idx := make(map[string]int)

	for _, ch := range changes {
		sec := ch.Section
		if sec == "" {
			sec = "other"
		}

		entry := changeEntry(ch, th)

		gi, ok := idx[sec]
		if !ok {
			gi = len(groups)
			idx[sec] = gi
			groups = append(groups, group{name: sec})
		}
		groups[gi].entries = append(groups[gi].entries, entry)
	}

	// render lines: one per section
	var lines []string
	for _, g := range groups {
		header := th.Muted.Render(g.name + ":")
		body := strings.Join(g.entries, th.Muted.Render(" · "))
		line := header + " " + body

		// truncate if wider than available space
		if lipgloss.Width(line) > width-2 {
			// trim entries until it fits
			for len(g.entries) > 1 {
				g.entries = g.entries[:len(g.entries)-1]
				body = strings.Join(g.entries, th.Muted.Render(" · "))
				overflow := th.Muted.Render(" +" + "…")
				line = header + " " + body + overflow
				if lipgloss.Width(line) <= width-2 {
					break
				}
			}
		}
		lines = append(lines, line)
	}

	if len(lines) > maxChangebarLines {
		extra := len(lines) - maxChangebarLines + 1
		lines = lines[:maxChangebarLines-1]
		lines = append(lines, th.Muted.Render("  …and "+strings.Repeat("", 0)+string(rune('0'+extra))+" more sections"))
	}

	return strings.Join(lines, "\n")
}

// changeEntry formats a single field change.
func changeEntry(ch pendingChange, th *theme.Theme) string {
	label := th.Text.Render(ch.Label)

	switch {
	case ch.Deleted:
		return label + " " + th.Warning.Render(ch.OldVal) + th.Muted.Render("→") + th.Muted.Render("∅")
	case ch.IsNew:
		return label + " " + th.Muted.Render("∅→") + th.Success.Render(ch.NewVal)
	default:
		return label + " " + th.Warning.Render(ch.OldVal) + th.Muted.Render("→") + th.Success.Render(ch.NewVal)
	}
}

// changebarHeight returns how many lines the changebar will occupy.
func changebarHeight(changes []pendingChange) int {
	if len(changes) == 0 {
		return 0
	}

	// count distinct sections
	sections := make(map[string]bool)
	for _, ch := range changes {
		sec := ch.Section
		if sec == "" {
			sec = "other"
		}
		sections[sec] = true
	}
	h := len(sections)
	if h > maxChangebarLines {
		h = maxChangebarLines
	}
	return h
}
