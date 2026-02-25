package ui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/emin/konfigurator/pkg"
	"github.com/emin/konfigurator/theme"

	tea "github.com/charmbracelet/bubbletea"
)

var stylestringRe = regexp.MustCompile(`^\[(.+?)\]\((.+?)\)$`)

// parseStyleString extracts symbol and style from "[symbol](style)".
// returns (raw, "") if the format doesn't match.
func parseStyleString(s string) (string, string) {
	m := stylestringRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return s, ""
	}
	return m[1], m[2]
}

// composeStyleString produces "[symbol](style)".
func composeStyleString(symbol, style string) string {
	return "[" + symbol + "](" + style + ")"
}

type stylestringEditor struct {
	symbols    []string
	styles     []string
	symCursor  int
	styCursor  int
	symCurrent int // index of current symbol, -1 if not found
	styCurrent int // index of current style, -1 if not found
	pane       int // 0=symbol, 1=style
	val        string
	th         *theme.Theme
}

func (e *stylestringEditor) Init(field pkg.Field, currentValue string, th *theme.Theme) tea.Cmd {
	e.th = th
	e.symbols = field.Options
	e.styles = field.AltOptions
	e.symCurrent = -1
	e.styCurrent = -1

	sym, sty := parseStyleString(currentValue)

	for i, o := range e.symbols {
		if o == sym {
			e.symCursor = i
			e.symCurrent = i
			break
		}
	}
	for i, o := range e.styles {
		if o == sty {
			e.styCursor = i
			e.styCurrent = i
			break
		}
	}
	return nil
}

func (e *stylestringEditor) Update(msg tea.Msg) (tea.Cmd, bool, bool) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil, false, false
	}
	switch km.String() {
	case "j", "down":
		if e.pane == 0 {
			if e.symCursor < len(e.symbols)-1 {
				e.symCursor++
			}
		} else {
			if e.styCursor < len(e.styles)-1 {
				e.styCursor++
			}
		}
	case "k", "up":
		if e.pane == 0 {
			if e.symCursor > 0 {
				e.symCursor--
			}
		} else {
			if e.styCursor > 0 {
				e.styCursor--
			}
		}
	case "tab":
		e.pane = 1 - e.pane
	case "enter":
		sym := ""
		if e.symCursor < len(e.symbols) {
			sym = e.symbols[e.symCursor]
		}
		sty := ""
		if e.styCursor < len(e.styles) {
			sty = e.styles[e.styCursor]
		}
		e.val = composeStyleString(sym, sty)
		return nil, true, false
	case "esc":
		return nil, true, true
	}
	return nil, false, false
}

func (e *stylestringEditor) View(width int) string {
	// compute column widths
	symW := 14
	divider := " │ "

	rows := max(len(e.symbols), len(e.styles))

	var b strings.Builder

	// column headers
	symHeader := "symbol"
	styHeader := "style"
	if e.pane == 0 {
		symHeader = e.th.Text.Bold(true).Render(symHeader)
		styHeader = e.th.Muted.Render(styHeader)
	} else {
		symHeader = e.th.Muted.Render(symHeader)
		styHeader = e.th.Text.Bold(true).Render(styHeader)
	}
	fmt.Fprintf(&b, "    %-*s%s%s\n", symW, symHeader, e.th.Muted.Render(divider), styHeader)

	for i := range rows {
		// symbol column
		var symCell string
		if i < len(e.symbols) {
			sym := e.symbols[i]
			if i == e.symCursor && e.pane == 0 {
				symCell = e.th.Primary.Render("> ") + e.th.Text.Bold(true).Render(sym)
			} else if i == e.symCurrent {
				symCell = "  " + e.th.Accent.Render(sym)
			} else {
				symCell = "  " + e.th.Subtext.Render(sym)
			}
		}
		symCell = fmt.Sprintf("%-*s", symW, symCell)

		// style column
		var styCell string
		if i < len(e.styles) {
			sty := e.styles[i]
			if i == e.styCursor && e.pane == 1 {
				styCell = e.th.Primary.Render("> ") + e.th.Text.Bold(true).Render(sty)
			} else if i == e.styCurrent {
				styCell = "  " + e.th.Accent.Render(sty)
			} else {
				styCell = "  " + e.th.Subtext.Render(sty)
			}
		}

		b.WriteString("    " + symCell + e.th.Muted.Render(divider) + styCell)
		if i < rows-1 {
			b.WriteByte('\n')
		}
	}

	// preview line
	sym := ""
	if e.symCursor < len(e.symbols) {
		sym = e.symbols[e.symCursor]
	}
	sty := ""
	if e.styCursor < len(e.styles) {
		sty = e.styles[e.styCursor]
	}
	preview := composeStyleString(sym, sty)
	b.WriteByte('\n')
	b.WriteString("    " + e.th.Muted.Render("preview: ") + e.th.Accent.Render(preview))

	return b.String()
}

// PreviewValue returns the currently composed value for live detail panel updates.
func (e *stylestringEditor) PreviewValue() string {
	sym := ""
	if e.symCursor < len(e.symbols) {
		sym = e.symbols[e.symCursor]
	}
	sty := ""
	if e.styCursor < len(e.styles) {
		sty = e.styles[e.styCursor]
	}
	return composeStyleString(sym, sty)
}

func (e *stylestringEditor) Value() string { return e.val }

func (e *stylestringEditor) Height() int {
	return max(len(e.symbols), len(e.styles)) + 2 // header + rows + preview
}
