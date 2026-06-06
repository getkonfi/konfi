package ui

import (
	"fmt"
	"strings"

	"github.com/eminert/konfi/konfables"
	"github.com/eminert/konfi/pkg"
	"github.com/eminert/konfi/theme"

	"charm.land/lipgloss/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

// detail is a sub-model owned by content that renders the preview/detail pane.
type detail struct {
	previewLine  int
	previewFound bool
	previewKey   string
	previewGen   uint64
	docsURL      string
	theme        *theme.Theme

	// editor state (moved from content in M5)
	editing     bool
	editor      FieldEditor
	editField   int    // index into fields slice
	editOrigVal string // for cancel restoration

	// scroll state for browse mode
	scrollY int

	// nerd font glyphs or ASCII fallback
	nerdFont bool

	// context synced from content on state changes
	field      *pkg.Field
	config     *pkg.ConfigFile
	konfable   konfables.Konfable
	values     map[string]string
	origValues map[string]string // on-disk baseline for inline old→new diff
	focused    bool

	// cached config lines for renderFileSnippet
	snippetLines []string
	snippetGen   uint64 // config generation that produced snippetLines

	// cached styles
	badgeBase lipgloss.Style
	cachedMD  *mdRenderer
	cachedMDW int
}

func newDetail(th *theme.Theme) detail {
	return detail{
		previewLine: -1,
		theme:       th,
		badgeBase:   lipgloss.NewStyle().Bold(true).Padding(0, 1),
	}
}

// sync pushes the latest content state into detail and refreshes the preview line.
func (d *detail) sync(f *pkg.Field, config *pkg.ConfigFile, konfable konfables.Konfable, values map[string]string, focused bool) {
	// reset scroll when field changes
	if f != d.field {
		d.scrollY = 0
	}
	d.field = f
	d.config = config
	d.konfable = konfable
	d.values = values
	d.focused = focused
	d.refreshPreviewLine()
}

// reset clears all detail state on app switch.
func (d *detail) reset() {
	d.previewLine = -1
	d.previewFound = false
	d.previewKey = ""
	d.previewGen = 0
	d.docsURL = ""
	d.scrollY = 0
	d.field = nil
	d.config = nil
	d.konfable = nil
	d.values = nil
	d.origValues = nil
	d.focused = false
	d.snippetLines = nil
}

// forceRescan clears the cached key so the next sync re-scans the config.
func (d *detail) forceRescan() {
	d.previewKey = ""
	d.previewGen = 0
	d.snippetLines = nil
}

// refreshPreviewLine updates the preview line from config for the current field.
func (d *detail) refreshPreviewLine() {
	f := d.field
	if f == nil || d.config == nil || d.konfable == nil || d.konfable.Parser() == nil {
		d.previewLine = -1
		d.previewFound = false
		d.previewKey = ""
		d.previewGen = 0
		return
	}
	gen := d.config.Generation()
	if f.Key == d.previewKey && gen == d.previewGen {
		return
	}
	d.previewKey = f.Key
	d.previewGen = gen
	d.previewLine, d.previewFound = d.konfable.Parser().FindLine(d.config.Content(), f.Key)
}

// renderMarkdown renders markdown using the goldmark-based renderer in markdown.go.
func (d *detail) renderMarkdown(md string, width int) string {
	if d.cachedMD == nil || d.cachedMDW != width {
		d.cachedMD = newMDRenderer(d.theme, width)
		d.cachedMDW = width
	}
	if md == "" {
		return ""
	}
	source := []byte(md)
	p := goldmark.DefaultParser()
	doc := p.Parse(text.NewReader(source))
	return strings.TrimRight(d.cachedMD.render(doc, source), "\n")
}

// View renders the detail pane content — always browse mode.
// editing is handled inline in the field list (content.renderBody).
func (d *detail) View(width, height int) string {
	if d.focused && d.field != nil {
		return d.viewConfigFile(width, height)
	}
	return d.viewBrowse(width, height)
}

func (d *detail) scroll(delta int) {
	d.scrollY += delta
	if d.scrollY < 0 {
		d.scrollY = 0
	}
}

func (d *detail) scrollTop() {
	d.scrollY = 0
}

func (d *detail) scrollBottom() {
	d.scrollY = 1 << 30
}

func (d *detail) centerPreview(viewport int) {
	focusLine := d.configFocusLine()
	if focusLine < 0 {
		d.scrollY = 0
		return
	}
	if viewport < 1 {
		viewport = 1
	}
	d.scrollY = focusLine - viewport/2
	if d.scrollY < 0 {
		d.scrollY = 0
	}
}

func (d *detail) configFocusLine() int {
	if d.previewFound {
		return d.previewLine
	}
	if d.field == nil || d.config == nil || d.konfable == nil || d.konfable.Parser() == nil {
		return -1
	}
	if _, addedLine, ok := d.previewAddedContent(d.config.Content()); ok {
		return addedLine
	}
	if added := d.fallbackAddedLine(); added != "" {
		return len(d.configLines())
	}
	return -1
}

// typeBadgeStyle returns a styled badge for the field type with per-type coloring.
// for color fields, colorHex tints the badge with the actual field value.
func (d *detail) typeBadgeStyle(typ, colorHex string) lipgloss.Style {
	base := d.badgeBase
	switch typ {
	case "number":
		return base.Background(d.theme.Palette.Secondary).Foreground(d.theme.Palette.Base)
	case "enum":
		return base.Background(d.theme.Palette.Primary).Foreground(d.theme.Palette.Base)
	case "color":
		hex := colorRenderHex(colorHex)
		if hex != "" {
			return base.Background(lipgloss.Color(hex)).Foreground(d.theme.Palette.Base)
		}
		return base.Background(d.theme.Palette.Accent).Foreground(d.theme.Palette.Base)
	case "bool":
		return base.Background(d.theme.Palette.Success).Foreground(d.theme.Palette.Base)
	case "list", "multi":
		return base.Background(d.theme.Palette.Warning).Foreground(d.theme.Palette.Base)
	default:
		return d.theme.Badge
	}
}

func (d *detail) viewConfigFile(width, height int) string {
	if d.config == nil {
		return d.theme.Muted.Render("no preview")
	}
	if height <= 0 {
		return ""
	}

	rawLines, focusLine, addedLines := d.configPreviewLines()
	if len(rawLines) == 0 {
		rawLines = []string{""}
	}

	var out []string
	bodyH := height - 1
	if height >= 4 {
		header := d.configFileHeader(width, focusLine)
		if header != "" {
			out = append(out, header)
			bodyH--
		}
	}
	if bodyH < 1 {
		bodyH = 1
	}

	maxScroll := max(0, len(rawLines)-bodyH)
	if d.scrollY > maxScroll {
		d.scrollY = maxScroll
	}
	if d.scrollY < 0 {
		d.scrollY = 0
	}

	end := d.scrollY + bodyH
	if end > len(rawLines) {
		end = len(rawLines)
	}
	for i := d.scrollY; i < end; i++ {
		out = append(out, d.renderConfigFileLine(rawLines[i], i, len(rawLines), i == focusLine, addedLines[i], width))
	}

	if height > 1 {
		out = append(out, d.configScrollIndicator(d.scrollY, end, len(rawLines), focusLine, width))
	}
	return strings.Join(out, "\n")
}

func (d *detail) configPreviewLines() (lines []string, focusLine int, addedLines map[int]bool) {
	if d.config == nil {
		return nil, -1, nil
	}
	if d.previewFound {
		return d.configLines(), d.previewLine, nil
	}
	if d.field == nil || d.konfable == nil || d.konfable.Parser() == nil {
		return d.configLines(), -1, nil
	}

	originalLines := d.configLines()
	updated, addedLine, ok := d.previewAddedContent(d.config.Content())
	if ok {
		updatedLines := strings.Split(string(updated), "\n")
		addedLines := insertedLineSet(originalLines, updatedLines)
		addedLines[addedLine] = true
		return updatedLines, addedLine, addedLines
	}

	added := d.fallbackAddedLine()
	if added == "" {
		return originalLines, -1, nil
	}
	previewLines := append([]string(nil), originalLines...)
	addedLine = len(previewLines)
	previewLines = append(previewLines, added)
	return previewLines, addedLine, map[int]bool{addedLine: true}
}

func (d *detail) configFileHeader(width, focusLine int) string {
	path := d.config.Path
	if path == "" && d.konfable != nil {
		path = d.konfable.Info().Name
	}
	if path == "" {
		path = "configuration"
	}
	if d.field != nil {
		path += " · " + d.field.Key
	}
	if focusLine >= 0 {
		path += fmt.Sprintf(" · line %d", focusLine+1)
	}
	return d.theme.Subtext.Render(truncate(path, width))
}

func (d *detail) configScrollIndicator(start, end, total, focusLine, width int) string {
	if total < 1 {
		total = 1
	}
	if end < start {
		end = start
	}
	label := fmt.Sprintf("↕ %d-%d/%d", start+1, end, total)
	if focusLine >= 0 {
		label += fmt.Sprintf(" · target %d", focusLine+1)
	}
	return d.theme.Muted.Render(truncate(label, width))
}

func (d *detail) renderConfigFileLine(line string, idx, total int, focused, added bool, width int) string {
	gutterW := len(fmt.Sprintf("%d", total))
	if gutterW < 1 {
		gutterW = 1
	}
	maxW := width - gutterW - 3
	if maxW < 1 {
		maxW = 1
	}
	line = truncate(line, maxW)
	gutter := fmt.Sprintf("%*d ", gutterW, idx+1)

	switch {
	case added:
		return d.theme.Success.Render("+ ") +
			d.theme.Success.Faint(true).Render(gutter) +
			d.renderHighlightedConfigLine(line, d.theme.Success)
	case focused:
		return d.theme.Primary.Render("▶ ") +
			d.theme.Muted.Render(gutter) +
			d.renderHighlightedConfigLine(line, d.theme.Text)
	default:
		return d.theme.Muted.Faint(true).Render("  "+gutter) + d.theme.Muted.Render(line)
	}
}

// viewBrowse renders the structured detail panel in browse mode.
// all sections are rendered unconditionally, then scrolled into the viewport.
func (d *detail) viewBrowse(width, height int) string {
	if d.config == nil {
		return d.theme.Muted.Render("no preview")
	}

	f := d.field
	var b strings.Builder

	if f == nil {
		pathDisplay := d.config.Path
		if pathDisplay == "" && d.konfable != nil {
			pathDisplay = d.konfable.Info().Name
		}
		b.WriteString(d.theme.Subtext.Render(pathDisplay))
		b.WriteByte('\n')
		if d.docsURL != "" {
			link := d.theme.Subtext.Hyperlink(d.docsURL).Render("open docs")
			key := d.theme.Badge.Render(" o ")
			b.WriteString(key + " " + link)
		}
		return b.String()
	}

	// type badge — color-coded per type (color fields use actual value)
	icons := fieldIcons(d.nerdFont)
	icon := icons[f.Widget]
	if icon == "" {
		icon = icons[f.Type]
	}
	if icon == "" {
		icon = " "
	}
	colorHex := ""
	if f.Type == "color" {
		colorHex = f.Default
		if v, ok := d.values[f.Key]; ok {
			colorHex = v
		}
		if d.editing {
			if ce, ok := d.editor.(*colorEditor); ok {
				colorHex = ce.PreviewValue()
			}
		}
	}
	badgeStyle := d.typeBadgeStyle(f.Type, colorHex)
	b.WriteString(badgeStyle.Render(icon + " " + f.Type))

	// tier provenance badge
	if d.config != nil {
		if tier := d.config.TierOf(f.Key); tier != "" {
			b.WriteString(" " + d.theme.Muted.Render("["+tier+"]"))
			if tiers := d.config.Tiers(f.Key); len(tiers) > 1 {
				b.WriteString(" " + d.theme.Subtext.Render("← overrides "+tiers[1]))
			}
		}
	}

	// version badges (inline with type badge)
	if f.Since != "" {
		b.WriteString(" " + d.theme.Success.Render("since "+f.Since))
	}
	if f.Until != "" {
		b.WriteString(" " + d.theme.Warning.Render("until "+f.Until))
	}
	b.WriteByte('\n')

	// field label
	b.WriteString(d.theme.Text.Bold(true).Render(f.Label))
	b.WriteByte('\n')
	b.WriteByte('\n')

	// type-aware visuals
	typeVis := d.renderTypeVisual(f, width)
	if typeVis != "" {
		b.WriteString(typeVis)
		b.WriteByte('\n')
	}

	// description
	if f.Description != "" {
		rendered := d.renderMarkdown(f.Description, width)
		b.WriteString(rendered)
		b.WriteByte('\n')
	}

	// example
	if f.Example != "" {
		val := d.theme.Accent.Render(f.Example)
		b.WriteString(val)
		b.WriteByte('\n')
	}

	// hint
	if f.Hint != "" {
		val := d.theme.Muted.Italic(true).Render(f.Hint)
		b.WriteString(val)
		b.WriteByte('\n')
	}

	// doc link — OSC 8 clickable hyperlink
	docURL := f.DocURL
	if docURL == "" {
		docURL = d.docsURL
	}
	if docURL != "" {
		linkStyle := d.theme.Secondary.Underline(true).Hyperlink(docURL)
		b.WriteString(linkStyle.Render("docs ↗"))
		b.WriteByte('\n')
	}

	// file snippet (generous — 12 lines context), fenced off from the
	// explanation above with a labeled rule
	if snippet := d.renderFileSnippet(width, 12); snippet != "" {
		b.WriteByte('\n')
		b.WriteString(d.sectionRule("config", width))
		b.WriteByte('\n')
		b.WriteString(snippet)
	}

	// apply scroll + viewport clipping
	full := b.String()
	lines := strings.Split(full, "\n")
	totalLines := len(lines)
	scrollable := totalLines > height

	// reserve one line for the scroll indicator when content overflows
	clipH := height
	if scrollable {
		clipH = height - 1
	}

	if d.scrollY > totalLines-clipH {
		d.scrollY = max(0, totalLines-clipH)
	}
	if d.scrollY > 0 {
		lines = lines[d.scrollY:]
	}
	if len(lines) > clipH {
		lines = lines[:clipH]
	}

	// append scroll indicator on its own line
	if scrollable {
		lines = append(lines, d.theme.Muted.Render("↕ scroll"))
	}

	return strings.Join(lines, "\n")
}

// sectionRule renders a faint "── label ─────" horizontal divider spanning
// width, used to separate the explanation block from the live config snippet.
func (d *detail) sectionRule(label string, width int) string {
	if width < 8 {
		return d.theme.FaintSeparator.Render(strings.Repeat("─", max(0, width)))
	}
	lead := "── "
	tail := width - lipgloss.Width(lead) - lipgloss.Width(label) - 1
	if tail < 0 {
		tail = 0
	}
	return d.theme.FaintSeparator.Render(lead) +
		d.theme.Muted.Render(label) + " " +
		d.theme.FaintSeparator.Render(strings.Repeat("─", tail))
}

// renderTypeVisual returns type-aware visuals for the current field value.
func (d *detail) renderTypeVisual(f *pkg.Field, width int) string {
	val := f.Default
	if v, ok := d.values[f.Key]; ok {
		val = v
	}

	if f.Widget == "stylestring" {
		if d.editing {
			if se, ok := d.editor.(*stylestringEditor); ok {
				val = se.PreviewValue()
			}
		}
		return d.renderStylestringPreview(val)
	}

	switch f.Type {
	case "color":
		if val == "" {
			return ""
		}
		colorVal := val
		if d.editing {
			if ce, ok := d.editor.(*colorEditor); ok {
				colorVal = ce.PreviewValue()
			}
		}
		display := colorDisplayValue(colorVal)
		if display == "" {
			return ""
		}
		colorStyle := d.theme.FieldValue
		if hex := colorRenderHex(colorVal); hex != "" {
			colorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(hex))
		}
		return colorValue(colorVal) + " " + colorStyle.Render(f.Key+" = "+display)

	case "number":
		if f.Min == nil && f.Max == nil {
			return ""
		}
		return d.renderRangeBar(f, val, width)

	case "enum":
		if len(f.Options) == 0 {
			return ""
		}
		return d.renderEnumPills(f, val)
	}
	return ""
}

