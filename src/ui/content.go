package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/emin/konfigurator/konfables"
	"github.com/emin/konfigurator/pkg"
	"github.com/emin/konfigurator/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// field type icons (nerd font glyphs)
var fieldTypeIcon = map[string]string{
	"string": "\uf031",  //
	"number": "\uf292",  //
	"bool":   "\uf444",  //
	"enum":   "\uf150",  //
	"color":  "\uf53f",  //
}

type content struct {
	title    string
	konfable konfables.Konfable
	config   *pkg.ConfigFile
	schema   *pkg.Schema
	values   map[string]string
	cursor   int
	fields  []pkg.Field // flattened field list across all sections
	scrollY int
	focused bool
	width   int
	height  int
	theme   *theme.Theme
	program *tea.Program

	// version filtering
	versions map[string]string

	// inline editing
	editing     bool
	editor      FieldEditor
	editField   int    // index into c.fields
	editOrigVal string // for cancel restoration

	// preview pane state
	previewLine  int
	previewFound bool
	previewKey   string

	// app-level docs fallback
	docsURL string

	// insight cycling + split-flap animation
	insightIdx   int
	insightLines []string
	insightGen   int
	splitFlap    *splitFlapState
}

func newContent(th *theme.Theme) content {
	return content{
		title:  "konfigurator",
		values: make(map[string]string),
		theme:  th,
	}
}

func (c content) Update(msg tea.Msg) (content, tea.Cmd) {
	// when editing, forward all messages to the active editor
	if c.editing && c.editor != nil {
		cmd, done, canceled := c.editor.Update(msg)
		if done {
			if canceled {
				c.editing = false
				c.editor = nil
			} else {
				c.commitEdit(c.editor.Value())
			}
		}
		return c, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !c.focused {
			return c, nil
		}
		hasFields := c.schema != nil && len(c.fields) > 0

		switch msg.String() {
		case "enter":
			if hasFields && c.cursor >= 0 && c.cursor < len(c.fields) {
				f := c.fields[c.cursor]
				if f.Type == "bool" {
					c.toggleBool(f)
					return c, nil
				}
				cmd := c.openEditor()
				return c, cmd
			}
		case "j", "down":
			if hasFields {
				if c.cursor < len(c.fields)-1 {
					c.cursor++
				}
				c.updatePreviewLine()
			} else {
				c.scrollY++
			}
		case "k", "up":
			if hasFields {
				if c.cursor > 0 {
					c.cursor--
				}
				c.updatePreviewLine()
			} else if c.scrollY > 0 {
				c.scrollY--
			}
		case "home":
			if hasFields {
				c.cursor = 0
				c.updatePreviewLine()
			} else {
				c.scrollY = 0
			}
		case "end":
			if hasFields {
				c.cursor = len(c.fields) - 1
				c.updatePreviewLine()
			}
		case "pgdown":
			page := c.pageSize()
			if hasFields {
				c.cursor += page
				if c.cursor >= len(c.fields) {
					c.cursor = len(c.fields) - 1
				}
				c.updatePreviewLine()
			} else {
				c.scrollY += page
			}
		case "pgup":
			page := c.pageSize()
			if hasFields {
				c.cursor -= page
				if c.cursor < 0 {
					c.cursor = 0
				}
				c.updatePreviewLine()
			} else {
				c.scrollY -= page
				if c.scrollY < 0 {
					c.scrollY = 0
				}
			}
		default:
			// type-through: printable chars seed a new editor in replace mode
			if hasFields && len(msg.Runes) > 0 {
				f := c.fields[c.cursor]
				if f.Type != "bool" && f.Type != "enum" {
					cmd := c.openEditorWithSeed(msg.Runes[0])
					return c, cmd
				}
			}
		}

	case ThemeChangedMsg:
		c.theme = msg.Theme

	case ExternalChangeMsg:
		if c.config != nil && c.config.Path == msg.Path {
			if err := c.config.Reload(); err == nil {
				c.refreshValues()
			}
		}

	case insightTickMsg:
		if msg.gen != c.insightGen {
			return c, nil
		}
		if len(c.insightLines) > 1 {
			c.insightIdx = (c.insightIdx + 1) % len(c.insightLines)
		}
		return c, c.insightTickCmd()

	case splitFlapTickMsg:
		if c.splitFlap == nil || msg.gen != c.splitFlap.gen {
			return c, nil
		}
		if c.splitFlap.tick() {
			c.splitFlap = nil
			return c, nil
		}
		return c, splitFlapCmd(c.splitFlap.gen)
	}

	return c, nil
}

