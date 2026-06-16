package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/getkonfi/konfi/pkg"
	"github.com/getkonfi/konfi/theme"
	"github.com/getkonfi/konfi/ui/editors"
)

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
		hex := theme.ColorRenderHex(colorHex)
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
			link := d.theme.FieldDocLink.Hyperlink(d.docsURL).Render("open docs")
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
		if d.editor != nil {
			if pv, ok := d.editor.(editors.Previewer); ok {
				colorHex = pv.PreviewValue()
			}
		}
	}
	badgeStyle := d.typeBadgeStyle(f.Type, colorHex)
	if f.Until != "" {
		badgeStyle = d.badgeBase.
			Background(d.theme.Palette.Error).
			Foreground(d.theme.Palette.Base).
			Strikethrough(true)
	}
	b.WriteString(badgeStyle.Render(icon + " " + fieldBadgeName(*f)))

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
		b.WriteString(" " + d.theme.Muted.Render("since "+f.Since))
	}
	if f.Until != "" {
		b.WriteString(" " + d.theme.FieldWarn.Render("until "+f.Until))
	}
	b.WriteByte('\n')

	// field label
	labelStyle := d.theme.Text.Bold(true)
	if f.Until != "" {
		labelStyle = labelStyle.Foreground(d.theme.Palette.Error).Faint(true).Strikethrough(true)
	}
	b.WriteString(labelStyle.Render(f.Label))
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
		style := d.theme.Muted.Italic(true)
		if f.Until != "" {
			style = d.theme.FieldWarn.Italic(true)
		}
		val := style.Render(f.Hint)
		b.WriteString(val)
		b.WriteByte('\n')
	}

	// doc link — OSC 8 clickable hyperlink
	docURL := f.DocURL
	if docURL == "" {
		docURL = d.docsURL
	}
	if docURL != "" {
		linkStyle := d.theme.FieldDocLink.Hyperlink(docURL)
		b.WriteString(linkStyle.Render("docs ↗"))
		b.WriteByte('\n')
	}

	// file snippet (generous — 12 lines context)
	if snippet := d.renderFileSnippet(width, 12); snippet != "" {
		b.WriteByte('\n')
		b.WriteString(d.detailSeparator(width))
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

func (d *detail) detailSeparator(width int) string {
	if width < 1 {
		width = 1
	}
	return d.theme.FaintSeparator.Render(strings.Repeat("─", width))
}

// renderTypeVisual returns type-aware visuals for the current field value.
func (d *detail) renderTypeVisual(f *pkg.Field, width int) string {
	val := f.Default
	if v, ok := d.values[f.Key]; ok {
		val = v
	}

	if f.Widget == "stylestring" {
		if d.editor != nil {
			if pv, ok := d.editor.(editors.Previewer); ok {
				val = pv.PreviewValue()
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
		if d.editor != nil {
			if pv, ok := d.editor.(editors.Previewer); ok {
				colorVal = pv.PreviewValue()
			}
		}
		display := theme.ColorDisplayValue(colorVal)
		if display == "" {
			return ""
		}
		colorStyle := d.theme.FieldValue
		if hex := theme.ColorRenderHex(colorVal); hex != "" {
			colorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(hex))
		}
		return theme.ColorValue(colorVal, d.theme.Palette.BaseHex()) + " " + colorStyle.Render(f.Key+" = "+display)

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
	sym, sty := theme.ParseStyleString(val)
	if sty == "" {
		return d.theme.Text.Bold(true).Render(val)
	}
	symPill := d.theme.Badge.Render(sym)
	styPill := d.theme.Accent.Render(sty)
	return symPill + " " + styPill
}