// renderRangeBar renders a visual range indicator for number fields.
func (d *detail) renderRangeBar(f *pkg.Field, val string, width int) string {
	lo := 0.0
	hi := 100.0
	if f.Min != nil {
		lo = *f.Min
	}
	if f.Max != nil {
		hi = *f.Max
	}

	var cur float64
	if _, err := fmt.Sscanf(val, "%f", &cur); err != nil {
		cur = lo
	}

	barW := width - 10
	if barW < 5 {
		barW = 5
	}
	pos := int(float64(barW) * (cur - lo) / (hi - lo))
	if pos < 0 {
		pos = 0
	}
	if pos >= barW {
		pos = barW - 1
	}

	bar := strings.Repeat("─", pos) + d.theme.Primary.Render("●") + strings.Repeat("─", barW-pos-1)
	loS := fmt.Sprintf("%.0f", lo)
	hiS := fmt.Sprintf("%.0f", hi)
	return d.theme.Muted.Render(loS+" ") + d.theme.Muted.Render(bar) + d.theme.Muted.Render(" "+hiS)
}

// renderEnumPills renders available options as pills with current value highlighted.
func (d *detail) renderEnumPills(f *pkg.Field, val string) string {
	var parts []string
	for _, opt := range f.Options {
		if opt == val {
			parts = append(parts, d.theme.Badge.Render(opt))
		} else {
			parts = append(parts, d.theme.Muted.Render(opt))
		}
	}
	return strings.Join(parts, " ")
}

