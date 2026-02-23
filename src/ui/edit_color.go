package ui

import (
	"strings"

	"github.com/emin/konfigurator/pkg"
	"github.com/emin/konfigurator/theme"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type colorEditor struct {
	input     textinput.Model
	val       string
	oldHex    string
	th        *theme.Theme
	palette   []string
	palCursor int
	inPalette bool
	cols      int // grid columns, computed on first View
}

func (e *colorEditor) Init(field pkg.Field, currentValue string, th *theme.Theme) tea.Cmd {
	e.th = th
	e.oldHex = normalizeHex(currentValue)
	e.palette = field.Palette

	e.input = textinput.New()
	e.input.Prompt = "┊ "
	e.input.PromptStyle = th.Muted
	e.input.TextStyle = th.Text
	e.input.SetValue(currentValue)
	e.input.CursorEnd()

	// start in palette mode if palette is available
	if len(e.palette) > 0 {
		e.inPalette = true
		// try to select current value in palette
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
	km, ok := msg.(tea.KeyMsg)
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

func (e *colorEditor) updatePalette(km tea.KeyMsg) (tea.Cmd, bool, bool) {
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

func (e *colorEditor) updateHexInput(km tea.KeyMsg) (tea.Cmd, bool, bool) {
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
	// default: 8 columns for typical terminal width
	e.cols = 8
	return e.cols
}

func (e *colorEditor) View(width int) string {
	if len(e.palette) == 0 {
		return e.viewHexOnly(width)
	}
	return e.viewWithPalette(width)
}

func (e *colorEditor) viewHexOnly(width int) string {
	e.input.Width = width - 4

	inputLine := "    " + e.input.View()

	newHex := normalizeHex(e.input.Value())
	swatchLine := "    " + swatch(e.oldHex) +
		e.th.Muted.Render(" old → ") +
		swatch(newHex) +
		e.th.Muted.Render(" new")

	return inputLine + "\n" + swatchLine
}

func (e *colorEditor) viewWithPalette(width int) string {
	// compute grid layout: each swatch cell is "██ XXXXXX " = ~11 chars
	cellW := 11
	cols := (width - 4) / cellW
	if cols < 1 {
		cols = 1
	}
	e.cols = cols

	var b strings.Builder

	// palette grid
	for i, hex := range e.palette {
		if i > 0 && i%cols == 0 {
			b.WriteByte('\n')
		}
		if i%cols == 0 {
			b.WriteString("    ")
		}

		sw := swatch(normalizeHex(hex))
		label := hex
		if len(label) > 6 {
			label = label[:6]
		}

		if i == e.palCursor && e.inPalette {
			b.WriteString(e.th.Primary.Render("[") + sw + " " + e.th.Text.Bold(true).Render(label) + e.th.Primary.Render("]"))
		} else {
			b.WriteString(" " + sw + " " + e.th.Muted.Render(label) + " ")
		}
	}
	b.WriteByte('\n')

	// hex input line
	if e.inPalette {
		// show selected palette color info
		sel := ""
		if e.palCursor < len(e.palette) {
			sel = e.palette[e.palCursor]
		}
		newHex := normalizeHex(sel)
		b.WriteString("    " + swatch(e.oldHex) +
			e.th.Muted.Render(" old → ") +
			swatch(newHex) +
			e.th.Muted.Render(" new") +
			e.th.Muted.Render("  tab:hex input"))
	} else {
		e.input.Width = width - 4
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
	return rows + 1 // grid rows + swatch/input line
}

func swatch(hex string) string {
	return lipgloss.NewStyle().
		Background(lipgloss.Color(hex)).
		Render("██")
}

// normalizeHex prepends # if missing so lipgloss can render it.
func normalizeHex(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "#000000"
	}
	if s[0] != '#' {
		return "#" + s
	}
	return s
}
