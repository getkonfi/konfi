package ui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/eminert/konfi/konfables"
	"github.com/eminert/konfi/pkg"
	"github.com/eminert/konfi/pkg/pixelart"
	"github.com/eminert/konfi/theme"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// row represents a navigable item in the field list — either a section header or a field.
type row struct {
	isSection  bool
	sectionIdx int // index into schema.Sections
	fieldIdx   int // index into c.fields (-1 for section rows)
}

type content struct {
	title          string
	konfable       konfables.Konfable
	config         *pkg.ConfigFile
	schema         *pkg.Schema
	values         map[string]string
	cursor         int
	fields         []pkg.Field // flattened field list across all sections
	fieldSection   []int       // len == len(c.fields), maps field → section index
	visible        []row       // navigable rows (section headers + fields)
	searchIndex    *pkg.SearchIndex
	collapsed      map[int]bool // section index → collapsed
	configuredOnly bool
	changedOnly    bool
	searching      bool
	search         textinput.Model
	scrollY        int
	focused        bool
	detailFocused  bool
	width          int
	height         int
	theme          *theme.Theme
	program        *tea.Program

	// nerd font glyphs or ASCII fallback
	nerdFont bool

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
	logoAnim    *pixelart.AnimState
	logoAnimGen int

	// breadcrumb, undo/redo, diff preview
	breadcrumb breadcrumb
	undoStack  *UndoStack
	diffView   *diffView

	// cached pending changes — invalidated on value mutation
	cachedChanges      []pendingChange
	cachedChangesDirty bool

	// search match tracking for n/N navigation
	searchMatches   []int          // indices into c.visible for matched rows
	searchIdx       int            // current position in searchMatches
	searchMatchInfo map[int]string // visible row index → match explanation

	// "what's new" filter — toggled by root via n key
	showNewOnly bool

	// effective config view — shows all fields with defaults filled in
	showEffective bool

	// bookmarks
	bookmarks      map[string]bool
	bookmarkedOnly bool

	// last error from a void edit method — checked and cleared by Update callers
	lastErr string

	// cached label column width — set in buildFieldList
	labelW int
	// pre-padded labels for rendering — set in buildFieldList
	paddedLabels []string

	// dashboard data (shown when no app is selected)
	dashboardApps []dashboardApp
	appVersion    string

	// pre-parsed schema cache (populated at startup by computeNewCounts)
	schemaCache map[string]*pkg.Schema

	// cached layout styles — recomputed on resize
	outerStyle lipgloss.Style
	layoutW    int // width that produced outerStyle
	layoutH    int // height that produced outerStyle

	// cached content string for schema-less raw view (keyed by config generation)
	rawContentStr string
	rawContentGen uint64
}

