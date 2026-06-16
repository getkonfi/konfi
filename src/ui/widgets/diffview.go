package widgets

import (
	"fmt"
	"strings"

	"github.com/getkonfi/konfi/theme"

	"charm.land/lipgloss/v2"
)

// PendingChange describes a single field change relative to the on-disk snapshot.
type PendingChange struct {
	Section string
	Label   string
	Key     string
	OldVal  string
	NewVal  string
	IsNew   bool // key wasn't in origValues
	Deleted bool // key was removed
}

// DiffView renders a preview of all pending changes before saving.
// pure view component — no tea.Model, no Update.
type DiffView struct {
	entries []PendingChange
	width   int
	height  int
	theme   *theme.Theme
}

func NewDiffView(th *theme.Theme) *DiffView {
	return &DiffView{theme: th}
}

func (d *DiffView) SetEntries(entries []PendingChange) { d.entries = entries }
func (d *DiffView) SetSize(w, h int)                   { d.width = w; d.height = h }
func (d *DiffView) SetTheme(th *theme.Theme)           { d.theme = th }
func (d *DiffView) HasChanges() bool                   { return len(d.entries) > 0 }
func (d *DiffView) Count() int                         { return len(d.entries) }

func (d *DiffView) View() string {
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
			old = th.Error.Render("  - " + theme.Truncate(ch.OldVal, maxValWidth))
			n = th.Muted.Render("  + ∅")
		case ch.IsNew:
			old = th.Muted.Render("  - ∅")
			n = th.Success.Render("  + " + theme.Truncate(ch.NewVal, maxValWidth))
		default:
			ot := theme.Truncate(ch.OldVal, maxValWidth)
			nt := theme.Truncate(ch.NewVal, maxValWidth)
			old = th.Error.Render("  - ") + RenderWordDiff(ot, nt, DiffRemoved, th)
			n = th.Success.Render("  + ") + RenderWordDiff(nt, ot, DiffAdded, th)
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
