package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// renderDiffPreview renders a centered card with the diff view and confirmation hints.
func (r *root) renderDiffPreview() string {
	th := r.app.Theme

	cardW := r.width * 60 / 100
	if cardW < 40 {
		cardW = 40
	}
	if cardW > r.width-4 {
		cardW = r.width - 4
	}
	cardH := r.height * 70 / 100
	if cardH < 10 {
		cardH = 10
	}
	if cardH > r.height-4 {
		cardH = r.height - 4
	}

	innerW := cardW - 4
	innerH := cardH - 6 // border + padding + header + footer

	r.content.diffView.SetSize(innerW, innerH)

	var b strings.Builder
	b.WriteString(th.Primary.Bold(true).Render("  preview diff"))
	b.WriteString("\n\n")
	b.WriteString(r.content.diffView.View())
	b.WriteString("\n\n")
	hints := "  enter save  esc cancel"
	if r.currentAppInfo().AutoReload {
		hints = "  enter save  p live preview  esc cancel"
	}
	b.WriteString(th.Muted.Italic(true).Render(hints))

	card := helpCardStyle.
		BorderForeground(th.Palette.Primary).
		Background(th.Palette.Base).
		Width(cardW).
		Height(cardH).
		Render(b.String())

	return lipgloss.Place(r.width, r.height,
		lipgloss.Center, lipgloss.Center,
		card,
		lipgloss.WithWhitespaceChars(" "),
	)
}