// openEditor creates and initializes an editor for the current cursor field.
func (c *content) openEditor() tea.Cmd {
	if c.konfable == nil || c.config == nil || c.konfable.Parser() == nil {
		return nil
	}

	f := c.fields[c.cursor]
	c.editField = c.cursor
	c.editOrigVal = c.values[f.Key]

	e := editorForField(f)
	c.editor = e
	c.editing = true
	return e.Init(f, c.editOrigVal, c.theme)
}

// openEditorWithSeed starts the editor in replace mode (empty value) and
// injects the seed rune as the first keystroke.
func (c *content) openEditorWithSeed(seed rune) tea.Cmd {
	if c.konfable == nil || c.config == nil || c.konfable.Parser() == nil {
		return nil
	}

	f := c.fields[c.cursor]
	c.editField = c.cursor
	c.editOrigVal = c.values[f.Key]

	e := editorForField(f)
	c.editor = e
	c.editing = true

	initCmd := e.Init(f, "", c.theme)
	seedCmd := func() tea.Msg {
		return tea.KeyMsg{Runes: []rune{seed}, Type: tea.KeyRunes}
	}
	return tea.Sequence(initCmd, seedCmd)
}

// commitEdit writes the edited value back to the config.
func (c *content) commitEdit(value string) {
	c.editing = false
	c.editor = nil

	if c.konfable == nil || c.config == nil || c.konfable.Parser() == nil {
		return
	}

	f := c.fields[c.editField]
	serialized := formatValue(value, f.Type, c.konfable.Info().Format)

	data := c.config.Content()
	newData, err := c.konfable.Parser().SetValue(data, f.Key, serialized)
	if err != nil {
		return
	}
	c.config.SetContent(newData)
	c.refreshValues()
}

// toggleBool flips a boolean field value immediately.
func (c *content) toggleBool(f pkg.Field) {
	if c.konfable == nil || c.config == nil || c.konfable.Parser() == nil {
		return
	}

	cur := c.values[f.Key]
	if cur == "" {
		cur = f.Default
	}
	next := "true"
	if cur == "true" {
		next = "false"
	}
	serialized := formatValue(next, f.Type, c.konfable.Info().Format)

	data := c.config.Content()
	newData, err := c.konfable.Parser().SetValue(data, f.Key, serialized)
	if err != nil {
		return
	}
	c.config.SetContent(newData)
	c.refreshValues()
}

