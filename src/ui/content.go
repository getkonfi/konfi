package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/emin/konfigurator/konfables"
	"github.com/emin/konfigurator/pkg"
	"github.com/emin/konfigurator/theme"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// glamourCache holds the glamour renderer, rebuilt on width or theme change.
type glamourCache struct {
	renderer *glamour.TermRenderer
	width    int
}

// field type icons (nerd font glyphs)
var fieldTypeIcon = map[string]string{
	"string": "\uf031",  //
	"number": "\uf292",  //
	"bool":   "\uf444",  //
	"enum":   "\uf150",  //
	"color":  "\uf53f",  //
	"list":   "\uf03a",  //
	"multi":  "\uf046",  //
}

// row represents a navigable item in the field list — either a section header or a field.
type row struct {
	isSection  bool
	sectionIdx int // index into schema.Sections
	fieldIdx   int // index into c.fields (-1 for section rows)
}

type content struct {
	title    string
	konfable konfables.Konfable
	config   *pkg.ConfigFile
	schema   *pkg.Schema
	values   map[string]string
	cursor   int
	fields       []pkg.Field // flattened field list across all sections
	fieldSection []int       // len == len(c.fields), maps field → section index
	visible      []row       // navigable rows (section headers + fields)
	collapsed      map[int]bool // section index → collapsed
	activeSection  int          // current section tab
	configuredOnly bool
	searching      bool
	search         textinput.Model
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

	// glamour markdown renderer cache (pointer survives value-receiver copies)
	glamCache *glamourCache

	// file state indicator ("", "unsaved", "saved", "reloaded", "new")
	fileState string

	// keyboard hints (set by root.updateHints)
	hints []keyHint

	// insight cycling + split-flap animation
	insightIdx          int
	insightLines        []string
	insightWarningCount int
	insightGen          int
	splitFlap           *splitFlapState
}

