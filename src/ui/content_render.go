package ui

import (
	"strings"

	"github.com/eminert/konfi/ui/editors"

	"charm.land/lipgloss/v2"
)

// logoBlockH is the fixed height of the header/logo block (lines).
const logoBlockH = 6

// wideLayoutMinW is the content panel width threshold for switching
// to the wide layout where the detail pane spans the full height.
const wideLayoutMinW = 100

const detailPaneWidthPercent = 45

// splitWidths computes the field list and detail pane widths for a horizontal split.
// detail gets a fixed share of the available width. returns detailW=0 when hidden.
func (c *content) splitWidths(innerW int) (fieldW, detailW int) {
	if c.schema == nil || c.config == nil || len(c.fields) == 0 {
		return innerW, 0
	}
	if innerW < 50 {
		return innerW, 0
	}
	detailW = innerW * detailPaneWidthPercent / 100
	if detailW < 20 {
		detailW = 20
	}
	fieldW = innerW - detailW
	if fieldW < 30 {
		fieldW = 30
		detailW = innerW - fieldW
	}
	if detailW < 20 {
		return innerW, 0
	}
	return fieldW, detailW
}

func (c *content) fieldListHeight() int {
	bodyH := c.height - logoBlockH
	// breadcrumb takes 1 line when an app is loaded
	if c.breadcrumb.app != "" {
		bodyH--
	}
	h := bodyH - c.fieldAreaOverhead()
	if h < 3 {
		h = 3
	}
	return h
}

func (c *content) pageSize() int {
	p := c.fieldListHeight() - 1
	if p < 1 {
		p = 1
	}
	return p
}

// fieldAreaOverhead returns the number of lines before the first field row
// in the field area (tabs + search bar). used by cursorLine for scroll.
func (c *content) fieldAreaOverhead() int {
	h := 0
	if c.schema != nil && len(c.schema.Sections) > 1 {
		h++ // tab bar line
	}
	if c.searching || len(c.searchMatches) > 0 {
		h++ // search bar line
	}
	if c.filterIndicatorVisible() {
		h++ // filter indicator line
	}
	return h
}

// filterIndicatorVisible returns true when a filter indicator line should be shown.
func (c *content) filterIndicatorVisible() bool {
	return !c.searching && (c.configuredOnly || c.changedOnly || c.showEffective || c.bookmarkedOnly)
}

// cursorLine returns the rendered line number for the current cursor position
// within the field area (relative to the scrollable body, not the full view).
func (c *content) cursorLine() int {
	if c.schema == nil || len(c.visible) == 0 {
		return 0
	}
	line := c.fieldAreaOverhead()
	for i, r := range c.visible {
		if i == c.cursor {
			return line
		}
		// section headers have a blank line before them (except first)
		if r.isSection && i > 0 {
			line++
		}
		line++
	}
	return 0
}