// loadApp sets the active konfable, loads its config and schema, and reads values.
func (c *content) loadApp(k konfables.Konfable) tea.Cmd {
	// snapshot current header lines for split-flap transition
	var snapshot []string
	if c.splitFlap != nil && !c.splitFlap.done {
		snapshot = make([]string, len(c.splitFlap.current))
		copy(snapshot, c.splitFlap.current)
	} else if c.insightLines != nil && c.konfable != nil {
		snapshot = c.headerLeftLines()
	}

	// stop watching previous config
	if c.config != nil {
		c.config.StopWatching()
	}

	c.konfable = k
	c.title = k.Name()
	c.scrollY = 0
	c.cursor = 0
	c.fields = nil
	c.values = make(map[string]string)
	c.config = nil
	c.schema = nil
	c.editing = false
	c.editor = nil
	c.previewLine = -1
	c.previewFound = false
	c.previewKey = ""
	c.docsURL = ""

	info := k.Info()

	// load config file
	cf, err := pkg.LoadConfigFile(info.ConfigPath)
	if err != nil {
		return func() tea.Msg {
			return StatusMsg{Text: fmt.Sprintf("no config: %s", info.ConfigPath)}
		}
	}
	c.config = cf

	// load schema (filter by detected version if known)
	schemaData, err := k.Schema()
	if err == nil && schemaData != nil {
		s, err := pkg.LoadSchema(schemaData)
		if err == nil {
			if v, ok := c.versions[k.Name()]; ok {
				s = s.FilterByVersion(v)
			}
			c.schema = s
			c.docsURL = s.DocsURL
			c.buildFieldList()
		}
	}

	c.refreshValues()
	c.buildInsights()
	c.insightGen++

	// start file watching
	path := info.ConfigPath
	if c.config != nil && c.program != nil {
		p := c.program
		cfPath := c.config.Path
		_ = c.config.StartWatching(func() {
			p.Send(ExternalChangeMsg{Path: cfPath})
		})
	}

	var cmds []tea.Cmd
	cmds = append(cmds, func() tea.Msg {
		return StatusMsg{Text: path}
	})

	// init split-flap animation if we have a previous snapshot
	if snapshot != nil {
		c.splitFlap = newSplitFlap(snapshot, c.headerLeftLines(), c.insightGen)
		cmds = append(cmds, splitFlapCmd(c.insightGen))
	}
	cmds = append(cmds, c.insightTickCmd())

	return tea.Batch(cmds...)
}

func (c *content) buildFieldList() {
	c.fields = nil
	if c.schema == nil {
		return
	}
	for _, sec := range c.schema.Sections {
		c.fields = append(c.fields, sec.Fields...)
	}
}

func (c *content) refreshValues() {
	c.values = make(map[string]string)
	if c.config == nil || c.schema == nil || c.konfable == nil {
		return
	}

	p := c.konfable.Parser()
	if p == nil {
		return
	}

	data := c.config.Content()
	for _, sec := range c.schema.Sections {
		for i := range sec.Fields {
			if v, ok := p.FindValue(data, sec.Fields[i].Key); ok {
				c.values[sec.Fields[i].Key] = v
			}
		}
	}

	c.previewKey = "" // force re-scan
	c.updatePreviewLine()
	c.buildInsights()
}

func (c *content) updatePreviewLine() {
	if c.config == nil || c.konfable == nil || c.konfable.Parser() == nil ||
		len(c.fields) == 0 || c.cursor < 0 || c.cursor >= len(c.fields) {
		c.previewLine = -1
		c.previewFound = false
		c.previewKey = ""
		return
	}
	f := c.fields[c.cursor]
	if f.Key == c.previewKey {
		return
	}
	c.previewKey = f.Key
	c.previewLine, c.previewFound = c.konfable.Parser().FindLine(c.config.Content(), f.Key)
}

// splitHeights computes the field list and preview pane heights for a given inner height.
// returns previewH=0 when preview should be hidden.
func (c content) splitHeights(innerH int) (fieldH, previewH int) {
	if c.schema == nil || c.config == nil || len(c.fields) == 0 {
		return innerH, 0
	}
	previewH = innerH * 2 / 5
	if previewH < 3 {
		previewH = 3
	}
	fieldH = innerH - previewH - 1
	if fieldH < 5 {
		fieldH = 5
		previewH = innerH - fieldH - 1
	}
	if previewH < 1 {
		return innerH, 0
	}
	return fieldH, previewH
}

func (c content) fieldListHeight() int {
	innerH := c.height - 2 - 2
	if innerH < 3 {
		innerH = 3
	}
	fh, _ := c.splitHeights(innerH)
	return fh
}

