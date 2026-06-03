package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/eminert/konfi/pkg"
	"github.com/eminert/konfi/theme"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type askState int

const (
	askInput askState = iota
	askLoading
	askResults
	askError
)

const askMaxVisible = 8

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// askOverlay is the AI-powered config discovery overlay.
type askOverlay struct {
	visible      bool
	state        askState
	input        textinput.Model
	results      []pkg.AISuggestion
	selected     int
	errText      string
	width        int
	height       int
	theme        *theme.Theme
	spinnerFrame int
	spinnerGen   int
	cancel       context.CancelFunc
	schemas      map[string]*pkg.Schema
}

func newAskOverlay(th *theme.Theme, schemas map[string]*pkg.Schema) askOverlay {
	ti := textinput.New()
	ti.Placeholder = "describe what you want to configure..."
	ti.CharLimit = 256
	ti.Prompt = ""

	return askOverlay{
		input:   ti,
		theme:   th,
		schemas: schemas,
	}
}

func (a *askOverlay) Open() tea.Cmd {
	a.visible = true
	a.state = askInput
	a.results = nil
	a.selected = 0
	a.errText = ""
	a.input.SetValue("")
	a.input.Focus()
	return textinput.Blink
}

func (a *askOverlay) Close() {
	a.visible = false
	a.input.Blur()
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	a.results = nil
	a.errText = ""
}

func (a *askOverlay) Visible() bool {
	return a.visible
}

func askSpinnerCmd(gen int) tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg {
		return askSpinnerTickMsg{gen: gen}
	})
}

func (a askOverlay) Update(msg tea.Msg) (askOverlay, tea.Cmd) {
	if !a.visible {
		return a, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch a.state {
		case askInput:
			switch msg.String() {
			case "esc":
				a.Close()
				return a, nil
			case "enter":
				query := strings.TrimSpace(a.input.Value())
				if query == "" {
					return a, nil
				}
				a.state = askLoading
				a.spinnerGen++
				gen := a.spinnerGen
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				a.cancel = cancel
				schemas := a.schemas
				cmd := func() tea.Msg {
					results, err := pkg.AskClaude(ctx, schemas, query)
					return AskResultMsg{Results: results, Err: err}
				}
				return a, tea.Batch(cmd, askSpinnerCmd(gen))
			default:
				var cmd tea.Cmd
				a.input, cmd = a.input.Update(msg)
				return a, cmd
			}

		case askLoading:
			if msg.String() == "esc" {
				if a.cancel != nil {
					a.cancel()
					a.cancel = nil
				}
				a.state = askInput
				return a, nil
			}
			return a, nil

		case askResults:
			switch msg.String() {
			case "esc":
				// back to input to refine query
				a.state = askInput
				a.input.Focus()
				return a, textinput.Blink
			case "enter":
				if len(a.results) > 0 && a.selected < len(a.results) {
					r := a.results[a.selected]
					return a, func() tea.Msg {
						return AskJumpMsg{App: r.App, Key: r.Key}
					}
				}
				return a, nil
			case "j", "down":
				if a.selected < len(a.results)-1 {
					a.selected++
				}
				return a, nil
			case "k", "up":
				if a.selected > 0 {
					a.selected--
				}
				return a, nil
			}
			return a, nil

		case askError:
			if msg.String() == "esc" || msg.String() == "enter" {
				a.state = askInput
				a.input.Focus()
				return a, textinput.Blink
			}
			return a, nil
		}

	case AskResultMsg:
		if a.cancel != nil {
			a.cancel()
			a.cancel = nil
		}
		if msg.Err != nil {
			a.state = askError
			a.errText = msg.Err.Error()
			return a, nil
		}
		if len(msg.Results) == 0 {
			a.state = askError
			a.errText = "no matching options found"
			return a, nil
		}
		a.state = askResults
		a.results = msg.Results
		a.selected = 0
		return a, nil

	case askSpinnerTickMsg:
		if a.state != askLoading || msg.gen != a.spinnerGen {
			return a, nil
		}
		a.spinnerFrame++
		return a, askSpinnerCmd(a.spinnerGen)

	case ThemeChangedMsg:
		a.theme = msg.Theme
	}

	// forward non-key msgs to textinput (blink, etc.)
	var cmd tea.Cmd
	a.input, cmd = a.input.Update(msg)
	return a, cmd
}