func newContent(th *theme.Theme) content {
	ti := textinput.New()
	ti.Placeholder = "search..."
	ti.CharLimit = 64
	ti.Prompt = ""

	return content{
		title:     "konfigurator",
		values:    make(map[string]string),
		collapsed: make(map[int]bool),
		search:    ti,
		theme:     th,
		glamCache: &glamourCache{},
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

	// when searching, forward keys to search textinput
	if c.searching {
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "esc":
				c.searching = false
				c.search.SetValue("")
				c.search.Blur()
				c.refilter()
				c.updatePreviewLine()
				return c, nil
			case "enter":
				// lock filter and exit search mode
				c.searching = false
				c.search.Blur()
				return c, nil
			case "j", "down":
				if c.cursor < len(c.visible)-1 {
					c.cursor++
				}
				c.updatePreviewLine()
				return c, nil
			case "k", "up":
				if c.cursor > 0 {
					c.cursor--
				}
				c.updatePreviewLine()
				return c, nil
			default:
				var cmd tea.Cmd
				c.search, cmd = c.search.Update(msg)
				c.refilter()
				c.updatePreviewLine()
				return c, cmd
			}
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !c.focused {
			return c, nil
		}
		hasRows := c.schema != nil && len(c.visible) > 0

		switch msg.String() {
		case "enter":
			if f := c.currentField(); f != nil {
				if f.Type == "bool" {
					c.toggleBool(*f)
					return c, nil
				}
				cmd := c.openEditor()
				return c, cmd
			}
		case "[":
			if c.schema != nil && len(c.schema.Sections) > 1 {
				c.activeSection--
				if c.activeSection < 0 {
					c.activeSection = len(c.schema.Sections) - 1
				}
				c.cursor = 0
				c.scrollY = 0
				c.refilter()
				c.updatePreviewLine()
			}
		case "]":
			if c.schema != nil && len(c.schema.Sections) > 1 {
				c.activeSection++
				if c.activeSection >= len(c.schema.Sections) {
					c.activeSection = 0
				}
				c.cursor = 0
				c.scrollY = 0
				c.refilter()
				c.updatePreviewLine()
			}
		case "f":
			if c.schema != nil {
				c.configuredOnly = !c.configuredOnly
				c.refilter()
				c.updatePreviewLine()
			}
		case "/":
			if c.schema != nil {
				c.searching = true
				c.search.SetValue("")
				return c, c.search.Focus()
			}
		case "j", "down":
			if hasRows {
				if c.cursor < len(c.visible)-1 {
					c.cursor++
				}
				c.updatePreviewLine()
			} else {
				c.scrollY++
			}
		case "k", "up":
			if hasRows {
				if c.cursor > 0 {
					c.cursor--
				}
				c.updatePreviewLine()
			} else if c.scrollY > 0 {
				c.scrollY--
			}
		case "home":
			if hasRows {
				c.cursor = 0
				c.updatePreviewLine()
			} else {
				c.scrollY = 0
			}
		case "end":
			if hasRows {
				c.cursor = len(c.visible) - 1
				c.updatePreviewLine()
			}
		case "pgdown":
			page := c.pageSize()
			if hasRows {
				c.cursor += page
				if c.cursor >= len(c.visible) {
					c.cursor = len(c.visible) - 1
				}
				c.updatePreviewLine()
			} else {
				c.scrollY += page
			}
		case "pgup":
			page := c.pageSize()
			if hasRows {
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
		case "o":
			if url := c.currentDocURL(); url != "" {
				return c, c.openDocs(url)
			}
		default:
			// type-through: printable chars seed a new editor in replace mode
			if f := c.currentField(); f != nil && len(msg.Runes) > 0 {
				if f.Type != "bool" && f.Type != "enum" && f.Type != "list" && f.Type != "multi" {
					cmd := c.openEditorWithSeed(msg.Runes[0])
					return c, cmd
				}
			}
		}

	case ThemeChangedMsg:
		c.theme = msg.Theme
		if c.glamCache != nil {
			c.glamCache.renderer = nil
		}

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
	f := c.currentField()
	if f == nil || c.konfable == nil || c.config == nil || c.konfable.Parser() == nil {
		return nil
	}

	c.editField = c.visible[c.cursor].fieldIdx
	c.editOrigVal = c.values[f.Key]

	// for list fields, pass the actual multi-values (newline-joined)
	initVal := c.editOrigVal
	if f.Type == "list" {
		if mvp, ok := c.konfable.Parser().(konfables.MultiValueParser); ok {
			if vals, found := mvp.FindValues(c.config.Content(), f.Key); found {
				initVal = strings.Join(vals, "\n")
			} else {
				initVal = ""
			}
		}
	}

	e := editorForField(*f)
	c.editor = e
	c.editing = true
	return e.Init(*f, initVal, c.theme)
}

// openEditorWithSeed starts the editor in replace mode (empty value) and
// injects the seed rune as the first keystroke.
func (c *content) openEditorWithSeed(seed rune) tea.Cmd {
	f := c.currentField()
	if f == nil || c.konfable == nil || c.config == nil || c.konfable.Parser() == nil {
		return nil
	}

	c.editField = c.visible[c.cursor].fieldIdx
	c.editOrigVal = c.values[f.Key]

	e := editorForField(*f)
	c.editor = e
	c.editing = true

	initCmd := e.Init(*f, "", c.theme)
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
	data := c.config.Content()

	// list fields use MultiValueParser
	if f.Type == "list" {
		if mvp, ok := c.konfable.Parser().(konfables.MultiValueParser); ok {
			var vals []string
			if value != "" {
				vals = strings.Split(value, "\n")
			}
			newData, err := mvp.SetValues(data, f.Key, vals)
			if err != nil {
				return
			}
			c.config.SetContent(newData)
			c.refreshValues()
			c.emitSettingChanged(f.Key, value)
			return
		}
	}

	serialized := formatValue(value, f.Type, c.konfable.Info().Format)
	newData, err := c.konfable.Parser().SetValue(data, f.Key, serialized)
	if err != nil {
		return
	}
	c.config.SetContent(newData)
	c.refreshValues()

	// hot-reload konfigurator settings
	c.emitSettingChanged(f.Key, value)
}

// emitSettingChanged sends a KonfSettingChangedMsg if editing konfigurator.
func (c *content) emitSettingChanged(key, value string) {
	if c.konfable == nil || c.konfable.Name() != "konfigurator" || c.program == nil {
		return
	}
	c.program.Send(KonfSettingChangedMsg{Key: key, Value: value})
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

// showNotInstalled sets the active konfable for display without loading config or schema.
func (c *content) showNotInstalled(k konfables.Konfable) {
	if c.config != nil {
		c.config.StopWatching()
	}
	c.konfable = k
	c.title = k.Name()
	c.config = nil
	c.schema = nil
	c.fields = nil
	c.fieldSection = nil
	c.visible = nil
	c.collapsed = make(map[int]bool)
	c.activeSection = 0
	c.configuredOnly = false
	c.searching = false
	c.search.SetValue("")
	c.search.Blur()
	c.values = make(map[string]string)
	c.scrollY = 0
	c.cursor = 0
	c.editing = false
	c.editor = nil
	c.previewLine = -1
	c.previewFound = false
	c.previewKey = ""
	c.docsURL = ""
	c.insightLines = nil
	c.insightIdx = 0
	c.insightGen++
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
	c.fieldSection = nil
	c.visible = nil
	c.collapsed = make(map[int]bool)
	c.activeSection = 0
	c.configuredOnly = false
	c.searching = false
	c.search.SetValue("")
	c.search.Blur()
	c.values = make(map[string]string)
	c.config = nil
	c.schema = nil
	c.editing = false
	c.editor = nil
	c.previewLine = -1
	c.previewFound = false
	c.previewKey = ""
	c.docsURL = ""
	if c.glamCache != nil {
		c.glamCache.renderer = nil
	}

	info := k.Info()

	// load config file (virtual konfables create defaults if missing)
	cf, err := pkg.LoadConfigFile(info.ConfigPath)
	if err != nil && info.Binary == "" {
		cf, err = pkg.LoadOrCreateConfigFile(info.ConfigPath, []byte("theme: catppuccin\nlog_level: info\n"))
		if err == nil {
			c.fileState = "new"
		}
	}
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
	if c.config != nil && c.program != nil {
		p := c.program
		cfPath := c.config.Path
		_ = c.config.StartWatching(func() {
			p.Send(ExternalChangeMsg{Path: cfPath})
		})
	}

	var cmds []tea.Cmd

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
	c.fieldSection = nil
	if c.schema == nil {
		return
	}
	for si, sec := range c.schema.Sections {
		for range sec.Fields {
			c.fieldSection = append(c.fieldSection, si)
		}
		c.fields = append(c.fields, sec.Fields...)
	}
	c.refilter()
}

// refilter rebuilds the visible row slice for the active section tab.
func (c *content) refilter() {
	c.visible = c.visible[:0]
	if c.schema == nil {
		return
	}

	// clamp activeSection
	if c.activeSection >= len(c.schema.Sections) {
		c.activeSection = 0
	}

	query := strings.ToLower(strings.TrimSpace(c.search.Value()))
	hasSearch := c.searching && query != ""

	for i := range c.fields {
		f := &c.fields[i]
		if c.fieldSection[i] != c.activeSection {
			continue
		}
		if c.configuredOnly {
			if _, ok := c.values[f.Key]; !ok {
				continue
			}
		}
		if hasSearch {
			label := strings.ToLower(f.Label)
			key := strings.ToLower(f.Key)
			if !strings.Contains(label, query) && !strings.Contains(key, query) {
				continue
			}
		}
		c.visible = append(c.visible, row{sectionIdx: c.activeSection, fieldIdx: i})
	}

	// clamp cursor
	if len(c.visible) == 0 {
		c.cursor = 0
	} else if c.cursor >= len(c.visible) {
		c.cursor = len(c.visible) - 1
	}
	if c.cursor < 0 {
		c.cursor = 0
	}
}

// currentField returns the field under the cursor, or nil if empty.
func (c *content) currentField() *pkg.Field {
	if len(c.visible) == 0 || c.cursor < 0 || c.cursor >= len(c.visible) {
		return nil
	}
	r := c.visible[c.cursor]
	if r.isSection {
		return nil
	}
	return &c.fields[r.fieldIdx]
}

// currentDocURL returns the best doc URL for the cursor position:
// field-specific doc_url, then app-level docsURL, or empty on section header.
func (c content) currentDocURL() string {
	f := c.currentField()
	if f == nil {
		return ""
	}
	if f.DocURL != "" {
		return f.DocURL
	}
	return c.docsURL
}

// openDocs launches the system browser for the given URL.
func (c content) openDocs(url string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		default:
			cmd = exec.Command("xdg-open", url)
		}
		_ = cmd.Start()
		return DocOpenedMsg{URL: url}
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
	mvp, hasMVP := p.(konfables.MultiValueParser)
	for _, sec := range c.schema.Sections {
		for i := range sec.Fields {
			f := &sec.Fields[i]
			if f.Type == "list" && hasMVP {
				if vals, ok := mvp.FindValues(data, f.Key); ok {
					c.values[f.Key] = fmt.Sprintf("%d values", len(vals))
				}
			} else if v, ok := p.FindValue(data, f.Key); ok {
				c.values[f.Key] = v
			}
		}
	}

	c.previewKey = "" // force re-scan
	c.updatePreviewLine()
	c.buildInsights()
}

func (c *content) updatePreviewLine() {
	f := c.currentField()
	if f == nil || c.config == nil || c.konfable == nil || c.konfable.Parser() == nil {
		c.previewLine = -1
		c.previewFound = false
		c.previewKey = ""
		return
	}
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

// headerHeight returns the number of rendered lines the header occupies.
// includes the search bar when active.
func (c content) headerHeight() int {
	h := 8
	if c.schema != nil && len(c.schema.Sections) > 1 {
		h++ // tab bar line
	}
	if c.searching {
		h++ // search bar line
	}
	return h
}

// cursorLine returns the rendered line number for the current cursor position.
// walks visible rows to account for section headers, blank lines, and editors.
func (c content) cursorLine() int {
	if c.schema == nil || len(c.visible) == 0 {
		return 0
	}
	line := c.headerHeight()
	for i, r := range c.visible {
		if i == c.cursor {
			return line
		}
		line++
		if c.editing && c.editor != nil && r.fieldIdx == c.editField {
			line += c.editor.Height()
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
	if c.schema != nil && len(c.visible) > 0 {
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

	sep := c.renderSeparator(innerW)
	preview := c.renderPreview(innerW, previewH)
	combined := fieldView + "\n" + sep + "\n" + preview

	style = style.
		Width(c.width - 2).
		Height(c.height - 2).
		Align(lipgloss.Left, lipgloss.Top)

	return style.Render(combined)
}

// renderSeparator draws the line between field list and preview with scroll info.
func (c content) renderSeparator(width int) string {
	if len(c.visible) == 0 {
		info := "0/0"
		lineW := width - len(info) - 2
		if lineW < 4 {
			lineW = 4
		}
		left := lineW / 2
		right := lineW - left
		return c.theme.Muted.Render(strings.Repeat("─", left) + " " + info + " " + strings.Repeat("─", right))
	}

	var parts []string

	// position indicator
	parts = append(parts, fmt.Sprintf("%d/%d", c.cursor+1, len(c.visible)))

	// configured filter badge
	if c.configuredOnly {
		parts = append(parts, "configured")
	}

	info := strings.Join(parts, " · ")
	infoW := lipgloss.Width(info)
	lineW := width - infoW - 2 // 2 spaces padding
	if lineW < 4 {
		lineW = 4
	}
	left := lineW / 2
	right := lineW - left
	return c.theme.Muted.Render(strings.Repeat("─", left) + " " + info + " " + strings.Repeat("─", right))
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
// linter warnings come first, then stats and schema hints.
func (c *content) buildInsights() {
	c.insightLines = nil
	c.insightIdx = 0
	c.insightWarningCount = 0

	if c.schema == nil {
		return
	}

	// linter warnings from Diagnose
	if c.config != nil && c.konfable != nil && c.konfable.Parser() != nil {
		keys := c.konfable.Parser().ListKeys(c.config.Content())
		version := ""
		if v, ok := c.versions[c.konfable.Name()]; ok {
			version = v
		}
		diags := pkg.Diagnose(keys, c.schema, version)
		for _, d := range diags {
			c.insightLines = append(c.insightLines, d.Message)
		}
		c.insightWarningCount = len(diags)
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

// headerLeftLines returns the left column lines for the header.
func (c content) headerLeftLines() []string {
	title := ""
	if c.konfable != nil {
		title = c.konfable.Name()
		if v, ok := c.versions[c.konfable.Name()]; ok && v != "" {
			title += " " + v
		}
	}

	path := ""
	if c.config != nil {
		path = c.config.Path
	}
	if c.fileState != "" {
		path += " [" + c.fileState + "]"
	}

	insight := ""
	if len(c.insightLines) > 0 {
		insight = c.insightLines[c.insightIdx%len(c.insightLines)]
	}

	return []string{title, path, insight}
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

	// build right column: logo only (name is in the left column now)
	var rightLines []string
	if logo, ok := konfables.Logos[c.konfable.Name()]; ok {
		art := logo.Render()
		rightLines = strings.Split(art, "\n")
	}
	rightW := 0
	for _, l := range rightLines {
		if w := lipgloss.Width(l); w > rightW {
			rightW = w
		}
	}
	rightBlock := strings.Join(rightLines, "\n")

	leftW := width - rightW - 2 // 2 chars gap
	if leftW < 20 {
		// narrow fallback: centered logo
		var lines []string
		if logo, ok := konfables.Logos[c.konfable.Name()]; ok {
			art := logo.Render()
			lines = append(lines, strings.Split(centerBlock(art, width), "\n")...)
		}
		lines = append(lines, "")
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
	styles := []lipgloss.Style{c.theme.Primary, c.theme.Muted, c.theme.InsightText}
	for i, line := range leftData {
		// truncate to leftW (plain text before styling)
		if len(line) > leftW {
			line = line[:leftW-1] + "…"
		}
		s := c.theme.Text
		if i < len(styles) {
			s = styles[i]
		}
		// line 1 (path): color fileState suffix
		if i == 1 && c.fileState != "" {
			switch c.fileState {
			case "unsaved":
				s = c.theme.Warning
			case "reloaded":
				s = c.theme.Accent
			case "new":
				s = c.theme.Muted
			}
		}
		// line 2 (insight): use warning style for linter diagnostics
		if i == 2 && c.insightWarningCount > 0 {
			idx := c.insightIdx % len(c.insightLines)
			if idx < c.insightWarningCount {
				s = c.theme.Warning
			}
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

// renderTabs draws the horizontal section tab bar.
func (c content) renderTabs(_ int) string {
	if c.schema == nil || len(c.schema.Sections) <= 1 {
		return ""
	}
	var parts []string
	for i, sec := range c.schema.Sections {
		if i == c.activeSection {
			parts = append(parts, c.theme.Primary.Bold(true).Render(sec.Name))
		} else {
			parts = append(parts, c.theme.Muted.Render(sec.Name))
		}
	}
	return strings.Join(parts, c.theme.Muted.Render(" │ "))
}

func (c content) renderBody(width int) string {
	if c.schema == nil {
		if c.config != nil {
			return c.theme.Text.Render(string(c.config.Content()))
		}
		if c.konfable != nil {
			// not-installed state — show logo header, then hint
			header := c.renderHeader(width)
			msg := c.theme.Muted.Render(c.konfable.Name() + " is not installed")
			hint := c.theme.Muted.Italic(true).Render("install it to configure")
			return header + "\n" + centerLine(msg, width) + "\n" + centerLine(hint, width)
		}
		return c.theme.Muted.Render("select an app to view its configuration")
	}

	var b strings.Builder

	// header: two-column (or narrow fallback)
	b.WriteString(c.renderHeader(width))

	// section tabs
	if tabs := c.renderTabs(width); tabs != "" {
		b.WriteString(tabs)
		b.WriteByte('\n')
	}

	// search bar (when active)
	if c.searching {
		prompt := c.theme.Primary.Render("/ ")
		count := c.theme.Muted.Render(fmt.Sprintf("  %d/%d fields", len(c.visible), len(c.fields)))
		b.WriteString(prompt + c.search.View() + count)
		b.WriteByte('\n')
	}

	labelW := c.labelColumnWidth()

	for i, r := range c.visible {
		f := &c.fields[r.fieldIdx]
		isCursor := c.focused && i == c.cursor

		// type icon
		icon := fieldTypeIcon[f.Type]
		if icon == "" {
			icon = " "
		}

		// configured indicator
		_, isConfigured := c.values[f.Key]
		var dot string
		if isConfigured {
			dot = c.theme.Success.Render("●")
		} else {
			dot = c.theme.Muted.Render("○")
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

		// inline min/max bounds for number fields (skipped when inline-editing)
		showBounds := f.Type == "number" && (f.Min != nil || f.Max != nil)

		// build prefix and label
		paddedLabel := fmt.Sprintf("%-*s", labelW, f.Label)
		var prefix, label string
		if isCursor {
			prefix = c.theme.Primary.Render("▸ " + icon + " ")
			label = c.theme.Text.Bold(true).Render(paddedLabel)
		} else {
			prefix = "  " + c.theme.Muted.Render(icon) + " "
			label = c.theme.FieldLabel.Render(paddedLabel)
		}

		// inline editor replaces value on the same row
		isInlineEdit := false
		if c.editing && c.editor != nil && r.fieldIdx == c.editField {
			if ie, ok := c.editor.(InlineEditor); ok {
				isInlineEdit = true
				usedW := lipgloss.Width(prefix) + lipgloss.Width(label) + 2
				inlineW := width - usedW
				if inlineW < 10 {
					inlineW = 10
				}
				renderedVal = ie.InlineView(inlineW)
				showBounds = false
			}
		}

		if showBounds {
			lo, hi := "*", "*"
			if f.Min != nil {
				lo = formatNum(*f.Min)
			}
			if f.Max != nil {
				hi = formatNum(*f.Max)
			}
			renderedVal += c.theme.Muted.Render(fmt.Sprintf(" (%s\u2013%s)", lo, hi))
		}

		line := prefix + label + " " + dot + " " + renderedVal
		b.WriteString(line)
		b.WriteByte('\n')

		// below-row editor (enum, color — non-inline only)
		if c.editing && c.editor != nil && r.fieldIdx == c.editField && !isInlineEdit {
			b.WriteString(c.editor.View(width))
			b.WriteByte('\n')
		}
	}

	return b.String()
}

// glamourRender renders markdown using glamour, rebuilding the renderer on width/theme change.
// falls back to plain lipgloss on error.
func (c content) glamourRender(md string, width int) string {
	if c.glamCache == nil {
		return c.theme.Muted.Render(md)
	}
	if c.glamCache.renderer == nil || c.glamCache.width != width {
		r, err := glamour.NewTermRenderer(
			c.theme.GlamourStyle(),
			glamour.WithWordWrap(width),
		)
		if err != nil {
			return c.theme.Muted.Render(md)
		}
		c.glamCache.renderer = r
		c.glamCache.width = width
	}
	out, err := c.glamCache.renderer.Render(md)
	if err != nil {
		return c.theme.Muted.Render(md)
	}
	return strings.TrimRight(out, "\n")
}

func (c content) renderPreview(width, height int) string {
	if c.config == nil {
		return c.theme.Muted.Render("no preview")
	}

	f := c.currentField()

	var b strings.Builder

	if c.focused && f != nil {
		// field description (rendered as markdown via glamour)
		if f.Description != "" {
			rendered := c.glamourRender(f.Description, width)
			renderedLines := strings.Count(rendered, "\n") + 1
			b.WriteString(rendered)
			b.WriteByte('\n')
			height -= renderedLines
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
		}
		return b.String()
	}

	// cursor on section header or no field — show file snippet without highlight
	if f == nil {
		return b.String()
	}

	data := c.config.Content()
	rawLines := strings.Split(string(data), "\n")

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
