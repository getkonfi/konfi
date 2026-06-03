package ui

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/eminert/konfi/pkg"
	"github.com/eminert/konfi/theme"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type numberEditor struct {
	input    textinput.Model
	field    pkg.Field
	val      string
	errMsg   string
	errStyle lipgloss.Style
	hint     string
	isFloat  bool
}

func (e *numberEditor) Init(field pkg.Field, currentValue string, th *theme.Theme) tea.Cmd {
	e.field = field
	e.isFloat = strings.Contains(currentValue, ".")
	e.errStyle = th.Error

	e.input = textinput.New()
	e.input.Prompt = "┊ "
	s := textinput.DefaultDarkStyles()
	s.Focused.Prompt = th.Muted
	s.Focused.Text = th.Text
	e.input.SetStyles(s)
	e.input.Validate = numberValidateChar
	e.input.SetValue(currentValue)
	e.input.CursorEnd()

	// build range hint
	if field.Min != nil || field.Max != nil {
		lo, hi := "*", "*"
		if field.Min != nil {
			lo = formatNum(*field.Min)
		}
		if field.Max != nil {
			hi = formatNum(*field.Max)
		}
		e.hint = th.Muted.Render(fmt.Sprintf(" (%s — %s)", lo, hi))
	}

	return e.input.Focus()
}

func (e *numberEditor) Update(msg tea.Msg) (tea.Cmd, bool, bool) {
	if km, ok := msg.(tea.KeyPressMsg); ok {
		switch km.String() {
		case "enter":
			if err := e.validate(); err != "" {
				e.errMsg = err
				return nil, false, false
			}
			e.val = e.input.Value()
			return nil, true, false
		case "esc":
			return nil, true, true
		case "up":
			e.step(1)
			e.errMsg = ""
			return nil, false, false
		case "down":
			e.step(-1)
			e.errMsg = ""
			return nil, false, false
		}
	}

	e.errMsg = ""
	var cmd tea.Cmd
	e.input, cmd = e.input.Update(msg)
	return cmd, false, false
}

func (e *numberEditor) View(width int) string {
	e.input.SetWidth(width - 4 - lipgloss.Width(e.hint))
	line := "    " + e.input.View() + e.hint
	if e.errMsg != "" {
		line += " " + e.errStyle.Render(e.errMsg)
	}
	return line
}

func (e *numberEditor) InlineView(width int) string {
	suffix := e.hint
	if e.errMsg != "" {
		suffix += " " + e.errStyle.Render(e.errMsg)
	}
	w := width - lipgloss.Width(suffix)
	if w < 1 {
		w = 1
	}
	e.input.SetWidth(w)
	return e.input.View() + suffix
}

func (e *numberEditor) Value() string { return e.val }
func (e *numberEditor) Height() int   { return 0 }

func (e *numberEditor) step(dir int) {
	cur, err := strconv.ParseFloat(e.input.Value(), 64)
	if err != nil {
		return
	}
	step := 1.0
	if e.isFloat {
		step = 0.1
	}
	next := cur + float64(dir)*step
	if e.isFloat {
		e.input.SetValue(fmt.Sprintf("%.1f", next))
	} else {
		e.input.SetValue(strconv.Itoa(int(next)))
	}
	e.input.CursorEnd()
}

func (e *numberEditor) validate() string {
	v := e.input.Value()
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return "invalid number"
	}
	if e.field.Min != nil && n < *e.field.Min {
		return fmt.Sprintf("min %s", formatNum(*e.field.Min))
	}
	if e.field.Max != nil && n > *e.field.Max {
		return fmt.Sprintf("max %s", formatNum(*e.field.Max))
	}
	return ""
}

func numberValidateChar(s string) error {
	for _, r := range s {
		if !unicode.IsDigit(r) && r != '.' && r != '-' {
			return fmt.Errorf("invalid")
		}
	}
	return nil
}

func formatNum(f float64) string {
	if f == float64(int(f)) {
		return strconv.Itoa(int(f))
	}
	return fmt.Sprintf("%.1f", f)
}