func (a askOverlay) View() string {
	if !a.visible || a.theme == nil {
		return ""
	}

	pal := a.theme.Palette
	w := a.askWidth()
	innerW := w - 4
	if innerW < 20 {
		innerW = 20
	}

	// title
	title := a.theme.Primary.Bold(true).Render(" ask ")
	hint := a.theme.Muted.Render("  natural language config search")
	header := title + hint

	// input row
	prompt := a.theme.Primary.Render("> ")
	inputRow := prompt + a.input.View()

	sep := a.theme.Muted.Render(strings.Repeat("─", innerW))

	var body strings.Builder
	body.WriteString(header)
	body.WriteByte('\n')
	body.WriteString(inputRow)
	body.WriteByte('\n')
	body.WriteString(sep)

	switch a.state {
	case askInput:
		// just the input, nothing below separator

	case askLoading:
		frame := spinnerFrames[a.spinnerFrame%len(spinnerFrames)]
		body.WriteByte('\n')
		body.WriteString(a.theme.Primary.Render(frame) + a.theme.Muted.Render(" asking claude..."))

	case askResults:
		offset := 0
		visible := a.results
		if len(visible) > askMaxVisible {
			if a.selected >= askMaxVisible {
				offset = a.selected - askMaxVisible + 1
			}
			visible = visible[offset:]
			if len(visible) > askMaxVisible {
				visible = visible[:askMaxVisible]
			}
		}
		for i, r := range visible {
			realIdx := offset + i
			body.WriteByte('\n')
			body.WriteString(a.renderResult(r, realIdx == a.selected, innerW))
		}
		if len(a.results) > askMaxVisible {
			body.WriteByte('\n')
			body.WriteString(sep)
			body.WriteByte('\n')
			body.WriteString(a.theme.Muted.Render(
				fmt.Sprintf("  %d/%d", a.selected+1, len(a.results))))
		}

	case askError:
		body.WriteByte('\n')
		body.WriteString(a.theme.Error.Render("  " + a.errText))
		body.WriteByte('\n')
		body.WriteString(a.theme.Muted.Render("  press esc to go back"))
	}

	// footer hints
	body.WriteByte('\n')
	body.WriteString(sep)
	body.WriteByte('\n')
	switch a.state {
	case askInput:
		body.WriteString(a.theme.Muted.Render("  ⏎ search  esc close"))
	case askLoading:
		body.WriteString(a.theme.Muted.Render("  esc cancel"))
	case askResults:
		body.WriteString(a.theme.Muted.Render("  ↑↓ navigate  ⏎ jump to field  esc refine"))
	case askError:
		body.WriteString(a.theme.Muted.Render("  esc back  ⏎ retry"))
	}

	panel := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(pal.BorderFocus).
		Background(pal.Base).
		Foreground(pal.Text).
		Padding(0, 1).
		Width(w - 2).
		Render(body.String())

	return lipgloss.Place(
		a.width, a.height,
		lipgloss.Center, lipgloss.Top,
		panel,
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Foreground(pal.Muted)),
		lipgloss.WithWhitespaceChars(" "),
	)
}

func (a askOverlay) renderResult(r pkg.AISuggestion, selected bool, maxW int) string {
	prefix := "  "
	if selected {
		prefix = a.theme.Primary.Render("> ")
	}

	// app badge
	app := a.theme.Muted.Render("[" + r.App + "] ")

	// section / key
	loc := ""
	if r.Section != "" {
		loc = a.theme.Muted.Render(r.Section+"/") + a.theme.Text.Bold(true).Render(r.Key)
	} else {
		loc = a.theme.Text.Bold(true).Render(r.Key)
	}

	// suggested value
	val := ""
	if r.Value != "" {
		val = a.theme.Accent.Render(" → " + r.Value)
	}

	line1 := prefix + app + loc + val

	// reason on second line
	line2 := ""
	if r.Reason != "" {
		reason := r.Reason
		if len(reason) > maxW-4 {
			reason = reason[:maxW-7] + "..."
		}
		line2 = "\n    " + a.theme.Muted.Render(reason)
	}

	return line1 + line2
}

func (a askOverlay) askWidth() int {
	w := a.width * 70 / 100
	if w < 50 {
		w = 50
	}
	if w > 90 {
		w = 90
	}
	if w > a.width {
		w = a.width
	}
	return w
}