func (c content) pageSize() int {
	p := c.fieldListHeight() - 1
	if p < 1 {
		p = 1
	}
	return p
}

// headerHeight returns the fixed number of rendered lines the header occupies.
// must be constant regardless of rendering mode — cursorLine() depends on it.
func (c content) headerHeight() int {
	return 8
}

// cursorLine returns the rendered line number for the current cursor field.
// walks sections with headers to account for divider lines.
func (c content) cursorLine() int {
	if c.schema == nil || len(c.fields) == 0 {
		return 0
	}
	line := c.headerHeight()
	fieldIdx := 0
	for si, sec := range c.schema.Sections {
		// blank line before section (except first)
		if si > 0 {
			line++
		}
		// section header line
		line++
		for range sec.Fields {
			if fieldIdx == c.cursor {
				return line
			}
			line++
			if c.editing && c.editor != nil && fieldIdx == c.editField {
				line += c.editor.Height()
			}
			fieldIdx++
		}
	}
	return 0
}

func (c content) View() string {
	style := c.theme.Content
	if c.focused {
		style = style.BorderForeground(c.theme.Palette.BorderFocus)
	}

	// inner dimensions (border=2, padding is in the style)
	innerW := c.width - 2 - 4 // 2 border + 4 padding (2 each side)
	innerH := c.height - 2 - 2 // 2 border + 2 padding (1 each side)
	if innerW < 10 {
		innerW = 10
	}
	if innerH < 3 {
		innerH = 3
	}

	fieldListH, previewH := c.splitHeights(innerH)
	showPreview := previewH > 0

	// auto-scroll to keep cursor visible (schema mode)
	if c.schema != nil && len(c.fields) > 0 {
		cl := c.cursorLine()
		visibleEnd := cl
		if c.editing && c.editor != nil {
			visibleEnd = cl + c.editor.Height()
		}
		if cl < c.scrollY {
			c.scrollY = cl
		}
		if visibleEnd >= c.scrollY+fieldListH {
			c.scrollY = visibleEnd - fieldListH + 1
		}
	}

	body := c.renderBody(innerW)

	// apply scrolling
	lines := strings.Split(body, "\n")
	if c.scrollY >= len(lines) {
		c.scrollY = max(0, len(lines)-1)
	}
	if c.scrollY > 0 && c.scrollY < len(lines) {
		lines = lines[c.scrollY:]
	}

	// trim to fit
	if len(lines) > fieldListH {
		lines = lines[:fieldListH]
	}

	fieldView := strings.Join(lines, "\n")

	if !showPreview {
		style = style.
			Width(c.width - 2).
			Height(c.height - 2).
			Align(lipgloss.Left, lipgloss.Top)
		return style.Render(fieldView)
	}

	// pad field list to exact height
	fieldLines := strings.Count(fieldView, "\n") + 1
	for fieldLines < fieldListH {
		fieldView += "\n"
		fieldLines++
	}

	sep := c.theme.Muted.Render(strings.Repeat("─", innerW))
	preview := c.renderPreview(innerW, previewH)
	combined := fieldView + "\n" + sep + "\n" + preview

	style = style.
		Width(c.width - 2).
		Height(c.height - 2).
		Align(lipgloss.Left, lipgloss.Top)

	return style.Render(combined)
}

// labelColumnWidth computes the max label width for the active section.
func (c content) labelColumnWidth() int {
	w := 0
	for i := range c.fields {
		if len(c.fields[i].Label) > w {
			w = len(c.fields[i].Label)
		}
	}
	return w
}

// buildInsights computes the cycling insight lines from current state.
func (c *content) buildInsights() {
	c.insightLines = nil
	c.insightIdx = 0

	if c.schema == nil {
		return
	}

	totalFields := 0
	for _, sec := range c.schema.Sections {
		totalFields += len(sec.Fields)
	}

	configured := 0
	for _, v := range c.values {
		if v != "" {
			configured++
		}
	}

	sections := len(c.schema.Sections)
	stat := fmt.Sprintf("%d/%d fields configured across %d sections", configured, totalFields, sections)
	c.insightLines = append(c.insightLines, stat)
	c.insightLines = append(c.insightLines, c.schema.Hints...)
}

