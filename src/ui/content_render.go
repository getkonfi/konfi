package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/emin/konfigurator/konfables"
	"github.com/emin/konfigurator/pkg"

	"charm.land/lipgloss/v2"
)

// field type icons — nerd font glyphs
var fieldTypeIconNerd = map[string]string{
	"string": "\uf031",
	"number": "\uf292",
	"bool":   "\uf444",
	"enum":   "\uf150",
	"color":  "\uf53f",
	"list":   "\uf03a",
	"multi":  "\uf046",

	"font":        "\uf031",
	"slider":      "\U000F1A8A",
	"path":        "\uf115",
	"stylestring": "\uf893",
	"hook":        "\uf0e7",
	"structlist":  "\uf00b",
	"patternlist": "\uf03a",
	"togglemap":   "\uf444",
}

// field type icons — plain ASCII fallback
var fieldTypeIconASCII = map[string]string{
	"string": "Aa",
	"number": "#",
	"bool":   "?!",
	"enum":   "[]",
	"color":  "##",
	"list":   "=",
	"multi":  "**",

	"font":        "Aa",
	"slider":      "~",
	"path":        "/",
	"stylestring": "Ss",
	"hook":        "!",
	"structlist":  "=",
	"patternlist": "=",
	"togglemap":   "?!",
}

// fieldIcons returns the nerd or ASCII icon map based on the flag.
func fieldIcons(nerd bool) map[string]string {
	if nerd {
		return fieldTypeIconNerd
	}
	return fieldTypeIconASCII
}

// logoBlockH is the fixed height of the header/logo block (lines).
const logoBlockH = 6

// wideLayoutMinW is the content panel width threshold for switching
// to the wide layout where the detail pane spans the full height.
const wideLayoutMinW = 100

// footerH is the fixed height of the bottom preview bar.
const footerH = 1

