package ui

import (
	"strconv"
	"strings"
	"sync"

	"github.com/eminert/konfi/pkg"
	"github.com/eminert/konfi/theme"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// paletteGroup marks where a named color group starts in the flat palette slice.
type paletteGroup struct {
	name  string
	start int
}

type colorEditor struct {
	input     textinput.Model
	val       string
	oldHex    string
	th        *theme.Theme
	palette   []string       // hex values
	labels    []string       // display labels (same length as palette)
	groups    []paletteGroup // separator positions for each palette
	palCursor int
	inPalette bool
	cols      int // grid columns, computed on first View
}

func (e *colorEditor) Init(field pkg.Field, currentValue string, th *theme.Theme) tea.Cmd {
	e.th = th
	e.oldHex = normalizeHex(currentValue)
	e.groups = nil

	// build palette: field colors first, then all theme palettes
	e.palette = nil
	e.labels = nil

	// field-specific palette
	for _, hex := range field.Palette {
		e.palette = append(e.palette, hex)
		label := hex
		if len(label) > 7 {
			label = label[:7]
		}
		e.labels = append(e.labels, label)
	}

	// collect colors from all palettes, current theme first
	existing := make(map[string]bool)
	for _, hex := range e.palette {
		existing[normalizeHex(hex)] = true
	}

	// order: current palette first, then the rest
	ordered := []theme.Palette{th.Palette}
	for i := range theme.Palettes {
		if theme.Palettes[i].Name != th.Palette.Name {
			ordered = append(ordered, theme.Palettes[i])
		}
	}

	for i := range ordered {
		pal := &ordered[i]
		var group []theme.PaletteHex
		for _, ph := range pal.Hexes() {
			if !existing[ph.Hex] {
				group = append(group, ph)
				existing[ph.Hex] = true
			}
		}
		if len(group) == 0 {
			continue
		}
		e.groups = append(e.groups, paletteGroup{
			name:  pal.Name,
			start: len(e.palette),
		})
		for _, ph := range group {
			e.palette = append(e.palette, ph.Hex)
			e.labels = append(e.labels, ph.Name)
		}
	}

	e.input = textinput.New()
	e.input.Prompt = "┊ "
	s := textinput.DefaultDarkStyles()
	s.Focused.Prompt = th.Muted
	s.Focused.Text = th.Text
	e.input.SetStyles(s)
	e.input.SetValue(currentValue)
	e.input.CursorEnd()

	// start in palette mode if palette is available
	if len(e.palette) > 0 {
		e.inPalette = true
		currentRenderHex := colorRenderHex(currentValue)
		currentDisplay := normalizeHex(currentValue)
		for i, hex := range e.palette {
			if hex == currentValue || normalizeHex(hex) == currentDisplay ||
				(currentRenderHex != "" && colorRenderHex(hex) == currentRenderHex) {
				e.palCursor = i
				break
			}
		}
		return nil
	}

	return e.input.Focus()
}

func (e *colorEditor) Update(msg tea.Msg) (tea.Cmd, bool, bool) {
	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		if !e.inPalette {
			var cmd tea.Cmd
			e.input, cmd = e.input.Update(msg)
			return cmd, false, false
		}
		return nil, false, false
	}

	if e.inPalette {
		return e.updatePalette(km)
	}
	return e.updateHexInput(km)
}

func (e *colorEditor) updatePalette(km tea.KeyPressMsg) (tea.Cmd, bool, bool) {
	cols := e.gridCols()
	switch km.String() {
	case "left", "h":
		if e.palCursor > 0 {
			e.palCursor--
		}
	case "right", "l":
		if e.palCursor < len(e.palette)-1 {
			e.palCursor++
		}
	case "up", "k":
		if e.palCursor >= cols {
			e.palCursor -= cols
		}
	case "down", "j":
		if e.palCursor+cols < len(e.palette) {
			e.palCursor += cols
		}
	case "enter":
		e.val = formatPaletteColor(e.input.Value(), e.palette[e.palCursor])
		return nil, true, false
	case "esc":
		return nil, true, true
	case "tab":
		e.inPalette = false
		e.input.SetValue(e.palette[e.palCursor])
		e.input.CursorEnd()
		return e.input.Focus(), false, false
	}
	return nil, false, false
}

func (e *colorEditor) updateHexInput(km tea.KeyPressMsg) (tea.Cmd, bool, bool) {
	switch km.String() {
	case "enter":
		e.val = e.input.Value()
		return nil, true, false
	case "esc":
		return nil, true, true
	case "tab":
		if len(e.palette) > 0 {
			e.inPalette = true
			e.input.Blur()
			return nil, false, false
		}
	}
	var cmd tea.Cmd
	e.input, cmd = e.input.Update(km)
	return cmd, false, false
}

