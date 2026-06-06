package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/eminert/konfi/pkg"
	"github.com/eminert/konfi/theme"
	"github.com/eminert/konfi/ui/editors"
)

// field type icons — nerd font glyphs
var fieldTypeIconNerd = map[string]string{
	"string": "\uf031",
	"number": "\uf292",
	"bool":   "\uf205",
	"enum":   "\uf150",
	"color":  "\uf1fc",
	"list":   "\uf03a",
	"multi":  "\uf046",

	"font":        "\uf031",
	"slider":      "\U000F1A8A",
	"path":        "\uf115",
	"stylestring": "\uf0d0",
	"hook":        "\uf0e7",
	"structlist":  "\uf00b",
	"patternlist": "\uf03a",
	"togglemap":   "\uf205",
}

// field type icons — plain ASCII fallback
var fieldTypeIconASCII = map[string]string{
	"string": "Aa",
	"number": "#",
	"bool":   "<>",
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
	"togglemap":   "<>",
}

// fieldIcons returns the nerd or ASCII icon map based on the flag.
func fieldIcons(nerd bool) map[string]string {
	if nerd {
		return fieldTypeIconNerd
	}
	return fieldTypeIconASCII
}

// widgetBadgeLabels overrides the badge text for widgets whose underlying
// type reads wrong — e.g. a stylestring symbol picker has type "string" but
// is really a pick-from-options select, not free text.
var widgetBadgeLabels = map[string]string{
	"stylestring": "select",
}

