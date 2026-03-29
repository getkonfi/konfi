package ui

import (
	"strings"

	"github.com/emin/konfigurator/theme"
	"github.com/sahilm/fuzzy"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// PaletteMode determines what the palette searches.
type PaletteMode int

const (
	PaletteModeCommands PaletteMode = iota
	PaletteModeFields
)

const paletteMaxVisible = 10

// PaletteItem represents a searchable entry in the command palette.
type PaletteItem struct {
	Label       string  // display name
	Description string  // secondary text
	Shortcut    string  // keyboard shortcut hint (right-aligned)
	Category    string  // grouping (app name, "action", etc.)
	Action      tea.Msg // message to send when selected
	MatchTerms  string  // concatenated searchable text
}

// PaletteResult wraps a matched item with character positions that matched.
type PaletteResult struct {
	Item    PaletteItem
	Matched []int
}

// PaletteSelectedMsg is emitted when the user picks a palette item.
type PaletteSelectedMsg struct {
	Action tea.Msg
}

// PaletteClosedMsg is emitted when the palette is dismissed.
type PaletteClosedMsg struct{}

// palette is the command palette model — an overlay with fuzzy search.
type palette struct {
	input        textinput.Model
	mode         PaletteMode
	commandItems []PaletteItem
	fieldItems   []PaletteItem
	items        []PaletteItem
	results      []PaletteResult
	selected     int
	visible      bool
	width        int
	height       int
	theme        *theme.Theme
}

// matchSource adapts []PaletteItem for github.com/sahilm/fuzzy.
type matchSource []PaletteItem

func (s matchSource) String(i int) string { return s[i].MatchTerms }
func (s matchSource) Len() int            { return len(s) }

func newPalette(th *theme.Theme) palette {
	ti := textinput.New()
	ti.Placeholder = "type to search..."
	ti.CharLimit = 128
	ti.Prompt = ""

	return palette{
		input: ti,
		theme: th,
	}
}

// Open shows the palette with the given mode and items.
func (p *palette) Open(mode PaletteMode, cmdItems, fldItems []PaletteItem) tea.Cmd {
	p.visible = true
	p.mode = mode
	p.commandItems = cmdItems
	p.fieldItems = fldItems
	if mode == PaletteModeFields {
		p.items = fldItems
	} else {
		p.items = cmdItems
	}
	p.selected = 0
	p.input.SetValue("")
	p.input.Focus()
	p.filter()
	return textinput.Blink
}

// Close hides the palette and resets state.
func (p *palette) Close() {
	p.visible = false
	p.input.Blur()
	p.items = nil
	p.commandItems = nil
	p.fieldItems = nil
	p.results = nil
	p.selected = 0
}

// Visible reports whether the palette overlay is showing.
func (p *palette) Visible() bool {
	return p.visible
}

// Update handles input while the palette is visible.
func (p palette) Update(msg tea.Msg) (palette, tea.Cmd) {
	if !p.visible {
		return p, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			p.Close()
			return p, func() tea.Msg { return PaletteClosedMsg{} }

		case "enter":
			if len(p.results) > 0 && p.selected < len(p.results) {
				action := p.results[p.selected].Item.Action
				p.Close()
				return p, func() tea.Msg { return PaletteSelectedMsg{Action: action} }
			}
			return p, nil

		case "up", "ctrl+p":
			if p.selected > 0 {
				p.selected--
			}
			return p, nil

		case "down", "ctrl+n":
			if p.selected < len(p.results)-1 {
				p.selected++
			}
			return p, nil

		case "tab":
			if p.mode == PaletteModeCommands {
				p.mode = PaletteModeFields
				p.items = p.fieldItems
			} else {
				p.mode = PaletteModeCommands
				p.items = p.commandItems
			}
			p.selected = 0
			p.input.SetValue("")
			p.filter()
			return p, nil

		default:
			var cmd tea.Cmd
			p.input, cmd = p.input.Update(msg)
			p.filter()
			return p, cmd
		}

	case ThemeChangedMsg:
		p.theme = msg.Theme
	}

	// forward non-key msgs to textinput (blink, etc.)
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	return p, cmd
}