func (e *colorEditor) gridCols() int {
	if e.cols > 0 {
		return e.cols
	}
	e.cols = 8
	return e.cols
}

// PreviewValue returns the currently hovered/input color for live preview.
func (e *colorEditor) PreviewValue() string {
	if e.inPalette && e.palCursor < len(e.palette) {
		return e.palette[e.palCursor]
	}
	return e.input.Value()
}

func (e *colorEditor) View(width int) string {
	if len(e.palette) == 0 {
		return e.viewHexOnly(width)
	}
	return e.viewWithPalette(width)
}

func (e *colorEditor) viewHexOnly(width int) string {
	e.input.SetWidth(width - 4)

	inputLine := "    " + e.input.View()

	newHex := normalizeHex(e.input.Value())
	swatchLine := "    " + swatch(e.oldHex) +
		e.th.Muted.Render(" → ") +
		swatch(newHex) +
		" " + e.th.FieldValue.Render(newHex)

	return inputLine + "\n" + swatchLine
}

// groupAt returns the palette group starting at index i, or nil.
func (e *colorEditor) groupAt(i int) *paletteGroup {
	for g := range e.groups {
		if e.groups[g].start == i {
			return &e.groups[g]
		}
	}
	return nil
}

func (e *colorEditor) viewWithPalette(width int) string {
	cellW := 12
	cols := (width - 4) / cellW
	if cols < 1 {
		cols = 1
	}
	e.cols = cols

	cellStyle := lipgloss.NewStyle().Width(cellW).MaxWidth(cellW)

	var b strings.Builder

	colPos := 0
	for i, hex := range e.palette {
		// separator before each palette group
		if g := e.groupAt(i); g != nil && i > 0 {
			if colPos > 0 {
				b.WriteByte('\n')
				colPos = 0
			}
			sep := "── " + g.name + " ──"
			b.WriteString("    " + e.th.Muted.Render(sep))
			b.WriteByte('\n')
		}

		if colPos > 0 && colPos%cols == 0 {
			b.WriteByte('\n')
			colPos = 0
		}
		if colPos == 0 {
			b.WriteString("    ")
		}

		sw := swatch(normalizeHex(hex))
		label := e.labels[i]
		if len(label) > 8 {
			label = label[:8]
		}

		var cell string
		if i == e.palCursor && e.inPalette {
			cell = e.th.Primary.Render("▎") + sw + " " + e.th.Text.Bold(true).Render(label) + " "
		} else {
			cell = " " + sw + " " + e.th.Muted.Render(label) + " "
		}
		b.WriteString(cellStyle.Render(cell))
		colPos++
	}
	b.WriteByte('\n')

	// bottom line: live comparison + mode hint
	if e.inPalette {
		sel := ""
		if e.palCursor < len(e.palette) {
			sel = e.palette[e.palCursor]
		}
		newHex := normalizeHex(sel)
		b.WriteString("    " + swatch(e.oldHex) +
			e.th.Muted.Render(" → ") +
			swatch(newHex) +
			" " + e.th.FieldValue.Render(newHex) +
			e.th.Muted.Render("  tab:hex"))
	} else {
		e.input.SetWidth(width - 4)
		b.WriteString("    " + e.input.View())
		if len(e.palette) > 0 {
			b.WriteString(e.th.Muted.Render("  tab:palette"))
		}
	}

	return b.String()
}

func (e *colorEditor) Value() string { return e.val }

func (e *colorEditor) Height() int {
	if len(e.palette) == 0 {
		return 2
	}
	cols := e.gridCols()
	rows := (len(e.palette) + cols - 1) / cols
	h := rows + 1 // grid rows + comparison line
	// each group with start > 0 adds a separator line (and may cause a partial-row break)
	for _, g := range e.groups {
		if g.start > 0 {
			h++
		}
	}
	return h
}

var swatchCache sync.Map // string → string

func swatch(hex string) string {
	renderHex := colorRenderHex(hex)
	if renderHex == "" {
		return "  "
	}
	if v, ok := swatchCache.Load(renderHex); ok {
		return v.(string)
	}
	s := lipgloss.NewStyle().Foreground(lipgloss.Color(renderHex)).Render("██")
	swatchCache.Store(renderHex, s)
	return s
}

// colorValue renders a color's hex tinted in its own color. when the tint sits
// too close to bgHex to stay legible, it adds a contrasting backdrop so the
// value remains readable. bgHex "" skips the contrast guard.
func colorValue(hex, bgHex string) string {
	display := colorDisplayValue(hex)
	if display == "" {
		return ""
	}
	renderHex := colorRenderHex(hex)
	if renderHex == "" {
		return display
	}
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(renderHex))
	if bgHex != "" && lowContrast(renderHex, bgHex) {
		style = style.Background(lipgloss.Color(contrastBackdrop(renderHex)))
	}
	return style.Render(display)
}

