package ui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/emin/konfigurator/konfables"
	"github.com/emin/konfigurator/pkg"
	"github.com/emin/konfigurator/theme"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// field type icons (nerd font glyphs)
var fieldTypeIcon = map[string]string{
	"string": "\uf031",  //
	"number": "\uf292",  //
	"bool":   "\uf444",  //
	"enum":   "\uf150",  //
	"color":  "\uf53f",  //
	"list":   "\uf03a",  //
	"multi":  "\uf046",  //

	// widget-specific icons (checked before type)
	"font":        "\uf031",       //
	"slider":      "\U000F1A8A", // nf-md-tune_vertical
	"path":        "\uf115",       // nf-fa-folder_open
	"stylestring": "\uf893",       // nf-md-format_color_text
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

	// detail sub-model (preview/detail pane, editor state, docs URL)
	detail detail

	// original values at load time — for per-field change tracking
	origValues map[string]string

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

	// logo animation
	logoAnim    *pkg.AnimState
	logoAnimGen int

	// breadcrumb, undo/redo, diff preview
	breadcrumb breadcrumb
	undoStack  *UndoStack
	diffView   *diffView

	// search match tracking for n/N navigation
	searchMatches []int // indices into c.visible for matched rows
	searchIdx     int   // current position in searchMatches

	// "what's new" filter — toggled by root via n key
	showNewOnly bool
}

func newContent(th *theme.Theme) content {
	ti := textinput.New()
	ti.Placeholder = "search..."
	ti.CharLimit = 64
	ti.Prompt = ""

	return content{
		title:         "konfigurator",
		values:        make(map[string]string),
		collapsed:     make(map[int]bool),
		search:        ti,
		theme:         th,
		detail:        newDetail(th),
		breadcrumb:    newBreadcrumb(th),
		undoStack:     NewUndoStack(50),
		diffView:      newDiffView(th),
		searchMatches: make([]int, 0),
	}
}

func (c content) Update(msg tea.Msg) (content, tea.Cmd) {
	// when editing, forward all messages to the active editor
	if c.detail.editing && c.detail.editor != nil {
		cmd, done, canceled := c.detail.editor.Update(msg)
		if done {
			if canceled {
				c.detail.editing = false
				c.detail.editor = nil
			} else {
				settingCmd := c.commitEdit(c.detail.editor.Value())
				if settingCmd != nil {
					cmd = tea.Batch(cmd, settingCmd)
				}
			}
		}
		return c, cmd
	}

	// when searching, forward keys to search textinput
	if c.searching {
		if km, ok := msg.(tea.KeyPressMsg); ok {
			switch km.String() {
			case "esc":
				c.searching = false
				c.search.SetValue("")
				c.search.Blur()
				c.refilter()
				c.syncDetail()
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
				c.syncDetail()
				return c, nil
			case "k", "up":
				if c.cursor > 0 {
					c.cursor--
				}
				c.syncDetail()
				return c, nil
			default:
				var cmd tea.Cmd
				c.search, cmd = c.search.Update(msg)
				c.refilter()
				c.syncDetail()
				return c, cmd
			}
		}
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
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
				c.syncDetail()
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
				c.syncDetail()
			}
		case "f":
			if c.schema != nil {
				c.configuredOnly = !c.configuredOnly
				c.refilter()
				c.syncDetail()
			}
		case "/":
			if c.schema != nil {
				c.searching = true
				c.search.SetValue("")
				return c, c.search.Focus()
			}
		case "n":
			if len(c.searchMatches) > 0 {
				c.searchIdx = (c.searchIdx + 1) % len(c.searchMatches)
				c.cursor = c.searchMatches[c.searchIdx]
				c.syncDetail()
			}
		case "N":
			if len(c.searchMatches) > 0 {
				c.searchIdx--
				if c.searchIdx < 0 {
					c.searchIdx = len(c.searchMatches) - 1
				}
				c.cursor = c.searchMatches[c.searchIdx]
				c.syncDetail()
			}
		case "j", "down":
			if hasRows {
				if c.cursor < len(c.visible)-1 {
					c.cursor++
				}
				c.syncDetail()
			} else {
				c.scrollY++
			}
		case "k", "up":
			if hasRows {
				if c.cursor > 0 {
					c.cursor--
				}
				c.syncDetail()
			} else if c.scrollY > 0 {
				c.scrollY--
			}
		case "J", "shift+down":
			c.detail.scrollY++
		case "K", "shift+up":
			if c.detail.scrollY > 0 {
				c.detail.scrollY--
			}
		case "home":
			if hasRows {
				c.cursor = 0
				c.syncDetail()
			} else {
				c.scrollY = 0
			}
		case "end":
			if hasRows {
				c.cursor = len(c.visible) - 1
				c.syncDetail()
			}
		case "pgdown":
			page := c.pageSize()
			if hasRows {
				c.cursor += page
				if c.cursor >= len(c.visible) {
					c.cursor = len(c.visible) - 1
				}
				c.syncDetail()
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
				c.syncDetail()
			} else {
				c.scrollY -= page
				if c.scrollY < 0 {
					c.scrollY = 0
				}
			}
		case "backspace", "delete":
			if f := c.currentField(); f != nil && c.konfable != nil && c.config != nil {
				curVal, hasCur := c.values[f.Key]
				origVal, hasOrig := c.origValues[f.Key]
				if hasCur && curVal != origVal {
					// changed — revert to original
					if hasOrig {
						c.revertField(*f, origVal)
					} else {
						c.deleteField(*f)
					}
				} else if hasCur {
					// unchanged — delete from config
					c.deleteField(*f)
				}
			}
		case "o":
			if url := c.currentDocURL(); url != "" {
				return c, c.openDocs(url)
			}
		default:
			// type-through: printable chars seed a new editor in replace mode
			if f := c.currentField(); f != nil && msg.Text != "" {
				if f.Type != "bool" && f.Type != "enum" && f.Type != "list" && f.Type != "multi" && f.Widget == "" {
					cmd := c.openEditorWithSeed([]rune(msg.Text)[0])
					return c, cmd
				}
			}
		}

	case ThemeChangedMsg:
		c.theme = msg.Theme
		c.detail.theme = msg.Theme
		c.breadcrumb.theme = msg.Theme
		c.diffView.theme = msg.Theme

	case ExternalChangeMsg:
		if c.config != nil && c.config.Path == msg.Path {
			if err := c.config.Reload(context.Background()); err == nil {
				c.refreshValues()
				c.snapshotOrigValues()
			}
		}

	case UndoMsg:
		if op, ok := c.undoStack.Undo(); ok {
			c.applyFieldByKey(op.FieldKey, op.OldValue)
		}

	case RedoMsg:
		if op, ok := c.undoStack.Redo(); ok {
			c.applyFieldByKey(op.FieldKey, op.NewValue)
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

	case logoAnimTickMsg:
		if c.logoAnim == nil || msg.gen != c.logoAnimGen {
			return c, nil
		}
		if c.logoAnim.Tick() {
			c.logoAnim = nil
			return c, nil
		}
		return c, logoAnimCmd(c.logoAnimGen)
	}

	return c, nil
}

