package ui

import (
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

	for _, pal := range ordered {
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
		for i, hex := range e.palette {
			if hex == currentValue || normalizeHex(hex) == normalizeHex(currentValue) {
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
		e.val = e.palette[e.palCursor]
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
			cell = e.th.Primary.Render("[") + sw + " " + e.th.Text.Bold(true).Render(label) + e.th.Primary.Render("]")
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
	if hex == "" {
		return "  "
	}
	if v, ok := swatchCache.Load(hex); ok {
		return v.(string)
	}
	s := lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render("██")
	swatchCache.Store(hex, s)
	return s
}

// normalizeHex prepends # if missing so lipgloss can render it.
// returns "" for empty input — callers must handle that.
func normalizeHex(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if s[0] != '#' {
		return "#" + s
	}
	return s
}