// hexRGB parses the rgb channels of a "#rrggbb[aa]" hex, ignoring any alpha.
func hexRGB(hex string) (r, g, b int, ok bool) {
	h := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(hex)), "#")
	if len(h) < 6 || !isHex(h[:6]) {
		return 0, 0, 0, false
	}
	v, err := strconv.ParseInt(h[:6], 16, 64)
	if err != nil {
		return 0, 0, 0, false
	}
	return int(v>>16) & 0xff, int(v>>8) & 0xff, int(v) & 0xff, true
}

// relLuminance returns perceptual luminance 0..1 for a hex color (0 on parse failure).
func relLuminance(hex string) float64 {
	r, g, b, ok := hexRGB(hex)
	if !ok {
		return 0
	}
	return (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 255
}

// lowContrast reports whether fg is too close to bg to read, using a WCAG-style
// luminance contrast ratio.
func lowContrast(fg, bg string) bool {
	hi, lo := relLuminance(fg), relLuminance(bg)
	if lo > hi {
		hi, lo = lo, hi
	}
	return (hi+0.05)/(lo+0.05) < 2.5
}

// contrastBackdrop picks a backdrop that maximizes contrast with fg: a light
// chip for dark colors, a dark chip for light ones.
func contrastBackdrop(fg string) string {
	if relLuminance(fg) < 0.5 {
		return "#e6e6e6"
	}
	return "#1a1a1a"
}

// normalizeHex returns the color value used for display.
func normalizeHex(s string) string {
	return colorDisplayValue(s)
}

func colorDisplayValue(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "#0x") && len(lower) == 11 && isHex(lower[3:]) {
		return lower[1:]
	}
	if strings.HasPrefix(lower, "#") {
		digits := lower[1:]
		if (len(digits) == 6 || len(digits) == 8) && isHex(digits) {
			return "#" + digits
		}
		return s
	}
	if (len(lower) == 6 || len(lower) == 8) && isHex(lower) {
		return "#" + lower
	}
	return s
}

func colorRenderHex(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if hex := colorRenderHexToken(s); hex != "" {
		return hex
	}
	if fields := strings.Fields(s); len(fields) > 0 {
		return colorRenderHexToken(fields[0])
	}
	return ""
}

func colorRenderHexToken(s string) string {
	lower := strings.ToLower(strings.TrimSpace(s))
	if lower == "" {
		return ""
	}
	if strings.HasPrefix(lower, "#0x") {
		lower = lower[1:]
	}
	if strings.HasPrefix(lower, "0x") {
		digits := lower[2:]
		if len(digits) == 8 && isHex(digits) {
			return "#" + digits[2:]
		}
		return ""
	}
	for _, prefix := range []string{"rgba(", "rgb("} {
		if !strings.HasPrefix(lower, prefix) {
			continue
		}
		closeIdx := strings.IndexByte(lower, ')')
		if closeIdx < len(prefix) {
			return ""
		}
		digits := strings.TrimSpace(lower[len(prefix):closeIdx])
		if (len(digits) == 6 || len(digits) == 8) && isHex(digits) {
			return "#" + digits[:6]
		}
		return ""
	}
	if strings.HasPrefix(lower, "#") {
		digits := lower[1:]
		if (len(digits) == 6 || len(digits) == 8) && isHex(digits) {
			return "#" + digits[:6]
		}
		return ""
	}
	if (len(lower) == 6 || len(lower) == 8) && isHex(lower) {
		return "#" + lower[:6]
	}
	return ""
}

func formatPaletteColor(template, selected string) string {
	rgb := strings.TrimPrefix(colorRenderHex(selected), "#")
	if rgb == "" {
		return selected
	}
	t := strings.TrimSpace(template)
	lower := strings.ToLower(t)
	if strings.HasPrefix(lower, "#0x") {
		lower = lower[1:]
	}
	if strings.HasPrefix(lower, "0x") {
		digits := lower[2:]
		if len(digits) == 8 && isHex(digits) {
			return "0x" + digits[:2] + rgb
		}
	}
	for _, prefix := range []string{"rgba(", "rgb("} {
		if strings.HasPrefix(lower, prefix) {
			closeIdx := strings.IndexByte(lower, ')')
			if closeIdx != len(lower)-1 {
				return selected
			}
			digits := strings.TrimSpace(lower[len(prefix):closeIdx])
			switch {
			case prefix == "rgba(" && len(digits) == 8 && isHex(digits):
				return "rgba(" + rgb + digits[6:] + ")"
			case prefix == "rgb(" && len(digits) == 6 && isHex(digits):
				return "rgb(" + rgb + ")"
			}
		}
	}
	return selected
}

func isHex(s string) bool {
	for _, r := range s {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			continue
		}
		return false
	}
	return s != ""
}