// openEditor creates and initializes an editor for the current cursor field.
func (c *content) openEditor() tea.Cmd {
	f := c.currentField()
	if f == nil || c.konfable == nil || c.config == nil || c.konfable.Parser() == nil {
		return nil
	}

	c.detail.editField = c.visible[c.cursor].fieldIdx
	c.detail.editOrigVal = c.values[f.Key]

	// for list fields, pass the actual multi-values (newline-joined)
	initVal := c.detail.editOrigVal
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
	c.detail.editor = e
	c.detail.editing = true
	return e.Init(*f, initVal, c.theme)
}

// openEditorWithSeed starts the editor in replace mode (empty value) and
// injects the seed rune as the first keystroke.
func (c *content) openEditorWithSeed(seed rune) tea.Cmd {
	f := c.currentField()
	if f == nil || c.konfable == nil || c.config == nil || c.konfable.Parser() == nil {
		return nil
	}

	c.detail.editField = c.visible[c.cursor].fieldIdx
	c.detail.editOrigVal = c.values[f.Key]

	e := editorForField(*f)
	c.detail.editor = e
	c.detail.editing = true

	initCmd := e.Init(*f, "", c.theme)
	seedCmd := func() tea.Msg {
		return tea.KeyPressMsg{Code: seed, Text: string(seed)}
	}
	return tea.Sequence(initCmd, seedCmd)
}

// commitEdit writes the edited value back to the config and returns
// a cmd to propagate konfigurator setting changes (theme, log_level).
func (c *content) commitEdit(value string) tea.Cmd {
	c.detail.editing = false
	c.detail.editor = nil

	if c.konfable == nil || c.config == nil || c.konfable.Parser() == nil {
		return nil
	}

	// skip write if value is unchanged
	if value == c.detail.editOrigVal {
		return nil
	}

	f := c.fields[c.detail.editField]
	oldValue := c.detail.editOrigVal
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
				return nil
			}
			c.config.SetContent(newData)
			c.undoStack.Push(EditOp{FieldKey: f.Key, OldValue: oldValue, NewValue: value})
			c.refreshValues()
			return c.settingChangedCmd(f.Key, value)
		}
	}

	serialized := formatValue(value, f.Type, c.konfable.Info().Format)
	newData, err := c.konfable.Parser().SetValue(data, f.Key, serialized)
	if err != nil {
		return nil
	}
	c.config.SetContent(newData)
	c.undoStack.Push(EditOp{FieldKey: f.Key, OldValue: oldValue, NewValue: value})
	c.refreshValues()

	return c.settingChangedCmd(f.Key, value)
}

// settingChangedCmd returns a cmd that emits a KonfSettingChangedMsg,
// or nil if not editing the konfigurator app.
func (c *content) settingChangedCmd(key, value string) tea.Cmd {
	if c.konfable == nil || c.konfable.Name() != "konfigurator" {
		return nil
	}
	return func() tea.Msg {
		return KonfSettingChangedMsg{Key: key, Value: value}
	}
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
	c.undoStack.Push(EditOp{FieldKey: f.Key, OldValue: cur, NewValue: next})
	c.refreshValues()
}