func (c *content) View() string {
	// no border — structural division from sidebar edge and detail's left border
	innerW := c.width - 2 // 2 padding (1 each side)
	if innerW < 10 {
		innerW = 10
	}

	// recompute outerStyle only when dimensions change
	if c.layoutW != c.width || c.layoutH != c.height {
		c.outerStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Width(c.width).
			MaxWidth(c.width).
			Height(c.height).
			MaxHeight(c.height).
			Align(lipgloss.Left, lipgloss.Top)
		c.layoutW = c.width
		c.layoutH = c.height
	}
	outerStyle := c.outerStyle

	// body area below header, minus footer
	bodyH := c.height - logoBlockH
	if bodyH < 3 {
		bodyH = 3
	}

	// handle no-schema states (no detail panel, header at full width)
	if c.schema == nil {
		if c.konfable == nil {
			// dashboard — vertically centered, no header
			dash := c.renderDashboard(innerW)
			dashLines := strings.Count(dash, "\n") + 1
			topPad := (c.height - dashLines) / 3 // bias toward upper third
			if topPad < 0 {
				topPad = 0
			}
			return outerStyle.Render(strings.Repeat("\n", topPad) + dash)
		}
		headerStr := c.renderHeader(innerW)
		var bodyStr string
		switch {
		case c.config != nil:
			// cache the content string — only rebuild when config changes
			if gen := c.config.Generation(); gen != c.rawContentGen {
				c.rawContentStr = string(c.config.Content())
				c.rawContentGen = gen
			}
			bodyStr = c.theme.Text.Render(c.rawContentStr)
		default:
			msg := c.theme.Muted.Render(c.konfable.Name() + " is not installed")
			hint := c.theme.Muted.Italic(true).Render("install it to configure")
			bodyStr = centerLine(msg, innerW) + "\n" + centerLine(hint, innerW)
		}
		return outerStyle.Render(headerStr + bodyStr)
	}

	if c.hypridleDashboardActive() {
		return c.viewHypridleDashboard(outerStyle, innerW, bodyH)
	}

	fieldListW, detailW := c.splitWidths(innerW)
	wide := c.width > wideLayoutMinW && detailW > 0

	// header width: left column only in wide mode, full width in narrow
	headerW := innerW
	if wide {
		headerW = fieldListW
	}
	headerStr := c.renderHeader(headerW)

	// breadcrumb line between header and field list
	c.breadcrumb.SetWidth(fieldListW)
	crumbStr := c.breadcrumb.View()
	if crumbStr != "" {
		crumbStr += "\n"
		bodyH-- // breadcrumb takes one line from body
		if bodyH < 3 {
			bodyH = 3
		}
	}

	// auto-scroll (cursor position is relative to field area)
	if len(c.visible) > 0 {
		cl := c.cursorLine()
		if cl < c.scrollY {
			c.scrollY = cl
		}
		cursorBottom := cl
		if c.detail.editor != nil {
			if _, ok := c.detail.editor.(editors.InlineEditor); !ok {
				// for list/hook editors, track the active cursor, not the editor bottom
				if oe, ok := c.detail.editor.(editors.OffsetEditor); ok {
					cursorBottom += oe.CursorOffset() + 1
				} else {
					cursorBottom += c.detail.editor.Height() + 1
				}
			}
		}
		if cursorBottom >= c.scrollY+bodyH {
			c.scrollY = cursorBottom - bodyH + 1
		}
	}

	// render field area (tabs + search + fields — header is separate)
	body := c.renderBody(fieldListW)

	// apply scrolling to field area
	lines := strings.Split(body, "\n")
	if c.scrollY >= len(lines) {
		c.scrollY = max(0, len(lines)-1)
	}
	if c.scrollY > 0 && c.scrollY < len(lines) {
		lines = lines[c.scrollY:]
	}
	if len(lines) > bodyH {
		lines = lines[:bodyH]
	}

	fieldView := strings.Join(lines, "\n")

	if detailW == 0 {
		return outerStyle.Render(headerStr + crumbStr + fieldView)
	}

	detailContentW := detailW - 3
	if detailContentW < 10 {
		detailContentW = 10
	}

	if wide {
		// wide layout: detail spans full height, header lives in left column
		leftContent := headerStr + crumbStr + fieldView
		leftLines := strings.Count(leftContent, "\n") + 1
		for leftLines < c.height {
			leftContent += "\n"
			leftLines++
		}

		detailView := c.detail.View(detailContentW, c.height)

		detailStyle := c.theme.Detail
		if c.focused && c.detailFocused {
			detailStyle = detailStyle.BorderForeground(c.theme.Palette.BorderFocus)
		}
		detailStyled := detailStyle.
			Width(detailW - 1).
			MaxWidth(detailW).
			Height(c.height).
			MaxHeight(c.height).
			Render(detailView)

		leftCol := lipgloss.NewStyle().
			Width(fieldListW).
			MaxWidth(fieldListW).
			Height(c.height).
			MaxHeight(c.height).
			Render(leftContent)

		return outerStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, leftCol, detailStyled))
	}

	// narrow layout: header spans full width, detail shares bodyH with fields
	fieldLines := strings.Count(fieldView, "\n") + 1
	for fieldLines < bodyH {
		fieldView += "\n"
		fieldLines++
	}

	detailView := c.detail.View(detailContentW, bodyH)

	detailStyle := c.theme.Detail
	if c.focused && c.detailFocused {
		detailStyle = detailStyle.BorderForeground(c.theme.Palette.BorderFocus)
	}
	detailStyled := detailStyle.
		Width(detailW - 1).
		MaxWidth(detailW).
		Height(bodyH).
		MaxHeight(bodyH).
		Render(detailView)

	fieldCol := lipgloss.NewStyle().
		Width(fieldListW).
		MaxWidth(fieldListW).
		Height(bodyH).
		MaxHeight(bodyH).
		Render(fieldView)

	bodyRow := lipgloss.JoinHorizontal(lipgloss.Top, fieldCol, detailStyled)

	return outerStyle.Render(headerStr + crumbStr + bodyRow)
}

// centerBlock centers each line of a multi-line string within the given width.
func centerBlock(block string, width int) string {
	lines := strings.Split(block, "\n")
	for i, line := range lines {
		lines[i] = centerLine(line, width)
	}
	return strings.Join(lines, "\n")
}

// centerLine centers a single line within the given width using lipgloss.
func centerLine(line string, width int) string {
	w := lipgloss.Width(line)
	if w >= width {
		return line
	}
	pad := (width - w) / 2
	return strings.Repeat(" ", pad) + line
}