// filter runs fuzzy matching against current input.
func (p *palette) filter() {
	query := strings.TrimSpace(p.input.Value())
	p.selected = 0

	if query == "" {
		// show all items in original order
		p.results = make([]PaletteResult, len(p.items))
		for i := range p.items {
			p.results[i] = PaletteResult{Item: p.items[i]}
		}
		return
	}

	// split on spaces for AND logic
	terms := strings.Fields(query)
	if len(terms) == 0 {
		return
	}

	// start with all item indices as candidates
	type candidate struct {
		idx     int
		matched []int
	}
	candidates := make([]candidate, len(p.items))
	for i := range p.items {
		candidates[i] = candidate{idx: i}
	}

	// each term must match; intersect results
	for _, term := range terms {
		matches := fuzzy.FindFrom(term, matchSource(p.items))
		matchSet := make(map[int][]int, len(matches))
		for _, m := range matches {
			matchSet[m.Index] = m.MatchedIndexes
		}

		var next []candidate
		for _, c := range candidates {
			if positions, ok := matchSet[c.idx]; ok {
				next = append(next, candidate{
					idx:     c.idx,
					matched: mergePositions(c.matched, positions),
				})
			}
		}
		candidates = next
	}

	p.results = make([]PaletteResult, len(candidates))
	for i, c := range candidates {
		p.results[i] = PaletteResult{
			Item:    p.items[c.idx],
			Matched: c.matched,
		}
	}
}

// mergePositions combines two sorted slices of match positions, deduplicating.
func mergePositions(a, b []int) []int {
	seen := make(map[int]struct{}, len(a)+len(b))
	for _, v := range a {
		seen[v] = struct{}{}
	}
	for _, v := range b {
		seen[v] = struct{}{}
	}
	out := make([]int, 0, len(seen))
	for v := range seen {
		out = append(out, v)
	}
	return out
}

// View renders the palette overlay.
func (p palette) View() string {
	if !p.visible || p.theme == nil {
		return ""
	}

	pal := p.theme.Palette
	w := p.paletteWidth()

	// mode tabs
	cmdTab := " Commands "
	fldTab := " Fields "
	if p.mode == PaletteModeCommands {
		cmdTab = p.theme.Primary.Bold(true).Render(cmdTab)
		fldTab = p.theme.Muted.Render(fldTab)
	} else {
		cmdTab = p.theme.Muted.Render(cmdTab)
		fldTab = p.theme.Primary.Bold(true).Render(fldTab)
	}
	tabs := cmdTab + p.theme.Muted.Render("│") + fldTab + p.theme.Muted.Render("  tab to switch")

	// input row
	prompt := p.theme.Primary.Render("> ")
	inputRow := prompt + p.input.View()

	// separator
	innerW := w - 4 // border (2) + padding (2)
	if innerW < 10 {
		innerW = 10
	}
	sep := p.theme.Muted.Render(strings.Repeat("─", innerW))

	// results
	var rows []string
	visible := p.results
	offset := 0
	if len(visible) > paletteMaxVisible {
		// keep selected in view
		if p.selected >= paletteMaxVisible {
			offset = p.selected - paletteMaxVisible + 1
		}
		visible = visible[offset:]
		if len(visible) > paletteMaxVisible {
			visible = visible[:paletteMaxVisible]
		}
	}

	for i, r := range visible {
		realIdx := offset + i
		rows = append(rows, p.renderResult(r, realIdx == p.selected, innerW))
	}

	if len(p.results) == 0 && p.input.Value() != "" {
		rows = append(rows, p.theme.Muted.Render("  no matches"))
	}

	// count indicator
	var countLine string
	if len(p.results) > paletteMaxVisible {
		countLine = p.theme.Muted.Render(
			strings.Repeat(" ", innerW-20) + // rough right-align
				paletteCountLabel(p.selected+1, len(p.results)),
		)
	}

	// assemble body
	var body strings.Builder
	body.WriteString(tabs)
	body.WriteByte('\n')
	body.WriteString(inputRow)
	body.WriteByte('\n')
	body.WriteString(sep)
	for _, row := range rows {
		body.WriteByte('\n')
		body.WriteString(row)
	}
	if countLine != "" {
		body.WriteByte('\n')
		body.WriteString(sep)
		body.WriteByte('\n')
		body.WriteString(countLine)
	}

	// border and positioning
	panel := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(pal.BorderFocus).
		Background(pal.Base).
		Foreground(pal.Text).
		Padding(0, 1).
		Width(w - 2). // subtract border chars
		Render(body.String())

	// center horizontally, position near top
	return lipgloss.Place(
		p.width, p.height,
		lipgloss.Center, lipgloss.Top,
		panel,
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Foreground(pal.Muted)),
		lipgloss.WithWhitespaceChars(" "),
	)
}

