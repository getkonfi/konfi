package ui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/eminert/konfi/pkg"
	"github.com/eminert/konfi/theme"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	styCursor  int // flat index into styles
	symCurrent int // index of current symbol, -1 if not found
	styCurrent int // index of current style, -1 if not found
	pane       int // 0=symbol, 1=style-left, 2=style-right
	val        string
	th         *theme.Theme
}

// styHalf returns the split point for style columns (ceil division).
func (e *stylestringEditor) styHalf() int {
	return (len(e.styles) + 1) / 2
}

// adjustStyCursorForPane moves styCursor into the correct half for the active pane.
func (e *stylestringEditor) adjustStyCursorForPane() {
	if e.pane != 1 && e.pane != 2 {
		return
	}
	half := e.styHalf()
	inLeft := e.styCursor < half

	if e.pane == 1 && !inLeft {
		// cursor is in right half but we switched to left pane
		e.styCursor = min(e.styCursor-half, half-1)
	} else if e.pane == 2 && inLeft {
		// cursor is in left half but we switched to right pane
		e.styCursor = min(e.styCursor+half, len(e.styles)-1)
	}
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
	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil, false, false
	}
	half := e.styHalf()
	switch km.String() {
	case "j", "down":
		switch e.pane {
		case 0:
			if e.symCursor < len(e.symbols)-1 {
				e.symCursor++
			}
		case 1:
			if e.styCursor < half-1 {
				e.styCursor++
			}
		case 2:
			if e.styCursor < len(e.styles)-1 {
				e.styCursor++
			}
		}
	case "k", "up":
		switch e.pane {
		case 0:
			if e.symCursor > 0 {
				e.symCursor--
			}
		case 1:
			if e.styCursor > 0 {
				e.styCursor--
			}
		case 2:
			if e.styCursor > half {
				e.styCursor--
			}
		}
	case "tab":
		e.pane = (e.pane + 1) % 3
		e.adjustStyCursorForPane()
	case "shift+tab":
		e.pane = (e.pane + 2) % 3
		e.adjustStyCursorForPane()
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

// padTo pads s with spaces to reach visible width w (ANSI-aware).
func padTo(s string, w int) string {
	vis := lipgloss.Width(s)
	if vis >= w {
		return s
	}
	return s + strings.Repeat(" ", w-vis)
}

func (e *stylestringEditor) View(width int) string {
	half := e.styHalf()
	symW := 14
	styW := 16
	divider := " │ "

	rows := max(len(e.symbols), half)

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
	fmt.Fprintf(&b, "    %s%s%s\n", padTo(symHeader, symW), e.th.Muted.Render(divider), styHeader)

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

		// left style column (indices 0..half-1)
		var styLeftCell string
		if i < half {
			sty := e.styles[i]
			if i == e.styCursor && e.pane == 1 {
				styLeftCell = e.th.Primary.Render("> ") + e.th.Text.Bold(true).Render(sty)
			} else if i == e.styCurrent {
				styLeftCell = "  " + e.th.Accent.Render(sty)
			} else {
				styLeftCell = "  " + e.th.Subtext.Render(sty)
			}
		}

		// right style column (indices half..len-1)
		var styRightCell string
		ri := half + i
		if ri < len(e.styles) {
			sty := e.styles[ri]
			if ri == e.styCursor && e.pane == 2 {
				styRightCell = e.th.Primary.Render("> ") + e.th.Text.Bold(true).Render(sty)
			} else if ri == e.styCurrent {
				styRightCell = "  " + e.th.Accent.Render(sty)
			} else {
				styRightCell = "  " + e.th.Subtext.Render(sty)
			}
		}

		b.WriteString("    " + padTo(symCell, symW) + e.th.Muted.Render(divider) + padTo(styLeftCell, styW) + "  " + styRightCell)
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
	half := (len(e.styles) + 1) / 2
	return max(len(e.symbols), half) + 2
}