// splitWidths computes the field list and detail pane widths for a horizontal split.
// detail gets a fixed ~35%. returns detailW=0 when hidden.
func (c *content) splitWidths(innerW int) (fieldW, detailW int) {
	if c.schema == nil || c.config == nil || len(c.fields) == 0 {
		return innerW, 0
	}
	if innerW < 50 {
		return innerW, 0
	}
	detailW = innerW * 35 / 100
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
	bodyH := c.height - logoBlockH - footerH
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
	return !c.searching && (c.configuredOnly || c.showNewOnly || c.showEffective || c.bookmarkedOnly)
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

// labelColumnWidth computes the max label width for the active section.
// headerLeftLines returns the left column lines for the header.
func (c *content) headerLeftLines() []string {
	title := ""
	if c.konfable != nil {
		title = c.konfable.Name()
		if v, ok := c.versions[c.konfable.Name()]; ok && v != "" {
			title += " " + v
		}
	}

	path := ""
	if c.config != nil {
		path = c.config.Path
		if path == "" && c.konfable != nil {
			path = c.konfable.Info().Name
		}
	} else if c.konfable != nil {
		path = "not installed — browse only"
	}
	if c.fileState != "" {
		path += " [" + c.fileState + "]"
	}

	insight := ""
	if len(c.insightLines) > 0 {
		insight = c.insightLines[c.insightIdx%len(c.insightLines)]
	}

	return []string{title, path, insight}
}

// renderHeader produces the two-column header or narrow fallback.
// always renders exactly logoBlockH lines + trailing newline.
func (c *content) renderHeader(width int) string {
	hh := logoBlockH

	if c.konfable == nil {
		// no app selected — empty header padded to height
		lines := make([]string, hh)
		for i := range lines {
			lines[i] = ""
		}
		return strings.Join(lines, "\n") + "\n"
	}

	// build right column: logo (animated if running, static otherwise)
	var rightLines []string
	if c.logoAnim != nil && !c.logoAnim.Done {
		art := c.logoAnim.CurrentFrame().Render()
		rightLines = strings.Split(art, "\n")
	} else if logo, ok := konfables.Logos[c.konfable.Name()]; ok {
		art := logo.Render()
		rightLines = strings.Split(art, "\n")
	}
	rightW := 0
	for _, l := range rightLines {
		if w := lipgloss.Width(l); w > rightW {
			rightW = w
		}
	}
	rightBlock := strings.Join(rightLines, "\n")

	leftW := width - rightW - 2 // 2 chars gap
	if leftW < 20 {
		// narrow fallback: centered logo
		var lines []string
		if c.logoAnim != nil && !c.logoAnim.Done {
			art := c.logoAnim.CurrentFrame().Render()
			lines = append(lines, strings.Split(centerBlock(art, width), "\n")...)
		} else if logo, ok := konfables.Logos[c.konfable.Name()]; ok {
			art := logo.Render()
			lines = append(lines, strings.Split(centerBlock(art, width), "\n")...)
		}
		lines = append(lines, "")
		for len(lines) < hh {
			lines = append(lines, "")
		}
		if len(lines) > hh {
			lines = lines[:hh]
		}
		return strings.Join(lines, "\n") + "\n"
	}

	// two-column: build left lines
	leftData := c.headerLeftLines()
	if c.splitFlap != nil && !c.splitFlap.done {
		// replace with split-flap animation frames
		leftData = make([]string, len(c.splitFlap.current))
		copy(leftData, c.splitFlap.current)
	}

	// style + truncate left lines
	styledLeft := make([]string, len(leftData))
	styles := []lipgloss.Style{c.theme.Primary, c.theme.Muted, c.theme.InsightText}
	for i, line := range leftData {
		// truncate to leftW (plain text before styling)
		if len(line) > leftW {
			line = line[:leftW-1] + "…"
		}
		s := c.theme.Text
		if i < len(styles) {
			s = styles[i]
		}
		// line 1 (path): color fileState suffix
		if i == 1 && c.fileState != "" {
			switch c.fileState {
			case "unsaved":
				s = c.theme.Warning
			case "reloaded":
				s = c.theme.Accent
			case "new":
				s = c.theme.Muted
			}
		}
		// line 2 (insight): use warning style for linter diagnostics
		if i == 2 && c.insightWarningCount > 0 && len(c.insightLines) > 0 {
			idx := c.insightIdx % len(c.insightLines)
			if idx < c.insightWarningCount {
				s = c.theme.Warning
			}
		}
		styledLeft[i] = s.Render(line)
	}

	// pad left lines to headerHeight
	for len(styledLeft) < hh {
		styledLeft = append(styledLeft, "")
	}

	// build left block with fixed width for alignment
	leftBlock := lipgloss.NewStyle().Width(leftW).Render(strings.Join(styledLeft[:hh], "\n"))

	// right-align the right column
	rightStyle := lipgloss.NewStyle().Width(rightW + 2).Align(lipgloss.Right)
	// note: these two styles depend on dynamic widths, computed once per renderHeader call
	rightAligned := rightStyle.Render(rightBlock)

	joined := lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, rightAligned)

	// pad output to exactly headerHeight rows
	outLines := strings.Split(joined, "\n")
	for len(outLines) < hh {
		outLines = append(outLines, "")
	}
	if len(outLines) > hh {
		outLines = outLines[:hh]
	}

	return strings.Join(outLines, "\n") + "\n"
}