// dashboardApp holds summary info for the landing page.
type dashboardApp struct {
	icon            string
	name            string
	installed       bool
	version         string
	configuredCount int    // fields with non-default values
	totalFields     int    // total schema fields
	deprecatedCount int    // deprecated diagnostics
	newCount        int    // fields added in the detected version
	coverage        string // from schema.Coverage
	minAppVersion   string // schema min supported version
	maxAppVersion   string // schema max supported version
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
			case "down":
				if c.cursor < len(c.visible)-1 {
					c.cursor++
				}
				c.syncDetail()
				return c, nil
			case "up":
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

		if c.detailFocused {
			switch msg.String() {
			case "left", "esc":
				c.detailFocused = false
				c.syncDetail()
			case "j", "down":
				c.detail.scroll(1)
			case "k", "up":
				c.detail.scroll(-1)
			case "pgdown":
				c.detail.scroll(c.detailPageSize())
			case "pgup":
				c.detail.scroll(-c.detailPageSize())
			case "home":
				c.detail.scrollTop()
			case "end":
				c.detail.scrollBottom()
			}
			return c, nil
		}

		hasRows := c.schema != nil && len(c.visible) > 0

		switch msg.String() {
		case "right":
			if c.currentField() != nil && c.detailPaneVisible() {
				c.detailFocused = true
				c.syncDetail()
				c.detail.centerPreview(c.detailPageSize())
			}
		case "space", "enter":
			if hasRows && c.cursor < len(c.visible) && c.visible[c.cursor].isSection {
				si := c.visible[c.cursor].sectionIdx
				c.collapsed[si] = !c.collapsed[si]
				c.refilter()
				c.syncDetail()
				return c, nil
			}
			if f := c.currentField(); f != nil {
				if f.Type == "bool" {
					settingCmd := c.toggleBool(*f)
					errCmd := c.drainErr()
					return c, tea.Batch(settingCmd, errCmd)
				}
				cmd := c.openEditor()
				return c, cmd
			}
		case "f":
			if c.schema != nil {
				c.configuredOnly = !c.configuredOnly
				c.refilter()
				c.syncDetail()
			}
		case "g":
			if c.schema != nil {
				c.showEffective = !c.showEffective
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
			if c.detail.scrollY < 500 {
				c.detail.scrollY++
			}
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
				if _, hasCur := c.values[f.Key]; hasCur {
					c.deleteField(*f)
				}
				cmd := c.drainErr()
				return c, cmd
			}
		case "d":
			// delete key from config entirely
			if f := c.currentField(); f != nil && c.konfable != nil && c.config != nil {
				if _, hasCur := c.values[f.Key]; hasCur {
					c.deleteField(*f)
				}
				cmd := c.drainErr()
				return c, cmd
			}
		case "[":
			if hasRows {
				for vi := c.cursor - 1; vi >= 0; vi-- {
					if c.visible[vi].isSection {
						c.cursor = vi
						c.syncDetail()
						break
					}
				}
			}
		case "]":
			if hasRows {
				for vi := c.cursor + 1; vi < len(c.visible); vi++ {
					if c.visible[vi].isSection {
						c.cursor = vi
						c.syncDetail()
						break
					}
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
					r, _ := utf8.DecodeRuneInString(msg.Text)
					cmd := c.openEditorWithSeed(r)
					return c, cmd
				}
			}
		}

	case ThemeChangedMsg:
		c.theme = msg.Theme
		c.detail.theme = msg.Theme
		c.detail.cachedMD = nil
		c.breadcrumb.theme = msg.Theme
		c.diffView.theme = msg.Theme

	case ExternalChangeMsg:
		if c.config != nil && c.config.Path == msg.Path {
			if c.config.Dirty() {
				c.fileState = "external change (unsaved edits kept)"
				return c, nil
			}
			cfg := c.config
			return c, func() tea.Msg {
				_ = cfg.Reload(context.Background())
				return reloadResultMsg{source: "external"}
			}
		}

	case UndoMsg:
		if op, ok := c.undoStack.Undo(); ok {
			c.applyFieldByKey(op.FieldKey, op.OldValue)
			cmd := c.drainErr()
			return c, cmd
		}

	case RedoMsg:
		if op, ok := c.undoStack.Redo(); ok {
			c.applyFieldByKey(op.FieldKey, op.NewValue)
			cmd := c.drainErr()
			return c, cmd
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

	e := editorForField(*f)

	// for raw JSON widgets, use FindValue (not MultiValueParser)
	initVal := c.detail.editOrigVal
	if f.Widget == "hook" || f.Widget == "togglemap" || f.Widget == "structlist" {
		if raw, ok := c.konfable.Parser().FindValue(c.config.Content(), f.Key); ok {
			initVal = raw
		} else if f.Widget == "structlist" {
			initVal = ""
		} else {
			initVal = "[]"
		}
	} else if f.Type == "list" {
		// for list fields, resolve the init value based on which editor was chosen
		if mvp, ok := c.konfable.Parser().(konfables.MultiValueParser); ok {
			if vals, found := mvp.FindValues(c.config.Content(), f.Key); found {
				if _, isListEd := e.(*listEditor); isListEd {
					// list editor: pass all values newline-joined
					initVal = strings.Join(vals, "\n")
				} else if len(vals) > 0 {
					// widget override (e.g. font picker): pass first value only
					initVal = vals[0]
				} else {
					initVal = ""
				}
			} else {
				initVal = ""
			}
		}
	}

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

	// raw JSON widgets: write via SetValue (not MultiValueParser)
	if f.Widget == "hook" || f.Widget == "togglemap" || f.Widget == "structlist" {
		newData, err := c.konfable.Parser().SetValue(data, f.Key, value)
		if err != nil {
			return func() tea.Msg { return StatusMsg{Text: "edit failed: " + err.Error()} }
		}
		c.config.SetContent(newData)
		c.undoStack.Push(EditOp{FieldKey: f.Key, OldValue: oldValue, NewValue: value})
		c.refreshValues()
		return c.settingChangedCmd(f.Key, value)
	}

	// list fields use MultiValueParser
	if f.Type == "list" {
		if mvp, ok := c.konfable.Parser().(konfables.MultiValueParser); ok {
			vals := splitListValue(value)
			newData, err := mvp.SetValues(data, f.Key, vals)
			if err != nil {
				return func() tea.Msg { return StatusMsg{Text: "edit failed: " + err.Error()} }
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
		return func() tea.Msg { return StatusMsg{Text: "edit failed: " + err.Error()} }
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

// toggleBool flips a boolean field value immediately and returns a cmd
// that propagates konfigurator setting changes (e.g. nerd_font, browse_loads_app).
func (c *content) toggleBool(f pkg.Field) tea.Cmd {
	if c.konfable == nil || c.config == nil || c.konfable.Parser() == nil {
		return nil
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
		c.lastErr = "toggle failed: " + err.Error()
		return nil
	}
	c.config.SetContent(newData)
	c.undoStack.Push(EditOp{FieldKey: f.Key, OldValue: cur, NewValue: next})
	c.refreshValues()
	return c.settingChangedCmd(f.Key, next)
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
		c.lastErr = "delete failed: " + err.Error()
		return
	}
	c.config.SetContent(newData)
	c.undoStack.Push(EditOp{FieldKey: f.Key, OldValue: oldVal, NewValue: ""})
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
			c.lastErr = "undo/redo failed: " + err.Error()
			return
		}
		c.config.SetContent(newData)
	} else {
		// find the field to get its type for formatting
		fmtStr := c.konfable.Info().Format
		fieldType := "string"
		fieldWidget := ""
		for i := range c.fields {
			if c.fields[i].Key == key {
				fieldType = c.fields[i].Type
				fieldWidget = c.fields[i].Widget
				break
			}
		}

		// raw JSON widgets use plain SetValue
		if fieldWidget == "hook" || fieldWidget == "togglemap" || fieldWidget == "structlist" {
			newData, err := p.SetValue(data, key, value)
			if err != nil {
				c.lastErr = "undo/redo failed: " + err.Error()
				return
			}
			c.config.SetContent(newData)
			c.refreshValues()
			return
		}

		// list fields need SetValues for repeated-key formats
		if fieldType == "list" {
			if mvp, ok := p.(konfables.MultiValueParser); ok {
				vals := splitListValue(value)
				newData, err := mvp.SetValues(data, key, vals)
				if err != nil {
					c.lastErr = "undo/redo failed: " + err.Error()
					return
				}
				c.config.SetContent(newData)
				c.refreshValues()
				return
			}
		}

		serialized := formatValue(value, fieldType, fmtStr)
		newData, err := p.SetValue(data, key, serialized)
		if err != nil {
			c.lastErr = "undo/redo failed: " + err.Error()
			return
		}
		c.config.SetContent(newData)
	}
	c.refreshValues()
}

// drainErr returns a Cmd that surfaces lastErr as a StatusMsg, then clears it.
func (c *content) drainErr() tea.Cmd {
	if c.lastErr == "" {
		return nil
	}
	text := c.lastErr
	c.lastErr = ""
	return func() tea.Msg { return StatusMsg{Text: text} }
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

// showDashboard resets content to the landing/welcome state.
func (c *content) showDashboard() {
	c.stopWatching()
	c.konfable = nil
	c.title = "konfigurator"
	c.config = nil
	c.schema = nil
	c.fields = nil
	c.fieldSection = nil
	c.visible = nil
	c.searchIndex = nil
	c.collapsed = make(map[int]bool)
	c.configuredOnly = false
	c.changedOnly = false
	c.showNewOnly = false
	c.showEffective = false
	c.searching = false
	c.search.SetValue("")
	c.search.Blur()
	c.values = make(map[string]string)
	c.origValues = make(map[string]string)
	c.scrollY = 0
	c.detailFocused = false
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
	c.breadcrumb.SetPath("", "", "")
	c.diffView.SetEntries(nil)
}

// loadApp sets the active konfable, loads its schema (fast, embedded), and
// dispatches an async config load. the appLoadedMsg handler applies the config.
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
	c.detailFocused = false
	c.cursor = 0
	c.fields = nil
	c.fieldSection = nil
	c.visible = nil
	c.searchIndex = nil
	c.collapsed = make(map[int]bool)
	c.configuredOnly = false
	c.changedOnly = false
	c.showNewOnly = false
	c.showEffective = false
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

	// load schema — use cache from startup if available, else parse
	if cached, ok := c.schemaCache[k.Name()]; ok {
		s := cached
		if v, ok := c.versions[k.Name()]; ok {
			s = s.FilterByVersion(v)
		}
		c.schema = s
		c.detail.docsURL = s.DocsURL
		c.buildFieldList()
	} else {
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
	}

	c.buildInsights()
	c.insightGen++

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
			c.logoAnim = pixelart.NewAnimState(logo, cfg)
			cmds = append(cmds, logoAnimCmd(c.logoAnimGen))
		}
	} else {
		c.logoAnim = nil
	}

	// dispatch async config load (slow for gsettings/dconf backends)
	isNewFile := k.ConfigPath() != "" && !pkg.FileExists(k.ConfigPath())
	path := k.ConfigPath()
	appName := k.Name()
	cmds = append(cmds, func() tea.Msg {
		cf, loadErr := pkg.NewConfigFile(context.Background(), k)
		return appLoadedMsg{
			config:  cf,
			path:    path,
			isNew:   isNewFile,
			err:     loadErr,
			appName: appName,
		}
	})

	return tea.Batch(cmds...)
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

func (c *content) fieldListFocused() bool {
	return c.focused && !c.detailFocused
}

func (c *content) detailPaneVisible() bool {
	if c.schema == nil || c.config == nil || len(c.fields) == 0 {
		return false
	}
	innerW := c.width - 2
	if innerW < 10 {
		innerW = 10
	}
	_, detailW := c.splitWidths(innerW)
	return detailW > 0
}

func (c *content) detailPaneHeight() int {
	if !c.detailPaneVisible() {
		return 0
	}
	innerW := c.width - 2
	if innerW < 10 {
		innerW = 10
	}
	_, detailW := c.splitWidths(innerW)
	if c.width > wideLayoutMinW && detailW > 0 {
		return c.height
	}
	bodyH := c.height - logoBlockH - footerH
	if c.breadcrumb.app != "" {
		bodyH--
	}
	if bodyH < 3 {
		bodyH = 3
	}
	return bodyH
}

func (c *content) detailPageSize() int {
	page := c.detailPaneHeight() - 2
	if page < 1 {
		page = 1
	}
	return page
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
	c.cachedChangesDirty = true
	if c.config == nil || c.schema == nil || c.konfable == nil {
		return
	}

	p := c.konfable.Parser()
	if p == nil {
		return
	}

	data := c.config.Content()

	// batch lookup: single pass over the file when supported
	switch p := p.(type) {
	case konfables.BatchMultiParser:
		// flat parsers with repeated keys (ghostty keybind, palette)
		singles, multi := p.FindAllMulti(data)
		for _, sec := range c.schema.Sections {
			for i := range sec.Fields {
				f := &sec.Fields[i]
				if vals, isList := multi[f.Key]; isList {
					c.values[f.Key] = strings.Join(vals, ", ")
				} else if v, found := singles[f.Key]; found {
					c.values[f.Key] = v
				}
			}
		}
	case konfables.BatchParser:
		all := p.FindAll(data)
		for _, sec := range c.schema.Sections {
			for i := range sec.Fields {
				if v, found := all[sec.Fields[i].Key]; found {
					c.values[sec.Fields[i].Key] = v
				}
			}
		}
	default:
		// fallback: per-field lookup
		mvp, hasMVP := p.(konfables.MultiValueParser)
		for _, sec := range c.schema.Sections {
			for i := range sec.Fields {
				f := &sec.Fields[i]
				if f.Widget == "hook" || f.Widget == "togglemap" || f.Widget == "structlist" {
					if v, ok := p.FindValue(data, f.Key); ok {
						c.values[f.Key] = v
					}
				} else if f.Type == "list" && hasMVP {
					if vals, ok := mvp.FindValues(data, f.Key); ok {
						switch len(vals) {
						case 0:
						case 1:
							c.values[f.Key] = vals[0]
						default:
							c.values[f.Key] = strings.Join(vals, ", ")
						}
					}
				} else if v, ok := p.FindValue(data, f.Key); ok {
					c.values[f.Key] = v
				}
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

// pendingChanges returns cached per-field diffs, recomputing only when dirty.
func (c *content) pendingChanges() []pendingChange {
	if !c.cachedChangesDirty {
		return c.cachedChanges
	}
	c.cachedChangesDirty = false
	c.cachedChanges = c.computePendingChanges()
	return c.cachedChanges
}

func (c *content) computePendingChanges() []pendingChange {
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
	c.cachedChangesDirty = true
	c.origValues = make(map[string]string, len(c.values))
	for k, v := range c.values {
		c.origValues[k] = v
	}
}

// syncDetail pushes content state into the detail sub-model and updates breadcrumb.
func (c *content) syncDetail() {
	c.detail.sync(c.currentField(), c.config, c.konfable, c.values, c.focused && c.detailFocused)

	// update breadcrumb path
	app := ""
	if c.konfable != nil {
		app = c.konfable.Name()
	}
	field := ""
	if f := c.currentField(); f != nil {
		field = f.Key
	}
	section := ""
	if c.cursor >= 0 && c.cursor < len(c.visible) {
		r := c.visible[c.cursor]
		if !r.isSection && c.schema != nil && r.sectionIdx < len(c.schema.Sections) {
			section = c.schema.Sections[r.sectionIdx].Name
		}
	}
	c.breadcrumb.SetPath(app, section, field)
}

// Editing returns whether the detail panel is in edit mode.
func (c content) Editing() bool {
	return c.detail.editing
}

// buildInsights computes the cycling insight lines from current state.
// linter warnings come first, then stats.
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
			// skip "unknown key" warnings — schemas don't cover every valid key
			if d.Kind == "unknown" {
				continue
			}
			c.insightLines = append(c.insightLines, d.Message)
			c.insightWarningCount++
		}
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
}
