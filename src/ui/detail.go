package ui

import (
	"fmt"
	"strings"

	"github.com/emin/konfigurator/konfables"
	"github.com/emin/konfigurator/pkg"
	"github.com/emin/konfigurator/theme"

	"charm.land/lipgloss/v2"
)

// detail is a sub-model owned by content that renders the preview/detail pane.
type detail struct {
	previewLine  int
	previewFound bool
	previewKey   string
	docsURL      string
	theme        *theme.Theme

	// editor state (moved from content in M5)
	editing     bool
	editor      FieldEditor
	editField   int    // index into fields slice
	editOrigVal string // for cancel restoration

	// scroll state for browse mode
	scrollY int

	// context synced from content on state changes
	field    *pkg.Field
	config   *pkg.ConfigFile
	konfable konfables.Konfable
	values   map[string]string
	focused  bool
}

func newDetail(th *theme.Theme) detail {
	return detail{
		previewLine: -1,
		theme:       th,
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
	d.docsURL = ""
	d.scrollY = 0
	d.field = nil
	d.config = nil
	d.konfable = nil
	d.values = nil
	d.focused = false
}

// forceRescan clears the cached key so the next sync re-scans the config.
func (d *detail) forceRescan() {
	d.previewKey = ""
}

// refreshPreviewLine updates the preview line from config for the current field.
func (d *detail) refreshPreviewLine() {
	f := d.field
	if f == nil || d.config == nil || d.konfable == nil || d.konfable.Parser() == nil {
		d.previewLine = -1
		d.previewFound = false
		d.previewKey = ""
		return
	}
	if f.Key == d.previewKey {
		return
	}
	d.previewKey = f.Key
	d.previewLine, d.previewFound = d.konfable.Parser().FindLine(d.config.Content(), f.Key)
}

// renderMarkdown renders markdown using the goldmark-based renderer in markdown.go.
func (d detail) renderMarkdown(md string, width int) string {
	return RenderMarkdown(md, width, d.theme)
}

// View renders the detail pane content — always browse mode.
// editing is handled inline in the field list (content.renderBody).
func (d detail) View(width, height int) string {
	return d.viewBrowse(width, height)
}

// viewBrowse renders the structured detail panel in browse mode.
// all sections are rendered unconditionally, then scrolled into the viewport.
func (d detail) viewBrowse(width, height int) string {
	if d.config == nil {
		return d.theme.Muted.Render("no preview")
	}

	f := d.field
	var b strings.Builder

	if !d.focused || f == nil {
		pathDisplay := d.config.Path
		if pathDisplay == "" && d.konfable != nil {
			pathDisplay = d.konfable.Info().Name
		}
		b.WriteString(d.theme.Subtext.Render(pathDisplay))
		b.WriteByte('\n')
		if d.docsURL != "" {
			key := d.theme.Badge.Render(" o ")
			label := d.theme.Subtext.Render(" open docs")
			b.WriteString(key + label)
		}
		return b.String()
	}

	// type badge
	icon := fieldTypeIcon[f.Widget]
	if icon == "" {
		icon = fieldTypeIcon[f.Type]
	}
	if icon == "" {
		icon = " "
	}
	b.WriteString(d.theme.Badge.Render(icon + " " + f.Type))
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

	// doc link
	hasDoc := f.DocURL != "" || d.docsURL != ""
	if hasDoc {
		key := d.theme.Badge.Render("o")
		label := d.theme.Subtext.Render(" open doc")
		b.WriteString(key + label)
		b.WriteByte('\n')
	}

	// live config line (skip for color — already shown in type visual)
	if f.Type != "color" {
		val := f.Default
		if v, ok := d.values[f.Key]; ok {
			val = v
		}
		keyStr := f.Key
		sep := " = "
		usedW := len(keyStr) + len(sep)
		if len(val)+usedW > width && width > usedW+1 {
			val = val[:width-usedW-1] + "…"
		}
		b.WriteByte('\n')
		b.WriteString(d.theme.PreviewHL.Render(keyStr) + d.theme.Text.Render(sep) + d.theme.Accent.Render(val))
		b.WriteByte('\n')
	}

	// file snippet (generous — 12 lines context)
	b.WriteByte('\n')
	b.WriteString(d.renderFileSnippet(width, 12))

	// apply scroll + viewport clipping
	full := b.String()
	lines := strings.Split(full, "\n")
	if d.scrollY > len(lines)-height {
		d.scrollY = max(0, len(lines)-height)
	}
	if d.scrollY > 0 {
		lines = lines[d.scrollY:]
	}
	if len(lines) > height {
		lines = lines[:height]
	}

	// scroll indicator
	if d.scrollY > 0 || len(strings.Split(full, "\n")) > height {
		indicator := d.theme.Muted.Render("↕ scroll")
		if len(lines) > 0 {
			lines[len(lines)-1] = indicator
		}
	}

	return strings.Join(lines, "\n")
}

// renderTypeVisual returns type-aware visuals for the current field value.
func (d detail) renderTypeVisual(f *pkg.Field, width int) string {
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
		hex := normalizeHex(val)
		if d.editing {
			if ce, ok := d.editor.(*colorEditor); ok {
				hex = normalizeHex(ce.PreviewValue())
			}
		}
		colorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(hex))
		return swatch(hex) + swatch(hex) + " " + colorStyle.Render(f.Key+" = "+strings.TrimPrefix(hex, "#"))

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
func (d detail) renderRangeBar(f *pkg.Field, val string, width int) string {
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
func (d detail) renderEnumPills(f *pkg.Field, val string) string {
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
func (d detail) renderStylestringPreview(val string) string {
	sym, sty := parseStyleString(val)
	if sty == "" {
		return d.theme.Text.Bold(true).Render(val)
	}
	symPill := d.theme.Badge.Render(sym)
	styPill := d.theme.Accent.Render(sty)
	return symPill + " " + styPill
}

// renderFileSnippet renders the config file snippet centered on the field's line.
func (d detail) renderFileSnippet(width, height int) string {
	if d.config == nil {
		return ""
	}

	if !d.previewFound {
		val := ""
		if d.field != nil {
			val = d.field.Default
			if v, ok := d.values[d.field.Key]; ok {
				val = v
			}
		}
		if d.field != nil {
			return d.theme.Success.Render(fmt.Sprintf("+ %s = %s", d.field.Key, val))
		}
		return ""
	}

	data := d.config.Content()
	rawLines := strings.Split(string(data), "\n")

	startLine := d.previewLine - height/2
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
		line := rawLines[i]
		maxW := width - 2
		if lipgloss.Width(line) > maxW {
			line = line[:maxW]
		}

		if i == d.previewLine {
			b.WriteString(d.theme.PreviewHL.Render("▶ " + line))
		} else {
			b.WriteString(d.theme.Muted.Render("  " + line))
		}
		if i < endLine-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