// renderDashboard builds the welcome/landing page shown before any app is selected.
func (c *content) renderDashboard(width int) string {
	var b strings.Builder

	// logo
	if logo, ok := konfables.Logos["konfigurator"]; ok {
		art := logo.Render()
		b.WriteString(centerBlock(art, width))
		b.WriteByte('\n')
	}

	// title + version
	title := c.theme.Primary.Bold(true).Render("konfigurator")
	ver := c.theme.Muted.Render(" v" + c.appVersion)
	b.WriteString(centerLine(title+ver, width))
	b.WriteByte('\n')
	b.WriteByte('\n')

	// app list
	var installed, notInstalled []dashboardApp
	var totalDeprecated, totalNew int
	for _, a := range c.dashboardApps {
		if a.installed {
			installed = append(installed, a)
			totalDeprecated += a.deprecatedCount
			totalNew += a.newCount
		} else {
			notInstalled = append(notInstalled, a)
		}
	}

	// sort installed: most configured first, then alphabetical
	sort.Slice(installed, func(i, j int) bool {
		if installed[i].configuredCount != installed[j].configuredCount {
			return installed[i].configuredCount > installed[j].configuredCount
		}
		return installed[i].name < installed[j].name
	})
	// sort not-detected alphabetically
	sort.Slice(notInstalled, func(i, j int) bool {
		return notInstalled[i].name < notInstalled[j].name
	})

	// aggregate summary — actionable signals only
	if len(installed) > 0 {
		var parts []string
		if totalNew > 0 {
			parts = append(parts, fmt.Sprintf("%d new", totalNew))
		}
		if totalDeprecated > 0 {
			parts = append(parts, fmt.Sprintf("%d deprecated", totalDeprecated))
		}
		if bm := len(c.bookmarks); bm > 0 {
			parts = append(parts, fmt.Sprintf("%d bookmarked", bm))
		}
		if len(parts) > 0 {
			summary := strings.Join(parts, " · ")
			b.WriteString(centerLine(c.theme.Muted.Render(summary), width))
			b.WriteByte('\n')
			b.WriteByte('\n')
		}
	}

	ruleW := width / 2
	if ruleW < 20 {
		ruleW = 20
	}
	if ruleW > width {
		ruleW = width
	}

	// compute column widths across both groups for alignment
	nameW, verW := 0, 0
	for _, a := range installed {
		if len(a.name) > nameW {
			nameW = len(a.name)
		}
		if len(a.version) > verW {
			verW = len(a.version)
		}
	}
	for _, a := range notInstalled {
		if len(a.name) > nameW {
			nameW = len(a.name)
		}
	}

	// build all lines first, then left-align the block at a single offset
	var lines []string
	maxW := 0

	if len(installed) > 0 {
		label := "── installed "
		pad := ruleW - len(label)
		if pad < 0 {
			pad = 0
		}
		hdr := c.theme.Muted.Render(label + strings.Repeat("─", pad))
		lines = append(lines, hdr)
		for _, a := range installed {
			icon := c.theme.Primary.Render(a.icon)
			name := c.theme.Text.Render(" " + padRight(a.name, nameW))
			ver := strings.Repeat(" ", verW+2)
			if a.version != "" {
				ver = "  " + padRight(a.version, verW)
			}
			ver = c.theme.Muted.Render(ver)
			stats := c.dashboardStats(a)
			lines = append(lines, icon+name+ver+stats)
		}
	}

	if len(notInstalled) > 0 {
		lines = append(lines, "") // blank separator
		label := "── not detected "
		pad := ruleW - len(label)
		if pad < 0 {
			pad = 0
		}
		hdr := c.theme.Muted.Render(label + strings.Repeat("─", pad))
		lines = append(lines, hdr)
		for _, a := range notInstalled {
			icon := c.theme.Muted.Faint(true).Render(a.icon)
			name := c.theme.Muted.Faint(true).Render(" " + padRight(a.name, nameW))
			ver := ""
			if a.minAppVersion != "" && a.maxAppVersion != "" {
				ver = fmt.Sprintf("  %s – %s", a.minAppVersion, a.maxAppVersion)
			} else if a.minAppVersion != "" {
				ver = fmt.Sprintf("  %s+", a.minAppVersion)
			} else if a.maxAppVersion != "" {
				ver = fmt.Sprintf("  up to %s", a.maxAppVersion)
			}
			if ver != "" {
				ver = c.theme.Muted.Faint(true).Render(ver)
			}
			lines = append(lines, icon+name+ver)
		}
	}

	// find widest line, then left-align all lines at the same offset
	for _, l := range lines {
		if w := lipgloss.Width(l); w > maxW {
			maxW = w
		}
	}
	leftPad := (width - maxW) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	prefix := strings.Repeat(" ", leftPad)
	for _, l := range lines {
		b.WriteString(prefix + l)
		b.WriteByte('\n')
	}

	b.WriteByte('\n')
	hints := []struct{ key, desc string }{
		{"↑↓", "navigate"},
		{"⏎", "select"},
		{"/", "search"},
		{"?", "help"},
	}
	var parts []string
	for _, h := range hints {
		k := c.theme.Primary.Render(h.key)
		d := c.theme.Muted.Render(" " + h.desc)
		parts = append(parts, k+d)
	}
	hintLine := strings.Join(parts, c.theme.Muted.Render("   "))
	b.WriteString(centerLine(hintLine, width))

	return b.String()
}