// renderStylestringPreview renders a stylestring value as symbol + style pills.
func (d *detail) renderStylestringPreview(val string) string {
	sym, sty := parseStyleString(val)
	if sty == "" {
		return d.theme.Text.Bold(true).Render(val)
	}
	symPill := d.theme.Badge.Render(sym)
	styPill := d.theme.Accent.Render(sty)
	return symPill + " " + styPill
}

// renderFileSnippet renders the config file snippet centered on the field's line.
func (d *detail) renderFileSnippet(width, height int) string {
	if d.config == nil || height <= 0 {
		return ""
	}

	if !d.previewFound {
		return d.renderMissingFileSnippet(width, height)
	}

	rawLines := d.configLines()
	return d.renderConfigSnippet(rawLines, d.previewLine, height, width, nil)
}

func (d *detail) configLines() []string {
	if gen := d.config.Generation(); d.snippetLines == nil || d.snippetGen != gen {
		data := d.config.Content()
		d.snippetLines = strings.Split(string(data), "\n")
		d.snippetGen = gen
	}
	return d.snippetLines
}

func (d *detail) renderMissingFileSnippet(width, height int) string {
	if d.field == nil || d.konfable == nil || d.konfable.Parser() == nil {
		return ""
	}

	original := d.config.Content()
	updated, addedLine, ok := d.previewAddedContent(original)
	if !ok {
		return d.renderBottomWithAddedLine(width, height)
	}

	originalLines := d.configLines()
	updatedLines := strings.Split(string(updated), "\n")
	addedLines := insertedLineSet(originalLines, updatedLines)
	addedLines[addedLine] = true

	return d.renderConfigSnippet(updatedLines, addedLine, height, width, addedLines)
}

