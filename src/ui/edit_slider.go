package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/eminert/konfi/pkg"
	"github.com/eminert/konfi/theme"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type sliderEditor struct {
	field  pkg.Field
	val    float64
	min    float64
	max    float64
	step   float64
	prec   int
	th     *theme.Theme
	input  textinput.Model
	typing bool
	done   string // final committed value
}

func (e *sliderEditor) Init(field pkg.Field, currentValue string, th *theme.Theme) tea.Cmd {
	e.field = field
	e.th = th
	e.min = derefFloat(field.Min, 0)
	e.max = derefFloat(field.Max, 1)
	e.val, _ = strconv.ParseFloat(currentValue, 64)
	e.val = clampFloat(e.val, e.min, e.max)
	e.step = (e.max - e.min) / 50.0
	e.prec = precisionForStep(e.step)

	e.input = textinput.New()
	e.input.Prompt = "┊ "
	s := textinput.DefaultDarkStyles()
	s.Focused.Prompt = th.Muted
	e.input.SetStyles(s)
	e.input.Validate = numberValidateChar
	return nil
}

func (e *sliderEditor) Update(msg tea.Msg) (tea.Cmd, bool, bool) {
	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		if e.typing {
			var cmd tea.Cmd
			e.input, cmd = e.input.Update(msg)
			return cmd, false, false
		}
		return nil, false, false
	}

	if e.typing {
		return e.updateTyping(km)
	}

	switch km.String() {
	case "left", "h":
		e.val = clampFloat(e.val-e.step, e.min, e.max)
	case "right", "l":
		e.val = clampFloat(e.val+e.step, e.min, e.max)
	case "shift+left", "H":
		e.val = clampFloat(e.val-e.step*10, e.min, e.max)
	case "shift+right", "L":
		e.val = clampFloat(e.val+e.step*10, e.min, e.max)
	case "enter":
		e.done = strconv.FormatFloat(e.val, 'f', e.prec, 64)
		return nil, true, false
	case "esc":
		return nil, true, true
	default:
		// digit or dot starts direct entry
		if len(km.Text) == 1 {
			r := rune(km.Text[0])
			if (r >= '0' && r <= '9') || r == '.' || r == '-' {
				e.typing = true
				e.input.SetValue(km.Text)
				e.input.CursorEnd()
				return e.input.Focus(), false, false
			}
		}
	}
	return nil, false, false
}

func (e *sliderEditor) updateTyping(km tea.KeyPressMsg) (tea.Cmd, bool, bool) {
	switch km.String() {
	case "enter":
		v, err := strconv.ParseFloat(e.input.Value(), 64)
		if err != nil {
			e.typing = false
			e.input.Blur()
			return nil, false, false
		}
		e.val = clampFloat(v, e.min, e.max)
		e.done = strconv.FormatFloat(e.val, 'f', e.prec, 64)
		return nil, true, false
	case "esc":
		e.typing = false
		e.input.Blur()
		return nil, false, false
	}
	var cmd tea.Cmd
	e.input, cmd = e.input.Update(km)
	return cmd, false, false
}

func (e *sliderEditor) View(width int) string {
	return "    " + e.InlineView(width-4)
}

func (e *sliderEditor) InlineView(width int) string {
	if e.typing {
		e.input.SetWidth(width)
		return e.input.View()
	}

	valStr := strconv.FormatFloat(e.val, 'f', e.prec, 64)

	// try full layout with range hint, drop hint if it won't fit
	rangeHint := fmt.Sprintf("(%s — %s)", formatNum(e.min), formatNum(e.max))
	suffix := "  " + valStr + "  " + e.th.Muted.Render(rangeHint)
	suffixW := lipgloss.Width(suffix)
	barW := width - suffixW - 2
	if barW < 5 {
		suffix = "  " + valStr
		suffixW = lipgloss.Width(suffix)
		barW = width - suffixW - 2
	}
	barW = max(barW, 3)

	ratio := (e.val - e.min) / (e.max - e.min)
	ratio = max(0, min(ratio, 1))
	filled := min(int(ratio*float64(barW)), barW)

	// shade character reflects value — denser at higher ratios
	shade := shadeForRatio(ratio)
	bar := e.th.Primary.Render(strings.Repeat(shade, filled)) +
		e.th.Muted.Render(strings.Repeat("░", barW-filled))

	return "[" + bar + "]" + suffix
}

func shadeForRatio(ratio float64) string {
	switch {
	case ratio >= 0.75:
		return "█"
	case ratio >= 0.5:
		return "▓"
	case ratio >= 0.25:
		return "▒"
	default:
		return "░"
	}
}

func (e *sliderEditor) Value() string { return e.done }
func (e *sliderEditor) Height() int   { return 0 }

func derefFloat(p *float64, fallback float64) float64 {
	if p == nil {
		return fallback
	}
	return *p
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func precisionForStep(step float64) int {
	if step >= 1 {
		return 0
	}
	if step >= 0.1 {
		return 1
	}
	if step >= 0.01 {
		return 2
	}
	return 3
}