// deleteField removes a field's key from the config file.
func (c *content) deleteField(f pkg.Field) {
	p := c.konfable.Parser()
	if p == nil {
		return
	}
	oldVal := c.values[f.Key]
	data := c.config.Content()
	newData, err := p.DeleteKey(data, f.Key)
	if err != nil {
		return
	}
	c.config.SetContent(newData)
	c.undoStack.Push(EditOp{FieldKey: f.Key, OldValue: oldVal, NewValue: ""})
	c.refreshValues()
}

// revertField restores a field to its original value.
func (c *content) revertField(f pkg.Field, origVal string) {
	p := c.konfable.Parser()
	if p == nil {
		return
	}
	curVal := c.values[f.Key]
	data := c.config.Content()
	serialized := formatValue(origVal, f.Type, c.konfable.Info().Format)
	newData, err := p.SetValue(data, f.Key, serialized)
	if err != nil {
		return
	}
	c.config.SetContent(newData)
	c.undoStack.Push(EditOp{FieldKey: f.Key, OldValue: curVal, NewValue: origVal})
	c.refreshValues()
}

// applyFieldByKey writes a value to a field identified by key, used by undo/redo.
// empty value deletes the key from the config.
func (c *content) applyFieldByKey(key, value string) {
	if c.konfable == nil || c.config == nil || c.konfable.Parser() == nil {
		return
	}
	p := c.konfable.Parser()
	data := c.config.Content()

	if value == "" {
		newData, err := p.DeleteKey(data, key)
		if err != nil {
			return
		}
		c.config.SetContent(newData)
	} else {
		// find the field to get its type for formatting
		fmtStr := c.konfable.Info().Format
		fieldType := "string"
		for i := range c.fields {
			if c.fields[i].Key == key {
				fieldType = c.fields[i].Type
				break
			}
		}
		serialized := formatValue(value, fieldType, fmtStr)
		newData, err := p.SetValue(data, key, serialized)
		if err != nil {
			return
		}
		c.config.SetContent(newData)
	}
	c.refreshValues()
}

// stopWatching type-asserts the konfable for Watchable and calls Unwatch.
func (c *content) stopWatching() {
	if c.konfable == nil {
		return
	}
	if w, ok := c.konfable.(pkg.Watchable); ok {
		w.Unwatch()
	}
}