// renderFooter builds the 1-line preview bar showing key = value for the focused field.
func (c *content) renderFooter(width int) string {
	f := c.currentField()
	if f == nil {
		return c.theme.Muted.Render(strings.Repeat("─", width))
	}

	key := f.Key
	val := f.Default
	if v, ok := c.values[f.Key]; ok {
		val = v
	}

	// live editor preview override
	if c.detail.editing && c.detail.editor != nil {
		switch e := c.detail.editor.(type) {
		case *stylestringEditor:
			val = e.PreviewValue()
		case *colorEditor:
			val = e.PreviewValue()
		default:
			val = c.detail.editor.Value()
		}
	}

	// type-aware value rendering
	sep := c.theme.Muted.Render("─ ")
	keyStr := c.theme.PreviewHL.Render(key)
	eq := c.theme.Muted.Render(" = ")
	var valStr string

	switch {
	case f.Widget == "stylestring":
		sym, sty := parseStyleString(val)
		if sty != "" {
			valStr = c.theme.Primary.Render("[") +
				c.theme.Accent.Render(sym) +
				c.theme.Primary.Render("](") +
				c.theme.Muted.Render(sty) +
				c.theme.Primary.Render(")")
		} else {
			valStr = c.theme.Accent.Render(val)
		}
	case f.Type == "color":
		hex := normalizeHex(val)
		if c.detail.editing {
			if ce, ok := c.detail.editor.(*colorEditor); ok {
				hex = normalizeHex(ce.PreviewValue())
			}
		}
		if hex != "" {
			valStr = swatch(hex) + " " + c.theme.Accent.Render(strings.TrimPrefix(hex, "#"))
		} else {
			valStr = c.theme.Muted.Render("not set")
		}
	case f.Type == "bool":
		if val == "true" {
			valStr = c.theme.Success.Render("●") + " " + c.theme.Accent.Render("true")
		} else {
			valStr = c.theme.Muted.Render("○") + " " + c.theme.Accent.Render("false")
		}
	default:
		valStr = c.theme.Accent.Render(val)
	}

	line := sep + keyStr + eq + valStr

	// truncate to width
	if lipgloss.Width(line) > width {
		// rough truncation: re-render with shortened val
		maxVal := width - lipgloss.Width(sep+keyStr+eq) - 1
		if maxVal > 0 && len(val) > maxVal {
			val = val[:maxVal] + "…"
			valStr = c.theme.Accent.Render(val)
			line = sep + keyStr + eq + valStr
		}
	}

	return line
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
	bodyH := c.height - logoBlockH - footerH
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
		if c.detail.editing && c.detail.editor != nil {
			if _, ok := c.detail.editor.(InlineEditor); !ok {
				// for list/hook editors, track the active cursor, not the editor bottom
				if le, ok := c.detail.editor.(*listEditor); ok {
					cursorBottom += le.cursorOffset() + 1
				} else if he, ok := c.detail.editor.(*hookEditor); ok {
					cursorBottom += he.cursorOffset() + 1
				} else if se, ok := c.detail.editor.(*structListEditor); ok {
					cursorBottom += se.cursorOffset() + 1
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
		footerStr := c.renderFooter(innerW)
		return outerStyle.Render(headerStr + crumbStr + fieldView + "\n" + footerStr)
	}

	detailContentW := detailW - 3
	if detailContentW < 10 {
		detailContentW = 10
	}

	if wide {
		// wide layout: detail spans full height, header lives in left column
		footerStr := c.renderFooter(fieldListW)
		leftContent := headerStr + crumbStr + fieldView + "\n" + footerStr
		leftLines := strings.Count(leftContent, "\n") + 1
		for leftLines < c.height {
			leftContent += "\n"
			leftLines++
		}

		detailView := c.detail.View(detailContentW, c.height)

		detailStyled := c.theme.Detail.
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

	detailStyled := c.theme.Detail.
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
	footerStr := c.renderFooter(fieldListW)

	return outerStyle.Render(headerStr + crumbStr + bodyRow + "\n" + footerStr)
}

// renderBody produces the scrollable field area: search + field rows.
// header and no-schema states are handled in View.
func (c *content) renderBody(width int) string {
	var b strings.Builder

	// search bar (when active or has locked query)
	if c.searching || len(c.searchMatches) > 0 {
		prompt := c.theme.Primary.Render("/ ")
		var countStr string
		if len(c.searchMatches) > 0 {
			countStr = c.theme.Muted.Render(fmt.Sprintf("  %d/%d matches", c.searchIdx+1, len(c.searchMatches)))
		} else if c.searching {
			countStr = c.theme.Muted.Render(fmt.Sprintf("  %d/%d fields", len(c.visible), len(c.fields)))
		}
		if c.searching {
			b.WriteString(prompt + c.search.View() + countStr)
		} else {
			// locked search: show query text as static
			b.WriteString(prompt + c.theme.Subtext.Render(c.search.Value()) + countStr)
		}
		b.WriteByte('\n')
	}

	// filter indicator (when not searching)
	if c.filterIndicatorVisible() {
		var label string
		switch {
		case c.bookmarkedOnly:
			label = "bookmarks"
		case c.showEffective:
			label = "effective (all with defaults)"
		case c.showNewOnly:
			label = "new only"
		default:
			label = "configured only"
		}
		b.WriteString(c.theme.Warning.Render("▸ " + label))
		b.WriteByte('\n')
	}

	// detect inline editing state once before the loop
	editingInline := c.detail.editing && c.detail.editor != nil

	// rotating section colors for visual distinction
	sectionColors := []lipgloss.Style{
		c.theme.Primary, c.theme.Secondary, c.theme.Accent,
		c.theme.Success, c.theme.Warning,
	}

	// hoist per-field constants outside the loop
	icons := fieldIcons(c.nerdFont)

	for i, r := range c.visible {
		// section header row
		if r.isSection {
			name := c.schema.Sections[r.sectionIdx].Name
			sc := sectionColors[r.sectionIdx%len(sectionColors)]
			indicator := "▾ "
			if c.collapsed[r.sectionIdx] {
				indicator = "▸ "
			}
			isCursor := c.focused && i == c.cursor
			prefix := "── "
			if isCursor {
				prefix = sc.Render("▎ ")
			}
			header := sc.Bold(true).Render(prefix + indicator + name + " ")
			remaining := width - lipgloss.Width(header)
			if remaining > 0 {
				header += sc.Faint(true).Render(strings.Repeat("─", remaining))
			}
			// breathing room before sections (except the first visible row)
			if i > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(header)
			b.WriteByte('\n')
			continue
		}

		f := &c.fields[r.fieldIdx]
		isCursor := c.focused && i == c.cursor

		// is this row the one being edited?
		isEditRow := editingInline && isCursor && r.fieldIdx == c.detail.editField

		// type icon (widget hint takes precedence)
		icon := icons[f.Widget]
		if icon == "" {
			icon = icons[f.Type]
		}
		if icon == "" {
			icon = " "
		}

		// single map lookup for current value
		val, hasVal := c.values[f.Key]

		// configured indicator (only green when value differs from default)
		isConfigured := hasVal && val != f.Default
		var dot string
		if isConfigured {
			dot = c.theme.Success.Render("●")
		} else {
			dot = c.theme.Muted.Render("○")
		}

		// value rendering
		var renderedVal string
		if !hasVal {
			val = f.Default
			if c.showEffective && val != "" {
				renderedVal = c.theme.Muted.Italic(true).Render(val + " (default)")
			} else {
				renderedVal = c.renderFieldValue(*f, val, true)
			}
		} else {
			renderedVal = c.renderFieldValue(*f, val, false)
		}

		// inline editor: replace value portion with InlineView or live preview
		if isEditRow {
			switch e := c.detail.editor.(type) {
			case InlineEditor:
				renderedVal = e.InlineView(width / 2)
			case *colorEditor:
				newHex := normalizeHex(e.PreviewValue())
				renderedVal = swatch(e.oldHex) +
					c.theme.Muted.Render(" → ") +
					swatch(newHex) +
					" " + c.theme.FieldValue.Render(newHex)
			case *stylestringEditor:
				renderedVal = c.theme.Accent.Render(e.PreviewValue())
			}
		}

		// inline min/max bounds for number fields (skipped for slider widgets and inline-editing)
		showBounds := f.Type == "number" && f.Widget != "slider" && (f.Min != nil || f.Max != nil) && !isEditRow

		// build prefix and label (cursor/icon)
		paddedLabel := c.paddedLabels[r.fieldIdx]
		iconStyle := c.typeIconStyle(f.Type, val)
		isBookmarked := c.konfable != nil && c.bookmarks[c.konfable.Name()+"/"+f.Key]
		var prefix, label string
		if isCursor {
			prefix = c.theme.Primary.Render("▎ ") + iconStyle.Render(icon) + " "
			label = c.theme.Text.Bold(true).Render(paddedLabel)
		} else {
			prefix = "  " + iconStyle.Faint(true).Render(icon) + " "
			label = c.theme.FieldLabel.Render(paddedLabel)
		}
		if isBookmarked {
			label = c.theme.Warning.Render("★") + label
		}

		if showBounds {
			lo, hi := "*", "*"
			if f.Min != nil {
				lo = formatNum(*f.Min)
			}
			if f.Max != nil {
				hi = formatNum(*f.Max)
			}
			boundsStr := fmt.Sprintf(" (%s\u2013%s)", lo, hi)
			usedW := lipgloss.Width(prefix) + lipgloss.Width(label) + 2 + lipgloss.Width(renderedVal)
			if usedW+len(boundsStr) <= width {
				renderedVal += c.theme.Muted.Render(boundsStr)
			}
		}

		line := prefix + label + " " + dot + " " + renderedVal

		// truncate value with ellipsis if line exceeds available width (skip for inline editors)
		if lipgloss.Width(line) > width && !isEditRow {
			// re-render with truncated value
			usedW := lipgloss.Width(prefix) + lipgloss.Width(label) + lipgloss.Width(" "+dot+" ")
			maxValW := width - usedW - 1
			if maxValW > 0 {
				valPlain := val
				if !hasVal {
					valPlain = f.Default
				}
				if len(valPlain) > maxValW {
					valPlain = valPlain[:maxValW-1] + "…"
				}
				if !hasVal {
					renderedVal = c.renderFieldValue(*f, valPlain, true)
				} else {
					renderedVal = c.renderFieldValue(*f, valPlain, false)
				}
				line = prefix + label + " " + dot + " " + renderedVal
			}
		}

		// search match explanation
		if info, ok := c.searchMatchInfo[i]; ok {
			usedW := lipgloss.Width(line)
			infoStr := c.theme.Muted.Italic(true).Render("  " + info)
			if usedW+lipgloss.Width(infoStr) <= width {
				line += infoStr
			}
		}

		b.WriteString(line)
		b.WriteByte('\n')

		// expanded editor: render below cursor row for non-inline editors
		if isEditRow {
			if _, ok := c.detail.editor.(InlineEditor); !ok {
				editorView := c.detail.editor.View(width)
				b.WriteString(editorView)
				b.WriteByte('\n')
			}
		}
	}

	return b.String()
}

// renderFieldValue renders a field value with type-specific formatting.
func (c *content) renderFieldValue(f pkg.Field, val string, isDefault bool) string {
	// stylestring rendering (widget takes priority)
	if f.Widget == "stylestring" {
		sym, sty := parseStyleString(val)
		if sty != "" {
			style := c.theme.FieldDefault
			if !isDefault {
				style = c.theme.FieldValue
			}
			return c.theme.Primary.Render("[") +
				style.Render(sym) +
				c.theme.Primary.Render("](") +
				c.theme.Accent.Render(sty) +
				c.theme.Primary.Render(")")
		}
	}

	if isDefault {
		switch f.Type {
		case "bool":
			if val == "true" {
				return c.theme.FieldDefault.Render("● true")
			}
			return c.theme.FieldDefault.Render("○ false")
		case "color":
			hex := normalizeHex(val)
			if hex == "" {
				return c.theme.FieldDefault.Render("not set")
			}
			return swatch(hex) + " " + c.theme.FieldDefault.Render(val)
		default:
			return c.theme.FieldDefault.Render(val)
		}
	}

	switch f.Type {
	case "bool":
		if val == "true" {
			return c.theme.Success.Render("●") + " " + c.theme.FieldValue.Render("true")
		}
		return c.theme.Muted.Render("○") + " " + c.theme.FieldValue.Render("false")
	case "color":
		hex := normalizeHex(val)
		if hex == "" {
			return c.theme.Muted.Render("not set")
		}
		return swatch(hex) + " " + c.theme.FieldValue.Render(val)
	default:
		return c.theme.FieldValue.Render(val)
	}
}

// typeIconStyle returns a per-type color for field type icons.
// mirrors the type badge colors in detail.go for visual consistency.
// for color fields, colorHex tints the icon with the actual field value.
func (c *content) typeIconStyle(typ, colorHex string) lipgloss.Style {
	switch typ {
	case "number":
		return c.theme.Secondary
	case "enum":
		return c.theme.Primary
	case "color":
		hex := normalizeHex(colorHex)
		if hex != "" {
			return lipgloss.NewStyle().Foreground(lipgloss.Color(hex))
		}
		return c.theme.Accent
	case "bool":
		return c.theme.Success
	case "list", "multi":
		return c.theme.Warning
	default:
		return c.theme.Muted
	}
}

// centerBlock centers each line of a multi-line string within the given width.
func centerBlock(block string, width int) string {
	lines := strings.Split(block, "\n")
	for i, line := range lines {
		lines[i] = centerLine(line, width)
	}
	return strings.Join(lines, "\n")
}

// dashboardStats formats the actionable stats suffix for a dashboard app.
// configured count is omitted — sort order communicates engagement.
func (c *content) dashboardStats(a dashboardApp) string {
	var parts []string
	if a.newCount > 0 {
		parts = append(parts, fmt.Sprintf("%d new", a.newCount))
	}
	if a.deprecatedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d deprecated", a.deprecatedCount))
	}
	if a.coverage != "" && a.coverage != "full" {
		parts = append(parts, a.coverage)
	}
	if len(parts) == 0 {
		return ""
	}
	return c.theme.Muted.Render("  " + strings.Join(parts, " · "))
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