// headerLeftLines returns the 4-line left column for the header.
func (c content) headerLeftLines() []string {
	version := ""
	if c.konfable != nil {
		if v, ok := c.versions[c.konfable.Name()]; ok && v != "" {
			version = v
		}
	}

	path := ""
	if c.config != nil {
		path = c.config.Path
	}

	insight := ""
	if len(c.insightLines) > 0 {
		insight = c.insightLines[c.insightIdx%len(c.insightLines)]
	}

	return []string{version, path, "", insight}
}

// renderHeader produces the two-column header or narrow fallback.
func (c content) renderHeader(width int) string {
	hh := c.headerHeight()

	if c.konfable == nil {
		// no app selected — empty header padded to height
		lines := make([]string, hh)
		for i := range lines {
			lines[i] = ""
		}
		return strings.Join(lines, "\n") + "\n"
	}

	// build right column: logo + name badge
	var rightLines []string
	if logo, ok := konfables.Logos[c.konfable.Name()]; ok {
		art := logo.Render()
		rightLines = strings.Split(art, "\n")
	}
	nameBadge := c.theme.Badge.Render(c.konfable.Name())
	rightLines = append(rightLines, nameBadge)
	rightBlock := strings.Join(rightLines, "\n")
	rightW := 0
	for _, l := range rightLines {
		if w := lipgloss.Width(l); w > rightW {
			rightW = w
		}
	}

	leftW := width - rightW - 2 // 2 chars gap
	if leftW < 20 {
		// narrow fallback: centered logo + name
		var lines []string
		if logo, ok := konfables.Logos[c.konfable.Name()]; ok {
			art := logo.Render()
			lines = append(lines, strings.Split(centerBlock(art, width), "\n")...)
		}
		lines = append(lines, centerLine(nameBadge, width), "") // name + blank after
		for len(lines) < hh {
			lines = append(lines, "")
		}
		if len(lines) > hh {
			lines = lines[:hh]
		}
		return strings.Join(lines, "\n") + "\n"
	}

	// two-column: build left lines
	leftData := c.headerLeftLines()
	if c.splitFlap != nil && !c.splitFlap.done {
		// replace with split-flap animation frames
		leftData = make([]string, len(c.splitFlap.current))
		copy(leftData, c.splitFlap.current)
	}

	// style + truncate left lines
	styledLeft := make([]string, len(leftData))
	styles := []lipgloss.Style{c.theme.Subtext, c.theme.Muted, c.theme.Text, c.theme.InsightText}
	for i, line := range leftData {
		// truncate to leftW (plain text before styling)
		if len(line) > leftW {
			line = line[:leftW]
		}
		s := c.theme.Text
		if i < len(styles) {
			s = styles[i]
		}
		styledLeft[i] = s.Render(line)
	}

	// pad left lines to headerHeight
	for len(styledLeft) < hh {
		styledLeft = append(styledLeft, "")
	}

	// build left block with fixed width for alignment
	leftStyle := lipgloss.NewStyle().Width(leftW)
	leftBlock := leftStyle.Render(strings.Join(styledLeft[:hh], "\n"))

	// right-align the right column
	rightStyle := lipgloss.NewStyle().Width(rightW + 2).Align(lipgloss.Right)
	rightAligned := rightStyle.Render(rightBlock)

	joined := lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, rightAligned)

	// pad output to exactly headerHeight rows
	outLines := strings.Split(joined, "\n")
	for len(outLines) < hh {
		outLines = append(outLines, "")
	}
	if len(outLines) > hh {
		outLines = outLines[:hh]
	}

	return strings.Join(outLines, "\n") + "\n"
}