func (d *detail) renderConfigSnippet(rawLines []string, focusLine, height, width int, addedLines map[int]bool) string {
	if len(rawLines) == 0 {
		return ""
	}

	startLine := focusLine - height/2
	if startLine < 0 {
		startLine = 0
	}
	endLine := startLine + height
	if endLine > len(rawLines) {
		endLine = len(rawLines)
		startLine = endLine - height
		if startLine < 0 {
			startLine = 0
		}
	}

	var b strings.Builder
	for i := startLine; i < endLine; i++ {
		// the focused line of a modified field renders as a -/+ hunk in place
		if i == focusLine {
			if minus, plus, ok := d.diffHunk(rawLines[i], i, len(rawLines), width); ok {
				b.WriteString(minus)
				b.WriteByte('\n')
				b.WriteString(plus)
				if i < endLine-1 {
					b.WriteByte('\n')
				}
				continue
			}
		}
		b.WriteString(d.renderConfigSnippetLine(rawLines[i], i, len(rawLines), i == focusLine, addedLines[i], width))
		if i < endLine-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// diffHunk renders the focused field's config line as a removed/added pair when
// its value differs from the on-disk baseline: the old value as a red "-" line
// and the current line as a green "+" line, word-highlighting the changed run.
// line is the current (new-value) line. ok is false for unchanged fields or
// when no baseline is available, in which case the caller renders normally.
func (d *detail) diffHunk(line string, idx, total, width int) (minus, plus string, ok bool) {
	if d.field == nil || d.origValues == nil {
		return "", "", false
	}
	oldVal, hadOld := d.origValues[d.field.Key]
	newVal, hasNew := d.values[d.field.Key]
	if !hadOld || !hasNew || oldVal == newVal {
		return "", "", false // only in-place value changes get a -/+ pair
	}

	// locate the value span so key/separator/indent are preserved on both sides
	start, end := findKeySpan(line, d.field.Key)
	if start < 0 {
		return "", "", false
	}
	valStart := findValueStart(line, end)
	if valStart < 0 {
		return "", "", false
	}
	keyPart := line[:valStart]
	newText := line[valStart:]
	oldText := d.serializedFieldValue(oldVal)

	gutterW := len(fmt.Sprintf("%d", total))
	if gutterW < 1 {
		gutterW = 1
	}
	gutter := fmt.Sprintf("%*d ", gutterW, idx+1)
	valW := width - 2 - len(gutter) - lipgloss.Width(keyPart)
	if valW < 4 {
		valW = 4
	}
	oldText = truncate(oldText, valW)
	newText = truncate(newText, valW)

	minus = d.theme.Error.Render("- ") +
		d.theme.Error.Faint(true).Render(gutter) +
		d.theme.Error.Render(keyPart) +
		renderWordDiff(oldText, newText, diffRemoved, d.theme)
	plus = d.theme.Success.Render("+ ") +
		d.theme.Success.Faint(true).Render(gutter) +
		d.theme.Success.Render(keyPart) +
		renderWordDiff(newText, oldText, diffAdded, d.theme)
	return minus, plus, true
}

func (d *detail) renderConfigSnippetLine(line string, idx, total int, focused, added bool, width int) string {
	gutterW := len(fmt.Sprintf("%d", total))
	if gutterW < 1 {
		gutterW = 1
	}
	gutter := fmt.Sprintf("%*d ", gutterW, idx+1)
	maxW := width - 2 - len(gutter)
	if maxW < 1 {
		maxW = 1
	}
	line = truncate(line, maxW)

	switch {
	case added:
		return d.theme.Success.Render("+ ") +
			d.theme.Success.Faint(true).Render(gutter) +
			d.renderHighlightedConfigLine(line, d.theme.Success)
	case focused:
		return d.theme.Primary.Render("▶ ") +
			d.theme.Muted.Render(gutter) +
			d.renderHighlightedConfigLine(line, d.theme.Text)
	default:
		return d.theme.Muted.Faint(true).Render("  "+gutter) +
			d.theme.Muted.Render(line)
	}
}

func (d *detail) renderHighlightedConfigLine(line string, base lipgloss.Style) string {
	if d.field == nil {
		return base.Render(line)
	}

	start, end := findKeySpan(line, d.field.Key)
	if start < 0 {
		return d.theme.PreviewHL.Render(line)
	}

	valueStart := findValueStart(line, end)
	var b strings.Builder
	b.WriteString(base.Render(line[:start]))
	b.WriteString(d.theme.PreviewHL.Render(line[start:end]))
	if valueStart >= 0 {
		b.WriteString(base.Render(line[end:valueStart]))
		b.WriteString(d.theme.Accent.Render(line[valueStart:]))
	} else {
		b.WriteString(base.Render(line[end:]))
	}
	return b.String()
}

func findKeySpan(line, key string) (start, end int) {
	searchEnd := len(line)
	if sep := firstSeparatorIndex(line); sep >= 0 {
		searchEnd = sep
	}
	search := line[:searchEnd]

	for _, candidate := range keyCandidates(key) {
		if candidate == "" {
			continue
		}
		idx := strings.Index(search, candidate)
		if idx < 0 {
			idx = strings.Index(strings.ToLower(search), strings.ToLower(candidate))
		}
		if idx < 0 {
			continue
		}
		start := idx
		end := idx + len(candidate)
		if start > 0 && end < len(line) && line[start-1] == '"' && line[end] == '"' {
			start--
			end++
		}
		return start, end
	}

	return -1, -1
}

func keyCandidates(key string) []string {
	candidates := []string{key}
	if idx := strings.LastIndexByte(key, '.'); idx >= 0 && idx < len(key)-1 {
		candidates = append(candidates, key[idx+1:])
	}
	if idx := strings.LastIndexByte(key, '/'); idx >= 0 && idx < len(key)-1 {
		candidates = append(candidates, key[idx+1:])
	}

	out := candidates[:0]
	seen := make(map[string]bool, len(candidates))
	for _, candidate := range candidates {
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		out = append(out, candidate)
	}
	return out
}

func firstSeparatorIndex(line string) int {
	eq := strings.IndexByte(line, '=')
	colon := strings.IndexByte(line, ':')
	switch {
	case eq >= 0 && colon >= 0:
		return min(eq, colon)
	case eq >= 0:
		return eq
	case colon >= 0:
		return colon
	default:
		return -1
	}
}

func findValueStart(line string, keyEnd int) int {
	if keyEnd < 0 || keyEnd >= len(line) {
		return -1
	}
	start := keyEnd
	for start < len(line) && (line[start] == ' ' || line[start] == '\t') {
		start++
	}
	if start < len(line) && (line[start] == '=' || line[start] == ':') {
		start++
		for start < len(line) && (line[start] == ' ' || line[start] == '\t') {
			start++
		}
		return start
	}
	if start == keyEnd || start >= len(line) {
		return -1
	}
	return start
}

func (d *detail) previewAddedContent(data []byte) (updated []byte, line int, ok bool) {
	p := d.konfable.Parser()
	value := d.fieldValue()

	var err error
	if d.field.Type == "list" {
		if mvp, ok := p.(konfables.MultiValueParser); ok {
			updated, err = mvp.SetValues(data, d.field.Key, splitListValue(value))
		}
	}
	if updated == nil && err == nil {
		updated, err = p.SetValue(data, d.field.Key, d.serializedFieldValue(value))
	}
	if err != nil {
		return nil, -1, false
	}

	line, found := p.FindLine(updated, d.field.Key)
	if !found {
		return nil, -1, false
	}
	return updated, line, true
}

func (d *detail) fieldValue() string {
	if d.field == nil {
		return ""
	}
	if v, ok := d.values[d.field.Key]; ok {
		return v
	}
	return d.field.Default
}

func (d *detail) serializedFieldValue(value string) string {
	if d.field == nil {
		return value
	}
	switch d.field.Widget {
	case "hook", "togglemap", "structlist":
		return value
	}
	format := ""
	if d.konfable != nil {
		format = d.konfable.Info().Format
	}
	return formatValue(value, d.field.Type, format)
}

func (d *detail) renderBottomWithAddedLine(width, height int) string {
	lines := d.configLines()
	if len(lines) == 0 {
		lines = []string{""}
	}
	added := d.fallbackAddedLine()

	tailH := height - 1
	if tailH < 0 {
		tailH = 0
	}
	start := len(lines) - tailH
	if start < 0 {
		start = 0
	}

	total := len(lines)
	if added != "" {
		total++
	}

	var b strings.Builder
	for i := start; i < len(lines); i++ {
		b.WriteString(d.renderConfigSnippetLine(lines[i], i, total, false, false, width))
		if i < len(lines)-1 || added != "" {
			b.WriteByte('\n')
		}
	}
	if added != "" {
		b.WriteString(d.renderConfigSnippetLine(added, len(lines), total, false, true, width))
	}
	return b.String()
}

func (d *detail) fallbackAddedLine() string {
	if d.field == nil {
		return ""
	}
	return fmt.Sprintf("%s = %s", d.field.Key, d.serializedFieldValue(d.fieldValue()))
}

func insertedLineSet(original, updated []string) map[int]bool {
	added := make(map[int]bool)
	oi := 0
	for ui, line := range updated {
		if oi < len(original) && line == original[oi] {
			oi++
			continue
		}
		added[ui] = true
	}
	return added
}
