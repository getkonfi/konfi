package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/eminert/konfi/konfables"
	"github.com/eminert/konfi/theme"
	"github.com/eminert/konfi/ui/widgets"
)

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
	return d.theme.Subtext.Render(theme.Truncate(path, width))
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
	return d.theme.Muted.Render(theme.Truncate(label, width))
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
	line = theme.Truncate(line, maxW)
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
	oldText = theme.Truncate(oldText, valW)
	newText = theme.Truncate(newText, valW)

	minus = d.theme.Error.Render("- ") +
		d.theme.Error.Faint(true).Render(gutter) +
		d.theme.Error.Render(keyPart) +
		widgets.RenderWordDiff(oldText, newText, widgets.DiffRemoved, d.theme)
	plus = d.theme.Success.Render("+ ") +
		d.theme.Success.Faint(true).Render(gutter) +
		d.theme.Success.Render(keyPart) +
		widgets.RenderWordDiff(newText, oldText, widgets.DiffAdded, d.theme)
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
	line = theme.Truncate(line, maxW)

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

	format := ""
	if d.konfable != nil {
		format = d.konfable.Info().Format
	}
	updated, err := konfables.WriteField(p, data, *d.field, value, format)
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
	case "hook", "togglemap", "structlist", "blocklist":
		return value
	}
	format := ""
	if d.konfable != nil {
		format = d.konfable.Info().Format
	}
	return konfables.FormatValue(value, d.field.Type, format)
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