// insightTickCmd starts the next insight cycle timer.
func (c content) insightTickCmd() tea.Cmd {
	gen := c.insightGen
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg {
		return insightTickMsg{gen: gen}
	})
}

func (c content) renderBody(width int) string {
	if c.schema == nil {
		if c.config != nil {
			return c.theme.Text.Render(string(c.config.Content()))
		}
		return c.theme.Muted.Render("select an app to view its configuration")
	}

	var b strings.Builder

	// header: two-column (or narrow fallback)
	b.WriteString(c.renderHeader(width))

	// section headers + column-aligned field table
	labelW := c.labelColumnWidth()
	fieldIdx := 0
	for si, sec := range c.schema.Sections {
		// blank line before section (except first)
		if si > 0 {
			b.WriteByte('\n')
		}
		// section header
		header := c.theme.Subtext.Bold(true).Render(sec.Name)
		b.WriteString(header)
		b.WriteByte('\n')

		for fi := range sec.Fields {
			f := &sec.Fields[fi]
			isCursor := c.focused && fieldIdx == c.cursor

			// type icon
			icon := fieldTypeIcon[f.Type]
			if icon == "" {
				icon = " "
			}

			// value rendering
			val, hasVal := c.values[f.Key]
			var renderedVal string
			if !hasVal {
				val = f.Default
				renderedVal = c.renderFieldValue(*f, val, true)
			} else {
				renderedVal = c.renderFieldValue(*f, val, false)
			}

			// inline min/max bounds for number fields
			if f.Type == "number" && (f.Min != nil || f.Max != nil) {
				lo, hi := "*", "*"
				if f.Min != nil {
					lo = formatNum(*f.Min)
				}
				if f.Max != nil {
					hi = formatNum(*f.Max)
				}
				renderedVal += c.theme.Muted.Render(fmt.Sprintf(" (%s\u2013%s)", lo, hi))
			}

			// label + prefix
			paddedLabel := fmt.Sprintf("%-*s", labelW, f.Label)
			var line string
			if isCursor {
				prefix := c.theme.Primary.Render("▸ " + icon + " ")
				label := c.theme.Text.Bold(true).Render(paddedLabel)
				line = prefix + label + "  " + renderedVal
			} else {
				prefix := "  " + c.theme.Muted.Render(icon) + " "
				label := c.theme.FieldLabel.Render(paddedLabel)
				line = prefix + label + "  " + renderedVal
			}

			b.WriteString(line)
			b.WriteByte('\n')

			// inline editor below cursor row
			if c.editing && c.editor != nil && fieldIdx == c.editField {
				b.WriteString(c.editor.View(width))
				b.WriteByte('\n')
			}
			fieldIdx++
		}
	}

	return b.String()
}

