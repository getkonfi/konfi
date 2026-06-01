package ui

import (
	"fmt"
	"strings"

	"github.com/emin/konfigurator/theme"

	"charm.land/lipgloss/v2"
)

// diffView renders a preview of all pending changes before saving.
// pure view component — no tea.Model, no Update.
type diffView struct {
	entries []pendingChange
	width   int
	height  int
	theme   *theme.Theme
}

func newDiffView(th *theme.Theme) *diffView {
	return &diffView{theme: th}
}

func (d *diffView) SetEntries(entries []pendingChange) { d.entries = entries }
func (d *diffView) SetSize(w, h int)                   { d.width = w; d.height = h }
func (d *diffView) HasChanges() bool                   { return len(d.entries) > 0 }
func (d *diffView) Count() int                         { return len(d.entries) }

func (d *diffView) View() string {
	if !d.HasChanges() {
		msg := d.theme.Muted.Render("No pending changes")
		return lipgloss.Place(d.width, d.height, lipgloss.Center, lipgloss.Center, msg)
	}

	th := d.theme
	maxValWidth := d.width - 6 // leave room for prefix + padding
	if maxValWidth < 20 {
		maxValWidth = 20
	}

	sep := th.FaintSeparator.Render(strings.Repeat("─", min(d.width-2, 40)))

	var blocks []string
	for i, ch := range d.entries {
		section := ch.Section
		if section == "" {
			section = "other"
		}

		// section/key header
		header := th.Primary.Render(section) +
			th.Muted.Render("/") +
			th.Text.Render(ch.Key)

		var old, n string

		switch {
		case ch.Deleted:
			old = th.Error.Render("  - " + truncate(ch.OldVal, maxValWidth))
			n = th.Muted.Render("  + ∅")
		case ch.IsNew:
			old = th.Muted.Render("  - ∅")
			n = th.Accent.Render("  + " + truncate(ch.NewVal, maxValWidth))
		default:
			old = th.Error.Render("  - " + truncate(ch.OldVal, maxValWidth))
			n = th.Accent.Render("  + " + truncate(ch.NewVal, maxValWidth))
		}

		block := fmt.Sprintf("  %s\n%s\n%s", header, old, n)
		blocks = append(blocks, block)

		if i < len(d.entries)-1 {
			blocks = append(blocks, sep)
		}
	}

	out := strings.Join(blocks, "\n")

	// trim to height if needed
	if d.height > 0 {
		lines := strings.Split(out, "\n")
		if len(lines) > d.height {
			lines = lines[:d.height-1]
			lines = append(lines, th.Muted.Render(
				fmt.Sprintf("  … %d more", len(d.entries)-countEntries(lines)),
			))
		}
		out = strings.Join(lines, "\n")
	}

	return out
}

// countEntries counts how many full entry blocks (3 lines each) fit in n lines.
func countEntries(lines []string) int {
	// each entry = header + old + new = 3 lines, separator = 1 line
	// so pattern is: 3, 1, 3, 1, 3 ...
	count := 0
	remaining := len(lines)
	for remaining >= 3 {
		count++
		remaining -= 3
		if remaining >= 1 {
			remaining-- // separator
		}
	}
	return count
}

// truncate shortens s to fit within maxWidth, appending "…" if needed.
func truncate(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	for i := range s {
		if lipgloss.Width(s[:i]) > maxWidth-1 {
			return s[:i] + "…"
		}
	}
	return s
}