// fieldBadgeName returns the label shown in a field's type badge. when a widget
// is set the badge follows the widget (the icon already does), since the raw
// type is often misleading for widget fields.
func fieldBadgeName(f pkg.Field) string {
	if f.Widget != "" {
		if name, ok := widgetBadgeLabels[f.Widget]; ok {
			return name
		}
		return f.Widget
	}
	return f.Type
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
		var labels []string
		if c.bookmarkedOnly {
			labels = append(labels, "bookmarks")
		}
		if c.showEffective {
			labels = append(labels, "effective")
		}
		if c.showNewOnly {
			labels = append(labels, "new")
		}
		if c.changedOnly {
			labels = append(labels, "changed")
		}
		if c.configuredOnly {
			labels = append(labels, "configured")
		}
		label := strings.Join(labels, " + ")
		b.WriteString(c.theme.Warning.Render("▸ " + label))
		b.WriteByte('\n')
	}

	// detect inline editing state once before the loop
	editingInline := c.detail.editor != nil

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
			isCursor := c.fieldListFocused() && i == c.cursor
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
		isCursor := c.fieldListFocused() && i == c.cursor

		// is this row the one being edited?
		isEditRow := editingInline && isCursor && r.fieldIdx == c.detail.editField
		// changed-only (tab) view renders the value as an old → new diff
		isDiffRow := false

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

		// configured indicator: green when the key is present in the config file,
		// even if its value matches the default (consistent with the configured-only filter)
		isConfigured := hasVal
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
			switch ed := c.detail.editor.(type) {
			case editors.InlineEditor:
				renderedVal = ed.InlineView(width / 2)
			case editors.Previewer:
				switch {
				case f.Type == "color":
					bg := c.theme.Palette.BaseHex()
					renderedVal = theme.ColorValue(c.detail.editOrigVal, bg) +
						c.theme.Muted.Render(" → ") +
						theme.ColorValue(ed.PreviewValue(), bg)
				case f.Widget == "stylestring":
					renderedVal = c.theme.Accent.Render(ed.PreviewValue())
				}
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

		// changed-only view: replace the value with an old → new diff
		if c.changedOnly && !isEditRow {
			oldVal, hadOld := c.origValues[f.Key]
			newVal, hasNew := c.values[f.Key]
			if (hadOld || hasNew) && (hadOld != hasNew || oldVal != newVal) {
				usedW := lipgloss.Width(prefix) + lipgloss.Width(label) + lipgloss.Width(" "+dot+" ")
				renderedVal = c.renderInlineDiff(oldVal, hadOld, newVal, hasNew, width-usedW-1)
				isDiffRow = true
			}
		}

		if showBounds && !isDiffRow {
			lo, hi := "*", "*"
			if f.Min != nil {
				lo = theme.FormatNum(*f.Min)
			}
			if f.Max != nil {
				hi = theme.FormatNum(*f.Max)
			}
			boundsStr := fmt.Sprintf(" (%s\u2013%s)", lo, hi)
			usedW := lipgloss.Width(prefix) + lipgloss.Width(label) + 2 + lipgloss.Width(renderedVal)
			if usedW+len(boundsStr) <= width {
				renderedVal += c.theme.Muted.Render(boundsStr)
			}
		}

		line := prefix + label + " " + dot + " " + renderedVal

		// truncate value with ellipsis if line exceeds available width (skip for inline editors and diffs)
		if lipgloss.Width(line) > width && !isEditRow && !isDiffRow {
			// re-render with truncated value
			usedW := lipgloss.Width(prefix) + lipgloss.Width(label) + lipgloss.Width(" "+dot+" ")
			maxValW := width - usedW - 1
			if maxValW > 0 {
				valPlain := val
				if !hasVal {
					valPlain = f.Default
				}
				if len(valPlain) > maxValW {
					valPlain = theme.Truncate(valPlain, maxValW)
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
			if _, ok := c.detail.editor.(editors.InlineEditor); !ok {
				editorView := c.detail.editor.View(width)
				b.WriteString(editorView)
				b.WriteByte('\n')
			}
		}
	}

	return b.String()
}

// renderInlineDiff renders a changed field's value as "old → new" with
// word-level highlighting, fit within maxW total display cells. ∅ marks a value
// that was absent (newly set) or removed.
func (c *content) renderInlineDiff(oldVal string, hadOld bool, newVal string, hasNew bool, maxW int) string {
	th := c.theme
	if maxW < 8 {
		maxW = 8
	}
	arrow := th.Muted.Render(" → ")

	switch {
	case !hadOld:
		return th.Muted.Render("∅") + arrow + th.Success.Render(theme.Truncate(newVal, maxW-3))
	case !hasNew:
		return th.Error.Render(theme.Truncate(oldVal, maxW-3)) + arrow + th.Muted.Render("∅")
	default:
		side := (maxW - 3) / 2 // split remaining width across both sides of the arrow
		if side < 4 {
			side = 4
		}
		ot := theme.Truncate(oldVal, side)
		nt := theme.Truncate(newVal, side)
		return renderWordDiff(ot, nt, diffRemoved, th) + arrow + renderWordDiff(nt, ot, diffAdded, th)
	}
}

// singleLine flattens a value for one-row display: newlines and tabs become
// visible escapes so a multi-line value (e.g. a collapsed TOML """ string)
// can't break row alignment.
func singleLine(s string) string {
	if !strings.ContainsAny(s, "\n\r\t") {
		return s
	}
	return strings.NewReplacer("\r\n", "\\n", "\n", "\\n", "\r", "\\n", "\t", "\\t").Replace(s)
}

// renderFieldValue renders a field value with type-specific formatting.
func (c *content) renderFieldValue(f pkg.Field, val string, isDefault bool) string {
	val = singleLine(val)
	// stylestring rendering (widget takes priority)
	if f.Widget == "stylestring" {
		sym, sty := theme.ParseStyleString(val)
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
			return c.theme.FieldDefault.Render(val)
		case "color":
			if theme.ColorDisplayValue(val) == "" {
				return c.theme.FieldDefault.Render("not set")
			}
			return theme.ColorValue(val, c.theme.Palette.BaseHex())
		default:
			return c.theme.FieldDefault.Render(val)
		}
	}

	switch f.Type {
	case "bool":
		return c.theme.FieldValue.Render(val)
	case "color":
		if theme.ColorDisplayValue(val) == "" {
			return c.theme.Muted.Render("not set")
		}
		return theme.ColorValue(val, c.theme.Palette.BaseHex())
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
		hex := theme.ColorRenderHex(colorHex)
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