func (c content) renderPreview(width, height int) string {
	if c.config == nil || len(c.fields) == 0 {
		return c.theme.Muted.Render("no preview")
	}

	var b strings.Builder

	if c.focused {
		f := c.fields[c.cursor]

		// field description
		if f.Description != "" {
			b.WriteString(c.theme.Muted.Render(f.Description))
			b.WriteByte('\n')
			height--
		}

		// example value
		if f.Example != "" && height > 1 {
			label := c.theme.Subtext.Render("example: ")
			val := c.theme.Muted.Italic(true).Render(f.Example)
			b.WriteString(label + val)
			b.WriteByte('\n')
			height--
		}

		// hint
		if f.Hint != "" && height > 1 {
			label := c.theme.Subtext.Render("hint: ")
			val := c.theme.Muted.Italic(true).Render(f.Hint)
			b.WriteString(label + val)
			b.WriteByte('\n')
			height--
		}

		// doc link (field-specific or app-level fallback)
		docLink := f.DocURL
		docLabel := "doc: "
		if docLink == "" && c.docsURL != "" {
			docLink = c.docsURL
			docLabel = "docs: "
		}
		if docLink != "" && height > 1 {
			label := c.theme.Subtext.Render(docLabel)
			link := c.theme.Secondary.Underline(true).Render(docLink)
			b.WriteString(label + link)
			b.WriteByte('\n')
			height--
		}
	}

	// file path + line number
	pathLine := c.config.Path
	if c.focused && c.previewFound {
		pathLine += fmt.Sprintf(":%d", c.previewLine+1)
	}
	b.WriteString(c.theme.Subtext.Render(pathLine))
	b.WriteByte('\n')
	height--

	if height < 1 {
		return b.String()
	}

	if !c.focused {
		// show app-level docs link when unfocused
		if c.docsURL != "" && height > 1 {
			label := c.theme.Subtext.Render("docs: ")
			link := c.theme.Secondary.Underline(true).Render(c.docsURL)
			b.WriteString(label + link)
			b.WriteByte('\n')
			height--
		}
		return b.String()
	}

	data := c.config.Content()
	rawLines := strings.Split(string(data), "\n")

	f := c.fields[c.cursor]
	if !c.previewFound {
		val := f.Default
		if v, ok := c.values[f.Key]; ok {
			val = v
		}
		b.WriteString(c.theme.Success.Render(fmt.Sprintf("+ %s = %s  (will be added)", f.Key, val)))
		return b.String()
	}

	// center snippet window on previewLine
	startLine := c.previewLine - height/2
	if startLine < 0 {
		startLine = 0
	}
	endLine := startLine + height
	if endLine > len(rawLines) {
		endLine = len(rawLines)
		startLine = endLine - height
		if startLine < 0 {
			startLine = 0
		}
	}

	for i := startLine; i < endLine; i++ {
		line := rawLines[i]
		maxW := width - 2
		if lipgloss.Width(line) > maxW {
			line = line[:maxW]
		}

		if i == c.previewLine {
			b.WriteString(c.theme.PreviewHL.Render("▶ " + line))
		} else {
			b.WriteString(c.theme.Muted.Render("  " + line))
		}
		if i < endLine-1 {
			b.WriteByte('\n')
		}
	}

	return b.String()
}

// renderFieldValue renders a field value with type-specific formatting.
func (c content) renderFieldValue(f pkg.Field, val string, isDefault bool) string {
	if isDefault {
		switch f.Type {
		case "bool":
			if val == "true" {
				return c.theme.FieldDefault.Render("● true")
			}
			return c.theme.FieldDefault.Render("○ false")
		case "color":
			swatch := lipgloss.NewStyle().
				Background(lipgloss.Color(normalizeHex(val))).
				Render("██")
			return swatch + " " + c.theme.FieldDefault.Render(val)
		default:
			return c.theme.FieldDefault.Render(val)
		}
	}

	switch f.Type {
	case "bool":
		if val == "true" {
			return c.theme.Success.Render("●") + " " + c.theme.FieldValue.Render("true")
		}
		return c.theme.Muted.Render("○") + " " + c.theme.FieldValue.Render("false")
	case "color":
		swatch := lipgloss.NewStyle().
			Background(lipgloss.Color(normalizeHex(val))).
			Render("██")
		return swatch + " " + c.theme.FieldValue.Render(val)
	default:
		return c.theme.FieldValue.Render(val)
	}
}

// centerBlock centers each line of a multi-line string within the given width.
func centerBlock(block string, width int) string {
	lines := strings.Split(block, "\n")
	for i, line := range lines {
		lines[i] = centerLine(line, width)
	}
	return strings.Join(lines, "\n")
}

// centerLine centers a single line within the given width using lipgloss.
func centerLine(line string, width int) string {
	w := lipgloss.Width(line)
	if w >= width {
		return line
	}
	pad := (width - w) / 2
	return strings.Repeat(" ", pad) + line
}