// showNotInstalled sets the active konfable for display without loading config or schema.
func (c *content) showNotInstalled(k konfables.Konfable) {
	c.stopWatching()
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
	c.showNewOnly = false
	c.searching = false
	c.search.SetValue("")
	c.search.Blur()
	c.values = make(map[string]string)
	c.origValues = make(map[string]string)
	c.scrollY = 0
	c.cursor = 0
	c.detail.editing = false
	c.detail.editor = nil
	c.detail.reset()
	c.insightLines = nil
	c.insightIdx = 0
	c.insightWarningCount = 0
	c.insightGen++
	c.logoAnimGen++
	c.logoAnim = nil
	c.undoStack.Clear()
	c.breadcrumb.SetPath(k.Name(), "", "")
	c.diffView.SetEntries(nil)
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
	c.stopWatching()

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
	c.showNewOnly = false
	c.searching = false
	c.search.SetValue("")
	c.search.Blur()
	c.values = make(map[string]string)
	c.config = nil
	c.schema = nil
	c.detail.editing = false
	c.detail.editor = nil
	c.detail.reset()
	c.undoStack.Clear()
	c.diffView.SetEntries(nil)

	info := k.Info()

	// detect whether this is a fresh file (before Load potentially creates it)
	isNewFile := k.ConfigPath() != "" && !pkg.FileExists(k.ConfigPath())

	// load config through the konfable's persister
	cf, err := pkg.NewConfigFile(context.Background(), k)
	if err != nil {
		return func() tea.Msg {
			return StatusMsg{Text: fmt.Sprintf("no config: %s", info.ConfigPath)}
		}
	}
	c.config = cf
	c.config.Path = k.ConfigPath()

	if isNewFile {
		c.fileState = "new"
	}

	// load schema (filter by detected version if known)
	schemaData, err := k.Schema()
	if err == nil && schemaData != nil {
		s, err := pkg.LoadSchema(schemaData)
		if err == nil {
			if v, ok := c.versions[k.Name()]; ok {
				s = s.FilterByVersion(v)
			}
			c.schema = s
			c.detail.docsURL = s.DocsURL
			c.buildFieldList()
		}
	}

	c.refreshValues()
	c.snapshotOrigValues()
	c.buildInsights()
	c.insightGen++

	// start watching if the konfable supports it
	if c.config != nil && c.program != nil {
		if w, ok := k.(pkg.Watchable); ok {
			p := c.program
			cfPath := c.config.Path
			_ = w.Watch(func() {
				p.Send(ExternalChangeMsg{Path: cfPath})
			})
		}
	}

	var cmds []tea.Cmd

	// init split-flap animation if we have a previous snapshot
	if snapshot != nil {
		c.splitFlap = newSplitFlap(snapshot, c.headerLeftLines(), c.insightGen)
		cmds = append(cmds, splitFlapCmd(c.insightGen))
	}
	cmds = append(cmds, c.insightTickCmd())

	// start logo animation if one is registered for this app
	c.logoAnimGen++
	if cfg, ok := konfables.LogoAnims[k.Name()]; ok {
		if logo, lok := konfables.Logos[k.Name()]; lok {
			c.logoAnim = pkg.NewAnimState(logo, cfg)
			cmds = append(cmds, logoAnimCmd(c.logoAnimGen))
		}
	} else {
		c.logoAnim = nil
	}

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
		if c.showNewOnly && f.Since == "" {
			continue
		}
		if hasSearch {
			if !fieldMatchesQuery(f, query) {
				continue
			}
		}
		c.visible = append(c.visible, row{sectionIdx: c.activeSection, fieldIdx: i})
	}

	// rebuild search match indices (for n/N navigation after search is locked)
	c.searchMatches = c.searchMatches[:0]
	if query != "" {
		for vi, r := range c.visible {
			if fieldMatchesQuery(&c.fields[r.fieldIdx], query) {
				c.searchMatches = append(c.searchMatches, vi)
			}
		}
	}
	if c.searchIdx >= len(c.searchMatches) {
		c.searchIdx = 0
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

// fieldMatchesQuery checks if a field matches the search query against key, label, and description.
func fieldMatchesQuery(f *pkg.Field, query string) bool {
	return strings.Contains(strings.ToLower(f.Key), query) ||
		strings.Contains(strings.ToLower(f.Label), query) ||
		strings.Contains(strings.ToLower(f.Description), query)
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
	return c.detail.docsURL
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

	c.detail.forceRescan()
	c.syncDetail()
	c.buildInsights()
	c.syncDiffView()
}

// pendingChange describes a single field change relative to the on-disk snapshot.
type pendingChange struct {
	Section string
	Label   string
	Key     string
	OldVal  string
	NewVal  string
	IsNew   bool // key wasn't in origValues
	Deleted bool // key was removed
}

// pendingChanges computes per-field diffs between origValues and current values.
func (c *content) pendingChanges() []pendingChange {
	if c.schema == nil || c.origValues == nil {
		return nil
	}
	var changes []pendingChange
	seen := make(map[string]bool)

	for i := range c.fields {
		f := &c.fields[i]
		seen[f.Key] = true
		origVal, hadOrig := c.origValues[f.Key]
		curVal, hasCur := c.values[f.Key]

		if hadOrig == hasCur && origVal == curVal {
			continue
		}

		sec := ""
		if c.fieldSection != nil && i < len(c.fieldSection) {
			si := c.fieldSection[i]
			if si < len(c.schema.Sections) {
				sec = c.schema.Sections[si].Name
			}
		}

		changes = append(changes, pendingChange{
			Section: sec,
			Label:   f.Label,
			Key:     f.Key,
			OldVal:  origVal,
			NewVal:  curVal,
			IsNew:   !hadOrig && hasCur,
			Deleted: hadOrig && !hasCur,
		})
	}

	// check for keys in origValues that are no longer in values (deleted outside field list)
	for key, origVal := range c.origValues {
		if seen[key] {
			continue
		}
		if _, hasCur := c.values[key]; !hasCur {
			changes = append(changes, pendingChange{
				Key:     key,
				Label:   key,
				OldVal:  origVal,
				Deleted: true,
			})
		}
	}

	return changes
}

// syncDiffView populates the diff preview from current pending changes.
func (c *content) syncDiffView() {
	c.diffView.SetEntries(c.pendingChanges())
	c.diffView.SetSize(c.width-2, c.height-2)
}

// snapshotOrigValues copies the current values as the baseline for change tracking.
func (c *content) snapshotOrigValues() {
	c.origValues = make(map[string]string, len(c.values))
	for k, v := range c.values {
		c.origValues[k] = v
	}
}

// syncDetail pushes content state into the detail sub-model and updates breadcrumb.
func (c *content) syncDetail() {
	c.detail.sync(c.currentField(), c.config, c.konfable, c.values, c.focused)

	// update breadcrumb path
	app := ""
	if c.konfable != nil {
		app = c.konfable.Name()
	}
	section := ""
	if c.schema != nil && c.activeSection < len(c.schema.Sections) {
		section = c.schema.Sections[c.activeSection].Name
	}
	field := ""
	if f := c.currentField(); f != nil {
		field = f.Key
	}
	c.breadcrumb.SetPath(app, section, field)
}

// Editing returns whether the detail panel is in edit mode.
func (c content) Editing() bool {
	return c.detail.editing
}

// splitWidths computes the field list and detail pane widths for a horizontal split.
// detail gets a fixed ~35%. returns detailW=0 when hidden.
func (c content) splitWidths(innerW int) (fieldW, detailW int) {
	if c.schema == nil || c.config == nil || len(c.fields) == 0 {
		return innerW, 0
	}
	if innerW < 50 {
		return innerW, 0
	}
	detailW = innerW * 35 / 100
	if detailW < 20 {
		detailW = 20
	}
	fieldW = innerW - detailW
	if fieldW < 30 {
		fieldW = 30
		detailW = innerW - fieldW
	}
	if detailW < 20 {
		return innerW, 0
	}
	return fieldW, detailW
}

func (c content) fieldListHeight() int {
	bodyH := c.height - logoBlockH - footerH
	// breadcrumb takes 1 line when an app is loaded
	if c.breadcrumb.app != "" {
		bodyH--
	}
	h := bodyH - c.fieldAreaOverhead()
	if h < 3 {
		h = 3
	}
	return h
}

func (c content) pageSize() int {
	p := c.fieldListHeight() - 1
	if p < 1 {
		p = 1
	}
	return p
}

// logoBlockH is the fixed height of the header/logo block (lines).
const logoBlockH = 6

// wideLayoutMinW is the content panel width threshold for switching
// to the wide layout where the detail pane spans the full height.
const wideLayoutMinW = 100

// fieldAreaOverhead returns the number of lines before the first field row
// in the field area (tabs + search bar). used by cursorLine for scroll.
func (c content) fieldAreaOverhead() int {
	h := 0
	if c.schema != nil && len(c.schema.Sections) > 1 {
		h++ // tab bar line
	}
	if c.searching || len(c.searchMatches) > 0 {
		h++ // search bar line
	}
	return h
}

// cursorLine returns the rendered line number for the current cursor position
// within the field area (relative to the scrollable body, not the full view).
func (c content) cursorLine() int {
	if c.schema == nil || len(c.visible) == 0 {
		return 0
	}
	line := c.fieldAreaOverhead()
	for i := range c.visible {
		if i == c.cursor {
			return line
		}
		line++ // field row
	}
	return 0
}

// footerH is the fixed height of the bottom preview bar.
const footerH = 1

// renderFooter builds the 1-line preview bar showing key = value for the focused field.
func (c content) renderFooter(width int) string {
	f := c.currentField()
	if f == nil {
		return c.theme.Muted.Render(strings.Repeat("─", width))
	}

	key := f.Key
	val := f.Default
	if v, ok := c.values[f.Key]; ok {
		val = v
	}

	// live editor preview override
	if c.detail.editing && c.detail.editor != nil {
		switch e := c.detail.editor.(type) {
		case *stylestringEditor:
			val = e.PreviewValue()
		case *colorEditor:
			val = e.PreviewValue()
		default:
			val = c.detail.editor.Value()
		}
	}

	// type-aware value rendering
	sep := c.theme.Muted.Render("─ ")
	keyStr := c.theme.PreviewHL.Render(key)
	eq := c.theme.Muted.Render(" = ")
	var valStr string

	switch {
	case f.Widget == "stylestring":
		sym, sty := parseStyleString(val)
		if sty != "" {
			valStr = c.theme.Primary.Render("[") +
				c.theme.Accent.Render(sym) +
				c.theme.Primary.Render("](") +
				c.theme.Muted.Render(sty) +
				c.theme.Primary.Render(")")
		} else {
			valStr = c.theme.Accent.Render(val)
		}
	case f.Type == "color":
		hex := normalizeHex(val)
		if c.detail.editing {
			if ce, ok := c.detail.editor.(*colorEditor); ok {
				hex = normalizeHex(ce.PreviewValue())
			}
		}
		if hex != "" {
			valStr = swatch(hex) + " " + c.theme.Accent.Render(strings.TrimPrefix(hex, "#"))
		} else {
			valStr = c.theme.Muted.Render("not set")
		}
	case f.Type == "bool":
		if val == "true" {
			valStr = c.theme.Success.Render("●") + " " + c.theme.Accent.Render("true")
		} else {
			valStr = c.theme.Muted.Render("○") + " " + c.theme.Accent.Render("false")
		}
	default:
		valStr = c.theme.Accent.Render(val)
	}

	line := sep + keyStr + eq + valStr

	// truncate to width
	if lipgloss.Width(line) > width {
		// rough truncation: re-render with shortened val
		maxVal := width - lipgloss.Width(sep+keyStr+eq) - 1
		if maxVal > 0 && len(val) > maxVal {
			val = val[:maxVal] + "…"
			valStr = c.theme.Accent.Render(val)
			line = sep + keyStr + eq + valStr
		}
	}

	return line
}

func (c content) View() string {
	// no border — structural division from sidebar edge and detail's left border
	innerW := c.width - 2 // 2 padding (1 each side)
	if innerW < 10 {
		innerW = 10
	}

	outerStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Width(c.width).
		MaxWidth(c.width).
		Height(c.height).
		MaxHeight(c.height).
		Align(lipgloss.Left, lipgloss.Top)

	// body area below header, minus footer
	bodyH := c.height - logoBlockH - footerH
	if bodyH < 3 {
		bodyH = 3
	}

	// handle no-schema states (no detail panel, header at full width)
	if c.schema == nil {
		headerStr := c.renderHeader(innerW)
		var bodyStr string
		switch {
		case c.config != nil:
			bodyStr = c.theme.Text.Render(string(c.config.Content()))
		case c.konfable != nil:
			msg := c.theme.Muted.Render(c.konfable.Name() + " is not installed")
			hint := c.theme.Muted.Italic(true).Render("install it to configure")
			bodyStr = centerLine(msg, innerW) + "\n" + centerLine(hint, innerW)
		default:
			bodyStr = c.theme.Muted.Render("select an app to view its configuration")
		}
		return outerStyle.Render(headerStr + bodyStr)
	}

	fieldListW, detailW := c.splitWidths(innerW)
	wide := c.width > wideLayoutMinW && detailW > 0

	// header width: left column only in wide mode, full width in narrow
	headerW := innerW
	if wide {
		headerW = fieldListW
	}
	headerStr := c.renderHeader(headerW)

	// breadcrumb line between header and field list
	c.breadcrumb.SetWidth(fieldListW)
	crumbStr := c.breadcrumb.View()
	if crumbStr != "" {
		crumbStr += "\n"
		bodyH-- // breadcrumb takes one line from body
		if bodyH < 3 {
			bodyH = 3
		}
	}

	// auto-scroll (cursor position is relative to field area)
	if len(c.visible) > 0 {
		cl := c.cursorLine()
		if cl < c.scrollY {
			c.scrollY = cl
		}
		cursorBottom := cl
		if c.detail.editing && c.detail.editor != nil {
			if _, ok := c.detail.editor.(InlineEditor); !ok {
				cursorBottom += c.detail.editor.Height() + 1
			}
		}
		if cursorBottom >= c.scrollY+bodyH {
			c.scrollY = cursorBottom - bodyH + 1
		}
	}

	// render field area (tabs + search + fields — header is separate)
	body := c.renderBody(fieldListW)

	// apply scrolling to field area
	lines := strings.Split(body, "\n")
	if c.scrollY >= len(lines) {
		c.scrollY = max(0, len(lines)-1)
	}
	if c.scrollY > 0 && c.scrollY < len(lines) {
		lines = lines[c.scrollY:]
	}
	if len(lines) > bodyH {
		lines = lines[:bodyH]
	}

	fieldView := strings.Join(lines, "\n")

	if detailW == 0 {
		footerStr := c.renderFooter(innerW)
		return outerStyle.Render(headerStr + crumbStr + fieldView + "\n" + footerStr)
	}

	// sync detail
	c.detail.sync(c.currentField(), c.config, c.konfable, c.values, c.focused)

	detailContentW := detailW - 3
	if detailContentW < 10 {
		detailContentW = 10
	}

	if wide {
		// wide layout: detail spans full height, header lives in left column
		footerStr := c.renderFooter(fieldListW)
		leftContent := headerStr + crumbStr + fieldView + "\n" + footerStr
		leftLines := strings.Count(leftContent, "\n") + 1
		for leftLines < c.height {
			leftContent += "\n"
			leftLines++
		}

		detailView := c.detail.View(detailContentW, c.height)

		detailStyled := c.theme.Detail.
			Width(detailW - 1).
			MaxWidth(detailW).
			Height(c.height).
			MaxHeight(c.height).
			Render(detailView)

		leftCol := lipgloss.NewStyle().
			Width(fieldListW).
			MaxWidth(fieldListW).
			Height(c.height).
			MaxHeight(c.height).
			Render(leftContent)

		return outerStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, leftCol, detailStyled))
	}

	// narrow layout: header spans full width, detail shares bodyH with fields
	fieldLines := strings.Count(fieldView, "\n") + 1
	for fieldLines < bodyH {
		fieldView += "\n"
		fieldLines++
	}

	detailView := c.detail.View(detailContentW, bodyH)

	detailStyled := c.theme.Detail.
		Width(detailW - 1).
		MaxWidth(detailW).
		Height(bodyH).
		MaxHeight(bodyH).
		Render(detailView)

	fieldCol := lipgloss.NewStyle().
		Width(fieldListW).
		MaxWidth(fieldListW).
		Height(bodyH).
		MaxHeight(bodyH).
		Render(fieldView)

	bodyRow := lipgloss.JoinHorizontal(lipgloss.Top, fieldCol, detailStyled)
	footerStr := c.renderFooter(fieldListW)

	return outerStyle.Render(headerStr + crumbStr + bodyRow + "\n" + footerStr)
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

	// schema compatibility warning
	if c.konfable != nil {
		if v, ok := c.versions[c.konfable.Name()]; ok {
			if reason, ok := c.schema.CompatibleWith(v); !ok {
				c.insightLines = append(c.insightLines, reason)
				c.insightWarningCount++
			}
		}
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
		c.insightWarningCount += len(diags)
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
		if path == "" && c.konfable != nil {
			path = c.konfable.Info().Name
		}
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
// always renders exactly logoBlockH lines + trailing newline.
func (c content) renderHeader(width int) string {
	hh := logoBlockH

	if c.konfable == nil {
		// no app selected — empty header padded to height
		lines := make([]string, hh)
		for i := range lines {
			lines[i] = ""
		}
		return strings.Join(lines, "\n") + "\n"
	}

	// build right column: logo (animated if running, static otherwise)
	var rightLines []string
	if c.logoAnim != nil && !c.logoAnim.Done {
		art := c.logoAnim.CurrentFrame().Render()
		rightLines = strings.Split(art, "\n")
	} else if logo, ok := konfables.Logos[c.konfable.Name()]; ok {
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
		if c.logoAnim != nil && !c.logoAnim.Done {
			art := c.logoAnim.CurrentFrame().Render()
			lines = append(lines, strings.Split(centerBlock(art, width), "\n")...)
		} else if logo, ok := konfables.Logos[c.konfable.Name()]; ok {
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
		if i == 2 && c.insightWarningCount > 0 && len(c.insightLines) > 0 {
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

// logoAnimCmd schedules the next logo animation frame at 60ms.
func logoAnimCmd(gen int) tea.Cmd {
	return tea.Tick(60*time.Millisecond, func(time.Time) tea.Msg {
		return logoAnimTickMsg{gen: gen}
	})
}

// insightTickCmd starts the next insight cycle timer.
func (c content) insightTickCmd() tea.Cmd {
	gen := c.insightGen
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg {
		return insightTickMsg{gen: gen}
	})
}

// renderTabs draws a centered section tab bar.
// the active tab is pinned to the horizontal center; neighbors scroll around it
// with distance-based dimming for a carousel feel.
func (c content) renderTabs(width int) string {
	if c.schema == nil || len(c.schema.Sections) <= 1 {
		return ""
	}

	total := len(c.schema.Sections)
	const neighbors = 2 // show up to 2 sections on each side

	// render the active tab
	activeName := c.schema.Sections[c.activeSection].Name
	activeTab := c.theme.Primary.Bold(true).Underline(true).Padding(0, 1).Render(activeName)
	activeW := lipgloss.Width(activeTab)

	// build left neighbors (nearest first, then reverse for display order)
	var leftParts []string
	for offset := 1; offset <= neighbors; offset++ {
		idx := c.activeSection - offset
		if idx < 0 {
			break
		}
		name := c.schema.Sections[idx].Name
		var styled string
		if offset == 1 {
			styled = c.theme.Subtext.Padding(0, 1).Render(name)
		} else {
			styled = c.theme.Muted.Faint(true).Padding(0, 1).Render(name)
		}
		leftParts = append([]string{styled}, leftParts...)
	}

	// build right neighbors
	var rightParts []string
	for offset := 1; offset <= neighbors; offset++ {
		idx := c.activeSection + offset
		if idx >= total {
			break
		}
		name := c.schema.Sections[idx].Name
		var styled string
		if offset == 1 {
			styled = c.theme.Subtext.Padding(0, 1).Render(name)
		} else {
			styled = c.theme.Muted.Faint(true).Padding(0, 1).Render(name)
		}
		rightParts = append(rightParts, styled)
	}

	// overflow arrows
	leftArrow := "  "
	if c.activeSection > neighbors {
		leftArrow = c.theme.Muted.Render("‹ ")
	}
	rightArrow := "  "
	if c.activeSection+neighbors < total-1 {
		rightArrow = c.theme.Muted.Render(" ›")
	}

	leftStr := strings.Join(leftParts, " ")
	rightStr := strings.Join(rightParts, " ")

	// compute left padding to pin active tab at center
	leftW := lipgloss.Width(leftArrow) + lipgloss.Width(leftStr)
	if len(leftParts) > 0 {
		leftW++ // account for space between left group and active tab
	}
	centerTarget := width/2 - activeW/2
	pad := centerTarget - leftW
	if pad < 0 {
		pad = 0
	}

	var b strings.Builder
	b.WriteString(strings.Repeat(" ", pad))
	b.WriteString(leftArrow)
	b.WriteString(leftStr)
	if len(leftParts) > 0 {
		b.WriteByte(' ')
	}
	b.WriteString(activeTab)
	if len(rightParts) > 0 {
		b.WriteByte(' ')
	}
	b.WriteString(rightStr)
	b.WriteString(rightArrow)

	return b.String()
}

// renderBody produces the scrollable field area: tabs + search + field rows.
// header and no-schema states are handled in View.
func (c content) renderBody(width int) string {
	var b strings.Builder

	// section tabs
	if tabs := c.renderTabs(width); tabs != "" {
		b.WriteString(tabs)
		b.WriteByte('\n')
	}

	// search bar (when active or has locked query)
	if c.searching || len(c.searchMatches) > 0 {
		prompt := c.theme.Primary.Render("/ ")
		var countStr string
		if len(c.searchMatches) > 0 {
			countStr = c.theme.Muted.Render(fmt.Sprintf("  %d/%d matches", c.searchIdx+1, len(c.searchMatches)))
		} else if c.searching {
			countStr = c.theme.Muted.Render(fmt.Sprintf("  %d/%d fields", len(c.visible), len(c.fields)))
		}
		if c.searching {
			b.WriteString(prompt + c.search.View() + countStr)
		} else {
			// locked search: show query text as static
			b.WriteString(prompt + c.theme.Subtext.Render(c.search.Value()) + countStr)
		}
		b.WriteByte('\n')
	}

	labelW := c.labelColumnWidth()

	// detect inline editing state once before the loop
	editingInline := c.detail.editing && c.detail.editor != nil
	gutterStyle := c.theme.Muted.Faint(true)

	for i, r := range c.visible {
		f := &c.fields[r.fieldIdx]
		isCursor := c.focused && i == c.cursor

		// is this row the one being edited?
		isEditRow := editingInline && isCursor && r.fieldIdx == c.detail.editField

		// gutter annotations: [changed][version][type]
		gutter := c.renderGutter(f, gutterStyle)

		// type icon (widget hint takes precedence)
		icon := fieldTypeIcon[f.Widget]
		if icon == "" {
			icon = fieldTypeIcon[f.Type]
		}
		if icon == "" {
			icon = " "
		}

		// configured indicator (only green when value differs from default)
		val, isConfigured := c.values[f.Key]
		if isConfigured && val == f.Default {
			isConfigured = false
		}
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

		// inline editor: replace value portion with InlineView or live preview
		if isEditRow {
			switch e := c.detail.editor.(type) {
			case InlineEditor:
				renderedVal = e.InlineView(width / 2)
			case *colorEditor:
				newHex := normalizeHex(e.PreviewValue())
				renderedVal = swatch(e.oldHex) +
					c.theme.Muted.Render(" → ") +
					swatch(newHex) +
					" " + c.theme.FieldValue.Render(newHex)
			case *stylestringEditor:
				renderedVal = c.theme.Accent.Render(e.PreviewValue())
			}
		}

		// inline min/max bounds for number fields (skipped for slider widgets and inline-editing)
		showBounds := f.Type == "number" && f.Widget != "slider" && (f.Min != nil || f.Max != nil) && !isEditRow

		// build prefix and label (gutter + cursor/icon)
		paddedLabel := fmt.Sprintf("%-*s", labelW, f.Label)
		var prefix, label string
		if isCursor {
			prefix = gutter + c.theme.Primary.Render("▎ "+icon) + " "
			label = c.theme.Text.Bold(true).Render(paddedLabel)
		} else {
			prefix = gutter + "  " + c.theme.Muted.Render(icon) + " "
			label = c.theme.FieldLabel.Render(paddedLabel)
		}

		if showBounds {
			lo, hi := "*", "*"
			if f.Min != nil {
				lo = formatNum(*f.Min)
			}
			if f.Max != nil {
				hi = formatNum(*f.Max)
			}
			boundsStr := fmt.Sprintf(" (%s\u2013%s)", lo, hi)
			usedW := lipgloss.Width(prefix) + lipgloss.Width(label) + 2 + lipgloss.Width(renderedVal)
			if usedW+len(boundsStr) <= width {
				renderedVal += c.theme.Muted.Render(boundsStr)
			}
		}

		line := prefix + label + " " + dot + " " + renderedVal

		// truncate value with ellipsis if line exceeds available width (skip for inline editors)
		if lipgloss.Width(line) > width && !isEditRow {
			// re-render with truncated value
			usedW := lipgloss.Width(prefix) + lipgloss.Width(label) + lipgloss.Width(" "+dot+" ")
			maxValW := width - usedW - 1
			if maxValW > 0 {
				valPlain := val
				if !hasVal {
					valPlain = f.Default
				}
				if len(valPlain) > maxValW {
					valPlain = valPlain[:maxValW-1] + "…"
				}
				if !hasVal {
					renderedVal = c.renderFieldValue(*f, valPlain, true)
				} else {
					renderedVal = c.renderFieldValue(*f, valPlain, false)
				}
				line = prefix + label + " " + dot + " " + renderedVal
			}
		}

		b.WriteString(line)
		b.WriteByte('\n')

		// expanded editor: render below cursor row for non-inline editors
		if isEditRow {
			if _, ok := c.detail.editor.(InlineEditor); !ok {
				editorView := c.detail.editor.View(width)
				b.WriteString(editorView)
				b.WriteByte('\n')
			}
		}
	}

	return b.String()
}

// renderFieldValue renders a field value with type-specific formatting.
func (c content) renderFieldValue(f pkg.Field, val string, isDefault bool) string {
	// stylestring rendering (widget takes priority)
	if f.Widget == "stylestring" {
		sym, sty := parseStyleString(val)
		if sty != "" {
			style := c.theme.FieldDefault
			if !isDefault {
				style = c.theme.FieldValue
			}
			return c.theme.Primary.Render("[") +
				style.Render(sym) +
				c.theme.Primary.Render("](") +
				c.theme.Accent.Render(sty) +
				c.theme.Primary.Render(")")
		}
	}

	if isDefault {
		switch f.Type {
		case "bool":
			if val == "true" {
				return c.theme.FieldDefault.Render("● true")
			}
			return c.theme.FieldDefault.Render("○ false")
		case "color":
			hex := normalizeHex(val)
			if hex == "" {
				return c.theme.FieldDefault.Render("not set")
			}
			return swatch(hex) + " " + c.theme.FieldDefault.Render(val)
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
		hex := normalizeHex(val)
		if hex == "" {
			return c.theme.Muted.Render("not set")
		}
		return swatch(hex) + " " + c.theme.FieldValue.Render(val)
	default:
		return c.theme.FieldValue.Render(val)
	}
}

// gutterTypeIcon maps field types to compact ASCII gutter symbols.
var gutterTypeIcon = map[string]string{
	"number": "#",
	"bool":   "◉",
	"enum":   "▾",
	"color":  "█",
	"list":   "⊞",
}

// renderGutter builds the left-margin annotation column for a field row.
// format: [changed][version][type] (3 chars + trailing space).
func (c content) renderGutter(f *pkg.Field, dim lipgloss.Style) string {
	var ch, ver, typ string

	// changed from default: value exists and differs from schema default
	if v, ok := c.values[f.Key]; ok && v != f.Default {
		ch = c.theme.Warning.Faint(true).Render("*")
	} else {
		ch = " "
	}

	// version annotations: deprecated takes priority over new
	switch {
	case f.Until != "":
		ver = c.theme.Error.Faint(true).Render("!")
	case f.Since != "":
		ver = c.theme.Success.Faint(true).Render("+")
	default:
		ver = " "
	}

	// type icon
	if icon, ok := gutterTypeIcon[f.Type]; ok {
		typ = dim.Render(icon)
	} else {
		typ = " "
	}

	return ch + ver + typ + " "
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