func (p palette) renderResult(r PaletteResult, selected bool, maxW int) string {
	// build match highlight set
	matchSet := make(map[int]struct{}, len(r.Matched))
	for _, pos := range r.Matched {
		matchSet[pos] = struct{}{}
	}

	// category prefix
	var cat string
	if r.Item.Category != "" {
		cat = p.theme.Muted.Render("["+r.Item.Category+"] ")
	}

	// label with highlighted match chars
	label := p.highlightMatches(r.Item.Label, r.Item.MatchTerms, matchSet, selected)

	// description
	var desc string
	if r.Item.Description != "" {
		desc = p.theme.Muted.Render(" — " + r.Item.Description)
	}

	// shortcut (right-aligned)
	var shortcut string
	if r.Item.Shortcut != "" {
		shortcut = "  " + p.theme.KeyCap.Render(r.Item.Shortcut)
	}

	line := cat + label + desc + shortcut

	// selection indicator
	if selected {
		prefix := p.theme.Primary.Render("> ")
		line = prefix + line
	} else {
		line = "  " + line
	}

	return lipgloss.NewStyle().Width(maxW).MaxWidth(maxW).Render(line)
}

// highlightMatches renders label text, coloring characters whose positions
// in matchTerms correspond to matched indices.
func (p palette) highlightMatches(label, matchTerms string, matchSet map[int]struct{}, selected bool) string {
	if len(matchSet) == 0 {
		if selected {
			return p.theme.Text.Bold(true).Render(label)
		}
		return p.theme.Subtext.Render(label)
	}

	// map label chars to matchTerms positions
	lowerLabel := strings.ToLower(label)
	lowerTerms := strings.ToLower(matchTerms)
	labelOffset := strings.Index(lowerTerms, lowerLabel)
	if labelOffset < 0 {
		labelOffset = 0
	}

	// batch consecutive characters with the same style class
	matchStyle := p.theme.Accent.Bold(true)
	var normalStyle lipgloss.Style
	if selected {
		normalStyle = p.theme.Text.Bold(true)
	} else {
		normalStyle = p.theme.Subtext
	}

	runes := []rune(label)
	var b strings.Builder
	var run strings.Builder
	runIsMatch := false

	flush := func() {
		if run.Len() == 0 {
			return
		}
		if runIsMatch {
			b.WriteString(matchStyle.Render(run.String()))
		} else {
			b.WriteString(normalStyle.Render(run.String()))
		}
		run.Reset()
	}

	for i, ch := range runes {
		_, isMatch := matchSet[labelOffset+i]
		if i > 0 && isMatch != runIsMatch {
			flush()
		}
		runIsMatch = isMatch
		run.WriteRune(ch)
	}
	flush()
	return b.String()
}

func (p palette) paletteWidth() int {
	w := p.width * 60 / 100
	if w < 40 {
		w = 40
	}
	if w > 80 {
		w = 80
	}
	return w
}

func paletteCountLabel(current, total int) string {
	return formatCount(current, total)
}
