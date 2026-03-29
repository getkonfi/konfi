package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/emin/konfigurator/konfables"
	"github.com/emin/konfigurator/pkg"
	"github.com/emin/konfigurator/setup"
	"github.com/emin/konfigurator/theme"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"golang.org/x/mod/semver"
)

type pane int

const (
	paneSidebar pane = iota
	paneContent
)

// AppMode governs how key events are routed.
type AppMode int

const (
	ModeNormal AppMode = iota
	ModeEdit
	ModeSearch
)

// navEntry records a navigation position for back/forward.
type navEntry struct {
	appIndex int
}

type root struct {
	app      *setup.App
	sidebar  sidebar
	content  content
	status   statusbar
	palette  palette
	focus    pane
	mode     AppMode
	width    int
	height   int
	ready    bool
	showHelp     bool
	confirmQuit  bool
	confirmSwitch bool          // dirty-state guard for app switching
	pendingSwitch *AppSelectedMsg // deferred app switch awaiting confirmation

	// all konfables (indexed by sidebar item order)
	allKonfables []konfables.Konfable
	installed    map[string]bool

	// navigation history for back/forward
	navHistory    []navEntry
	navHistoryPos int

	// clipboard paste pending — set when ReadClipboard is issued
	clipboardPending bool

	// per-app count of "new" fields (since == detected version)
	newCounts map[string]int

	// cached parsed schemas — populated by computeNewCounts, reused by loadApp
	schemaCache map[string]*pkg.Schema

	// AI ask overlay
	ask         askOverlay
	pendingJump *pendingJump // deferred field jump after cross-app AI navigation

	// diff confirmation overlay before saving
	showDiffPreview bool

	// program ref for sending messages from outside the event loop
	program *tea.Program
}

// pendingJump stores a deferred field navigation for after an app load completes.
type pendingJump struct {
	fieldKey string
}

// ProgramSetter allows main to inject the tea.Program after creation.
type ProgramSetter interface {
	SetProgram(p *tea.Program)
}

func (r *root) SetProgram(p *tea.Program) {
	r.program = p
	r.content.program = p
}

// NewRoot creates the top-level Bubble Tea model.
func NewRoot(app *setup.App) tea.Model {
	th := app.Theme

	// build installed set from detected apps
	installed := make(map[string]bool)
	for _, d := range app.Detected {
		installed[d.Name()] = true
	}

	nerdFont := app.Config.NerdFont

	// home item at top of sidebar
	items := []sidebarItem{{
		icon:      "\uf015", // nf-fa-home
		plainIcon: "~",
		name:      "home",
		installed: true,
		home:      true,
	}}
	var allK []konfables.Konfable
	for _, ki := range setup.AllKonfablesWithInfo() {
		if k, ok := ki.Konfable.(konfables.Konfable); ok {
			info := k.Info()
			items = append(items, sidebarItem{
				icon:      info.NerdIcon,
				plainIcon: info.Icon,
				name:      k.Name(),
				installed: installed[k.Name()],
				system:    ki.System,
			})
			allK = append(allK, k)
		} else {
			app.Logger.Warn().Str("app", ki.Konfable.Name()).Msg("registered app does not satisfy full Konfable interface")
		}
	}

	// compute per-app "what's new" field counts + cache parsed schemas
	newCounts, schemaCache := computeNewCounts(allK, app.Versions)

	sb := newSidebar(items, th)
	sb.nerdFont = nerdFont
	sb.newCounts = newCounts
	ct := newContent(th)
	ct.nerdFont = nerdFont
	ct.detail.nerdFont = nerdFont
	ct.versions = app.Versions
	ct.appVersion = app.AppVersion
	ct.schemaCache = schemaCache

	// build cross-app equivalent field index
	var installedNames []string
	for name := range installed {
		installedNames = append(installedNames, name)
	}
	ct.crossRef = pkg.NewCrossRefIndex(schemaCache, installedNames)

	// parse bookmarks from config
	ct.bookmarks = make(map[string]bool)
	for _, b := range app.Config.Bookmarks {
		ct.bookmarks[b] = true
	}

	// populate dashboard data with stats
	for _, k := range allK {
		info := k.Info()
		nIcon := info.NerdIcon
		if !nerdFont {
			nIcon = info.Icon
		}
		if nIcon == "" {
			nIcon = info.Icon
		}
		da := dashboardApp{
			icon:      nIcon,
			name:      k.Name(),
			installed: installed[k.Name()],
		}
		if v, ok := app.Versions[k.Name()]; ok {
			da.version = v
		}

		// stats from schema
		if s, ok := schemaCache[k.Name()]; ok {
			for si := range s.Sections {
				da.totalFields += len(s.Sections[si].Fields)
			}
			da.coverage = s.Coverage

			// count configured + deprecated for installed apps
			if installed[k.Name()] {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				cf, err := pkg.NewConfigFile(ctx, k)
				cancel()
				if err == nil && cf != nil {
					data := cf.Content()
					p := k.Parser()
					if p != nil {
						configured := 0
						if bp, ok := p.(konfables.BatchParser); ok {
							all := bp.FindAll(data)
							configured = len(all)
						} else {
							// per-field lookup against schema keys
							for si := range s.Sections {
								for fi := range s.Sections[si].Fields {
									if _, found := p.FindValue(data, s.Sections[si].Fields[fi].Key); found {
										configured++
									}
								}
							}
						}
						da.configuredCount = configured

						// deprecated count via diagnostics
						var configKeys []string
						if bp, ok := p.(konfables.BatchParser); ok {
							for key := range bp.FindAll(data) {
								configKeys = append(configKeys, key)
							}
						}
						if len(configKeys) > 0 {
							diags := pkg.Diagnose(configKeys, s, da.version)
							for _, d := range diags {
								if d.Kind == "deprecated" {
									da.deprecatedCount++
								}
							}
						}
					}
				}
			}
		}

		ct.dashboardApps = append(ct.dashboardApps, da)
	}
	st := newStatusbar(th)
	pal := newPalette(th)
	ask := newAskOverlay(th, schemaCache)

	r := &root{
		app:          app,
		sidebar:      sb,
		content:      ct,
		status:       st,
		palette:      pal,
		ask:          ask,
		focus:        paneSidebar,
		mode:         ModeNormal,
		allKonfables: allK,
		installed:    installed,
		newCounts:    newCounts,
		schemaCache:  schemaCache,
	}
	r.sidebar.focused = true
	r.updateHints()
	return r
}

func (r *root) Init() tea.Cmd {
	return nil
}

func (r *root) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// palette intercepts ALL input when visible
	if r.palette.Visible() {
		switch msg.(type) {
		case tea.KeyPressMsg:
			p, cmd := r.palette.Update(msg)
			r.palette = p
			return r, cmd
		case PaletteSelectedMsg:
			sel := msg.(PaletteSelectedMsg)
			r.palette.Close()
			// re-dispatch the inner action
			return r.Update(sel.Action)
		case PaletteClosedMsg:
			r.palette.Close()
			return r, nil
		default:
			p, cmd := r.palette.Update(msg)
			r.palette = p
			return r, cmd
		}
	}

	// ask overlay intercepts ALL input when visible
	if r.ask.Visible() {
		var cmd tea.Cmd
		switch jump := msg.(type) {
		case AskJumpMsg:
			r.ask.Close()
			// same-app: jump directly
			if r.content.konfable != nil && r.content.konfable.Name() == jump.App {
				r.content.jumpToFieldByKey(jump.Key)
				r.focusPane(paneContent)
				r.updateHints()
				return r, nil
			}
			// cross-app: select app, store pending jump
			for i, k := range r.allKonfables {
				if k.Name() != jump.App {
					continue
				}
				r.pendingJump = &pendingJump{fieldKey: jump.Key}
				r.pushNav()
				r.sidebar.setCursorToApp(i)
				cmd = r.content.loadApp(r.allKonfables[i])
				r.focusPane(paneContent)
				return r, cmd
			}
			return r, nil
		default:
			r.ask, cmd = r.ask.Update(msg)
			return r, cmd
		}
	}

	// diff confirmation overlay intercepts keys when visible
	if r.showDiffPreview {
		if km, ok := msg.(tea.KeyPressMsg); ok {
			switch km.String() {
			case "enter", "y":
				r.showDiffPreview = false
				r.status.status = "saving..."
				cfg := r.content.config
				return r, func() tea.Msg {
					err := cfg.Save(context.Background())
					return saveResultMsg{err: err}
				}
			case "esc", "n":
				r.showDiffPreview = false
				r.status.status = ""
				return r, nil
			}
			return r, nil
		}
	}

	// when content is in edit mode, only ctrl+c stays at root level —
	// everything else passes through so esc/blink/keys reach the editor.
	if r.content.Editing() {
		if km, ok := msg.(tea.KeyPressMsg); ok && km.String() == "ctrl+c" {
			if r.content.config != nil {
				r.content.stopWatching()
			}
			return r, tea.Quit
		}
		var cmd tea.Cmd
		r.content, cmd = r.content.Update(msg)
		cmds = append(cmds, cmd)
		// sync file state after edit commits
		if r.content.config != nil && r.content.config.Dirty() {
			r.content.fileState = "unsaved"
		}
		r.updateHints()
		return r, tea.Batch(cmds...)
	}

	// when content is searching, forward keys to content (same pattern as sidebar search)
	if r.content.searching {
		if km, ok := msg.(tea.KeyPressMsg); ok {
			if km.String() == "ctrl+c" {
				if r.content.config != nil {
					r.content.stopWatching()
				}
				return r, tea.Quit
			}
			var cmd tea.Cmd
			r.content, cmd = r.content.Update(msg)
			r.updateHints()
			return r, cmd
		}
	}

	// when help overlay is shown, swallow all keys except ?/esc to dismiss
	if r.showHelp {
		if km, ok := msg.(tea.KeyPressMsg); ok {
			switch km.String() {
			case "?", "esc":
				r.showHelp = false
				r.updateHints()
			}
			return r, nil
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		r.width = msg.Width
		r.height = msg.Height
		r.ready = true
		r.layout()
		return r, nil

	case tea.KeyPressMsg:
		// when sidebar is searching, don't intercept keys that should reach the textinput
		if r.sidebar.searching {
			switch msg.String() {
			case "ctrl+c":
				if r.content.config != nil {
					r.content.stopWatching()
				}
				return r, tea.Quit
			case "esc":
				// let sidebar handle esc to clear search
				var cmd tea.Cmd
				r.sidebar, cmd = r.sidebar.Update(msg)
				r.updateHints()
				return r, cmd
			default:
				// forward everything else to sidebar
				var cmd tea.Cmd
				r.sidebar, cmd = r.sidebar.Update(msg)
				r.updateHints()
				return r, cmd
			}
		}

		// reset confirm-quit on any key that isn't q
		if msg.String() != "q" {
			r.confirmQuit = false
		}
		// reset confirm-switch on keys that aren't navigation
		switch msg.String() {
		case "1", "2", "3", "4", "5", "6", "7", "8", "9",
			"ctrl+n", "ctrl+p", "ctrl+o", "ctrl+]",
			"enter", "space", "j", "k", "up", "down":
			// keep confirmSwitch alive for navigation keys
		default:
			r.confirmSwitch = false
			r.pendingSwitch = nil
		}

		switch msg.String() {
		case "ctrl+c":
			if r.content.config != nil {
				r.content.stopWatching()
			}
			return r, tea.Quit

		case "q":
			if r.content.config != nil && r.content.config.Dirty() && !r.confirmQuit {
				r.confirmQuit = true
				r.status.status = "unsaved changes — q to quit, ctrl+s to save"
				return r, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
					return confirmQuitClearMsg{}
				})
			}
			if r.content.config != nil {
				r.content.stopWatching()
			}
			return r, tea.Quit

		case "ctrl+s":
			if r.content.config != nil && r.content.config.Dirty() {
				r.content.syncDiffView()
				r.showDiffPreview = true
				return r, nil
			}
			r.status.status = "no changes"
			return r, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return fileStateClearMsg{} })

		case "tab":
			r.cyclePane(1)
			return r, nil

		case "shift+tab":
			r.cyclePane(-1)
			return r, nil

		case "left":
			if r.focus != paneSidebar {
				r.focusPane(paneSidebar)
			}
			return r, nil

		case "right":
			if r.focus != paneContent {
				r.focusPane(paneContent)
			}
			return r, nil

		case "esc":
			// layered esc: clear filters before jumping to sidebar
			if r.content.bookmarkedOnly {
				r.content.bookmarkedOnly = false
				r.content.refilter()
				r.content.syncDetail()
				r.status.status = ""
				r.updateHints()
				return r, nil
			}
			if r.content.showEffective {
				r.content.showEffective = false
				r.content.refilter()
				r.content.syncDetail()
				r.updateHints()
				return r, nil
			}
			if r.content.showNewOnly {
				r.content.showNewOnly = false
				r.content.refilter()
				r.content.syncDetail()
				r.status.status = ""
				r.updateHints()
				return r, nil
			}
			if r.content.configuredOnly {
				r.content.configuredOnly = false
				r.content.refilter()
				r.content.syncDetail()
				r.updateHints()
				return r, nil
			}
			if len(r.content.searchMatches) > 0 || r.content.search.Value() != "" {
				r.content.search.SetValue("")
				r.content.searchMatches = r.content.searchMatches[:0]
				r.content.searchIdx = 0
				r.content.refilter()
				r.content.syncDetail()
				r.updateHints()
				return r, nil
			}
			if r.focus != paneSidebar {
				r.focusPane(paneSidebar)
			}
			return r, nil

		case "t":
			r.cycleTheme()
			themeMsg := ThemeChangedMsg{Theme: r.app.Theme}
			var cmd tea.Cmd
			r.sidebar, cmd = r.sidebar.Update(themeMsg)
			cmds = append(cmds, cmd)
			r.content, cmd = r.content.Update(themeMsg)
			cmds = append(cmds, cmd)
			r.status.theme = r.app.Theme
			r.status.themeName = r.app.Theme.Palette.Name
			r.status.refreshStyles()
			return r, tea.Batch(cmds...)

		case "?":
			r.showHelp = true
			r.updateHints()
			return r, nil

		case "ctrl+k":
			cmdItems := r.buildPaletteItems()
			fldItems := r.buildFieldItems()
			cmd := r.palette.Open(PaletteModeCommands, cmdItems, fldItems)
			r.palette.width = r.width
			r.palette.height = r.height
			return r, cmd

		case "ctrl+a":
			if r.installed["claude"] {
				cmd := r.ask.Open()
				r.ask.width = r.width
				r.ask.height = r.height
				return r, cmd
			}

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			idx := int(msg.String()[0]-'0') - 1
			if idx < len(r.allKonfables) {
				r.pushNav()
				r.sidebar.setCursorToApp(idx)
				return r, func() tea.Msg {
					return AppSelectedMsg{Index: idx, Confirmed: true}
				}
			}
			return r, nil

		case "ctrl+n":
			cur := r.sidebar.appIndex()
			if cur >= 0 && cur < len(r.allKonfables)-1 {
				r.pushNav()
				next := cur + 1
				r.sidebar.setCursorToApp(next)
				return r, func() tea.Msg {
					return AppSelectedMsg{Index: next, Confirmed: true}
				}
			}
			return r, nil

		case "ctrl+p":
			cur := r.sidebar.appIndex()
			if cur > 0 {
				r.pushNav()
				prev := cur - 1
				r.sidebar.setCursorToApp(prev)
				return r, func() tea.Msg {
					return AppSelectedMsg{Index: prev, Confirmed: true}
				}
			}
			return r, nil

		case "ctrl+o":
			if cmd := r.navBack(); cmd != nil {
				return r, cmd
			}
			return r, nil

		case "ctrl+]":
			if cmd := r.navForward(); cmd != nil {
				return r, cmd
			}
			return r, nil

		case "y":
			if r.focus == paneContent {
				cmd := r.yankField()
				if cmd != nil {
					r.updateHints()
					return r, cmd
				}
			}

		case "p":
			if r.focus == paneContent {
				cmd := r.pasteField()
				if cmd != nil {
					return r, cmd
				}
			}

		case "w":
			// only toggle "what's new" filter when content focused with no active search matches
			if r.focus == paneContent && len(r.content.searchMatches) == 0 {
				r.toggleNewFilter()
				r.updateHints()
				return r, nil
			}

		case "e":
			if r.focus == paneContent && r.content.config != nil && r.content.config.Path != "" {
				return r, r.openInEditor()
			}

		case "m":
			if r.focus == paneContent && r.content.konfable != nil {
				cmd := r.toggleBookmark()
				if cmd != nil {
					r.updateHints()
					return r, cmd
				}
			}

		case "b":
			if r.focus == paneContent && r.content.schema != nil {
				r.content.bookmarkedOnly = !r.content.bookmarkedOnly
				r.content.refilter()
				r.content.syncDetail()
				if r.content.bookmarkedOnly {
					r.status.status = "showing bookmarks"
				} else {
					r.status.status = ""
				}
				r.updateHints()
				return r, nil
			}

		case "ctrl+z":
			return r, func() tea.Msg { return UndoMsg{} }

		case "ctrl+y":
			return r, func() tea.Msg { return RedoMsg{} }
		}

	case ToggleFilterMsg:
		if r.content.schema != nil {
			r.content.configuredOnly = !r.content.configuredOnly
			r.content.refilter()
			r.content.syncDetail()
		}
		r.updateHints()
		return r, nil

	case CycleThemeMsg:
		r.cycleTheme()
		themeMsg := ThemeChangedMsg{Theme: r.app.Theme}
		var cmd tea.Cmd
		r.sidebar, cmd = r.sidebar.Update(themeMsg)
		cmds = append(cmds, cmd)
		r.content, cmd = r.content.Update(themeMsg)
		cmds = append(cmds, cmd)
		r.status.theme = r.app.Theme
		r.status.themeName = r.app.Theme.Palette.Name
		r.status.refreshStyles()
		return r, tea.Batch(cmds...)

	case ToggleNewMsg:
		if r.focus == paneContent && len(r.content.searchMatches) == 0 {
			r.toggleNewFilter()
			r.updateHints()
		}
		return r, nil

	case OpenEditorMsg:
		if r.focus == paneContent && r.content.config != nil && r.content.config.Path != "" {
			return r, r.openInEditor()
		}
		return r, nil

	case ToggleHelpMsg:
		r.showHelp = !r.showHelp
		r.updateHints()
		return r, nil

	case AskOpenMsg:
		if r.installed["claude"] {
			cmd := r.ask.Open()
			r.ask.width = r.width
			r.ask.height = r.height
			return r, cmd
		}
		return r, nil

	case JumpToFieldMsg:
		if len(r.content.fields) == 0 {
			return r, nil
		}
		// find the visible row matching this field index
		for vi, row := range r.content.visible {
			if !row.isSection && row.fieldIdx == msg.FieldIdx {
				r.content.cursor = vi
				r.content.syncDetail()
				r.focusPane(paneContent)
				r.updateHints()
				break
			}
		}
		return r, nil

	case SelectAppMsg:
		if msg.Index >= 0 && msg.Index < len(r.allKonfables) {
			r.pushNav()
			r.sidebar.setCursorToApp(msg.Index)
			return r, func() tea.Msg {
				return AppSelectedMsg{Index: msg.Index, Confirmed: true}
			}
		}
		return r, nil

	case SaveMsg:
		if r.content.config != nil && r.content.config.Dirty() {
			r.content.syncDiffView()
			r.showDiffPreview = true
			return r, nil
		}
		return r, nil

	case AppSelectedMsg:
		if msg.Index == -1 {
			if msg.Confirmed || r.app.Config.BrowseLoadsApp {
				r.content.showDashboard()
				r.status.status = ""
			}
			return r, nil
		}
		if msg.Index >= 0 && msg.Index < len(r.allKonfables) {
			if !msg.Confirmed && !r.app.Config.BrowseLoadsApp {
				return r, nil // browse only — don't load
			}
			// guard: warn before discarding unsaved edits
			if r.content.config != nil && r.content.config.Dirty() && !r.confirmSwitch {
				r.confirmSwitch = true
				r.pendingSwitch = &msg
				r.status.status = "unsaved changes — press again to switch, ctrl+s to save"
				return r, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
					return confirmSwitchClearMsg{}
				})
			}
			r.confirmSwitch = false
			r.pendingSwitch = nil
			k := r.allKonfables[msg.Index]
			r.status.status = ""
			cmd := r.content.loadApp(k)
			cmds = append(cmds, cmd)
			if msg.Confirmed {
				r.focusPane(paneContent)
			}
		}
		return r, tea.Batch(cmds...)

	case StatusMsg:
		r.status.status = msg.Text
		return r, nil

	case ErrorMsg:
		r.status.status = "error: " + msg.Err.Error()
		return r, nil

	case ExternalChangeMsg:
		var cmd tea.Cmd
		r.content, cmd = r.content.Update(msg)
		r.content.fileState = "reloaded"
		cmds = append(cmds, cmd)
		return r, tea.Batch(cmds...)

	case appLoadedMsg:
		// guard against stale loads after rapid app switching
		if r.content.konfable == nil || r.content.konfable.Name() != msg.appName {
			return r, nil
		}
		if msg.err == nil && msg.config != nil {
			r.content.config = msg.config
			r.content.config.Path = msg.path
			if msg.isNew {
				r.content.fileState = "new"
			}
		}
		r.content.refreshValues()
		r.content.snapshotOrigValues()
		// start watching if the konfable supports it
		if r.content.config != nil && r.content.program != nil {
			if w, ok := r.content.konfable.(pkg.Watchable); ok {
				p := r.content.program
				cfPath := r.content.config.Path
				_ = w.Watch(func() {
					p.Send(ExternalChangeMsg{Path: cfPath})
				})
			}
		}
		// fulfill pending jump from ask AI
		if r.pendingJump != nil {
			key := r.pendingJump.fieldKey
			r.pendingJump = nil
			r.content.jumpToFieldByKey(key)
		}
		r.updateHints()
		return r, nil

	case reloadResultMsg:
		if r.content.config != nil {
			r.content.refreshValues()
			r.content.snapshotOrigValues()
			switch msg.source {
			case "editor-dirty":
				r.content.fileState = "reloaded from $EDITOR"
				r.status.status = "reloaded — in-TUI edits replaced by $EDITOR"
			case "editor":
				r.content.fileState = ""
			case "external":
				r.content.fileState = "reloaded"
			}
		}
		r.updateHints()
		return r, nil

	case saveResultMsg:
		if msg.err != nil {
			r.status.status = "save failed: " + msg.err.Error()
		} else {
			r.content.fileState = "saved"
			r.content.snapshotOrigValues()
			info := r.currentAppInfo()
			if info.AutoReload {
				r.status.status = "saved (live reload)"
				return r, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
					return fileStateClearMsg{}
				})
			}
			if len(info.ReloadCmd) > 0 {
				r.status.status = "saved, reloading..."
				reloadCmd := info.ReloadCmd
				return r, func() tea.Msg {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					err := exec.CommandContext(ctx, reloadCmd[0], reloadCmd[1:]...).Run()
					return postSaveReloadMsg{err: err}
				}
			}
			r.status.status = "saved"
			return r, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
				return fileStateClearMsg{}
			})
		}
		return r, nil

	case postSaveReloadMsg:
		if msg.err != nil {
			r.status.status = "saved (reload failed)"
		} else {
			r.status.status = "saved + reloaded"
		}
		return r, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			return fileStateClearMsg{}
		})

	case fileStateClearMsg:
		if r.content.config != nil && r.content.config.Dirty() {
			r.content.fileState = "unsaved"
		} else {
			r.content.fileState = ""
		}
		return r, nil

	case confirmQuitClearMsg:
		r.confirmQuit = false
		if r.status.status == "unsaved changes — q to quit, ctrl+s to save" {
			r.status.status = ""
		}
		return r, nil

	case confirmSwitchClearMsg:
		r.confirmSwitch = false
		r.pendingSwitch = nil
		if r.status.status == "unsaved changes — press again to switch, ctrl+s to save" {
			r.status.status = ""
		}
		return r, nil

	case statusClearMsg:
		r.status.status = ""
		return r, nil

	case tea.ClipboardMsg:
		if r.clipboardPending {
			r.clipboardPending = false
			r.applyPaste(msg.Content)
			if r.content.config != nil && r.content.config.Dirty() {
				r.content.fileState = "unsaved"
			}
			// only show "pasted" if applyPaste didn't set an error
			if !strings.HasPrefix(r.status.status, "paste failed") {
				r.status.status = "pasted"
			}
			r.updateHints()
			return r, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
				return statusClearMsg{}
			})
		}
		return r, nil

	case UndoMsg:
		var cmd tea.Cmd
		r.content, cmd = r.content.Update(msg)
		if r.content.config != nil && r.content.config.Dirty() {
			r.content.fileState = "unsaved"
		}
		r.updateHints()
		return r, cmd

	case RedoMsg:
		var cmd tea.Cmd
		r.content, cmd = r.content.Update(msg)
		if r.content.config != nil && r.content.config.Dirty() {
			r.content.fileState = "unsaved"
		}
		r.updateHints()
		return r, cmd

	case DocOpenedMsg:
		if msg.URL != "" {
			r.status.status = "opened docs"
		} else {
			r.status.status = "no docs available"
		}
		r.updateHints()
		return r, nil

	case KonfSettingChangedMsg:
		switch msg.Key {
		case "theme":
			p := theme.PaletteByName(msg.Value)
			r.app.Theme.SetPalette(p)
			themeMsg := ThemeChangedMsg{Theme: r.app.Theme}
			var cmd tea.Cmd
			r.sidebar, cmd = r.sidebar.Update(themeMsg)
			cmds = append(cmds, cmd)
			r.content, cmd = r.content.Update(themeMsg)
			cmds = append(cmds, cmd)
			r.status.theme = r.app.Theme
			r.status.themeName = r.app.Theme.Palette.Name
			r.status.refreshStyles()
			r.app.Config.Theme = msg.Value
		case "log_level":
			r.app.Config.LogLevel = msg.Value
		case "browse_loads_app":
			r.app.Config.BrowseLoadsApp = msg.Value == "true"
		}
		cfg := r.app.Config
		cmds = append(cmds, func() tea.Msg {
			_ = setup.SaveConfig(cfg)
			return nil
		})
		return r, tea.Batch(cmds...)

	case EditorExitMsg:
		// reload config after external editor exits — async to avoid blocking
		if r.content.config != nil {
			wasDirty := r.content.config.Dirty()
			cfg := r.content.config
			r.updateHints()
			return r, func() tea.Msg {
				_ = cfg.Reload(context.Background())
				source := "editor"
				if wasDirty {
					source = "editor-dirty"
				}
				return reloadResultMsg{source: source}
			}
		}
		r.updateHints()
		return r, nil

	case insightTickMsg, splitFlapTickMsg, logoAnimTickMsg:
		var cmd tea.Cmd
		r.content, cmd = r.content.Update(msg)
		return r, cmd
	}

	// bounce nav keys from empty content back to sidebar
	if km, ok := msg.(tea.KeyPressMsg); ok && r.focus == paneContent {
		switch km.String() {
		case "j", "k", "up", "down":
			if c := &r.content; c.schema == nil && c.config == nil {
				r.focusPane(paneSidebar)
			}
		}
	}

	// fan-out to focused child
	var cmd tea.Cmd
	switch r.focus {
	case paneSidebar:
		r.sidebar, cmd = r.sidebar.Update(msg)
	case paneContent:
		r.content, cmd = r.content.Update(msg)
		// sync file state after potential bool toggle
		if r.content.config != nil && r.content.config.Dirty() {
			r.content.fileState = "unsaved"
		}
	}
	cmds = append(cmds, cmd)
	r.updateHints()

	return r, tea.Batch(cmds...)
}

func (r *root) View() tea.View {
	v := tea.View{
		AltScreen: true,
	}

	if !r.ready {
		v.Content = "loading..."
		return v
	}

	r.layout()

	// track dirty apps for sidebar indicator
	changes := r.content.pendingChanges()
	if r.sidebar.dirtyApps == nil {
		r.sidebar.dirtyApps = make(map[string]bool)
	}
	if r.content.konfable != nil {
		name := r.content.konfable.Name()
		r.sidebar.dirtyApps[name] = len(changes) > 0
	}

	// pass change count to statusbar
	r.status.changeCount = len(changes)

	sidebarView := r.sidebar.View()
	contentView := r.content.View()
	statusView := r.status.View()

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, contentView)
	content := lipgloss.JoinVertical(lipgloss.Left, body, statusView)

	if r.showHelp {
		v.Content = renderHelpCard(r.width, r.height, r.focus, r.content.Editing(), r.app.Theme)
		return v
	}

	if r.showDiffPreview {
		v.Content = r.renderDiffPreview()
		return v
	}

	if r.ask.Visible() {
		r.ask.width = r.width
		r.ask.height = r.height
		v.Content = r.ask.View()
		return v
	}

	if r.palette.Visible() {
		r.palette.width = r.width
		r.palette.height = r.height
		v.Content = r.palette.View()
		return v
	}

	v.Content = content
	return v
}

// sidebarW returns the sidebar width based on terminal width.
func (r *root) sidebarW() int {
	if r.width < 60 {
		return 14
	}
	if r.width >= 120 {
		return 24
	}
	return 20
}

func (r *root) layout() {
	// statusbar: 1 line at bottom
	r.status.width = r.width

	// sidebar: wider panel, full height minus statusbar
	bodyH := r.height - 1
	if bodyH < 3 {
		bodyH = 3
	}

	sw := r.sidebarW()
	r.sidebar.width = sw
	r.sidebar.height = bodyH

	// content: fill remaining width
	r.content.width = r.width - sw
	if r.content.width < 10 {
		r.content.width = 10
	}
	r.content.height = bodyH
}

func (r *root) focusPane(p pane) {
	r.sidebar.focused = false
	r.content.focused = false
	r.focus = p

	switch p {
	case paneSidebar:
		r.sidebar.focused = true
	case paneContent:
		r.content.focused = true
	}
	r.updateHints()
}

func (r *root) cyclePane(dir int) {
	panes := []pane{paneSidebar, paneContent}
	cur := 0
	for i, p := range panes {
		if p == r.focus {
			cur = i
			break
		}
	}
	next := (cur + dir + len(panes)) % len(panes)
	r.focusPane(panes[next])
}

func (r *root) updateHints() {
	// wire mode indicator and undo count into statusbar
	switch {
	case r.content.Editing():
		r.status.SetMode("EDIT")
	case r.content.searching || r.sidebar.searching:
		r.status.SetMode("SEARCH")
	default:
		r.status.SetMode("")
	}
	r.status.SetUndoCount(r.content.undoStack.Len())

	if r.content.Editing() {
		switch r.content.detail.editor.(type) {
		case *listEditor, *hookEditor, *structListEditor:
			r.content.hints = []keyHint{
				{"a", "add"}, {"d", "delete"},
				{"⏎", "edit"}, {"^S", "done"}, {"esc", "cancel"},
			}
		case *toggleMapEditor:
			r.content.hints = []keyHint{
				{"␣", "toggle"}, {"a", "add"}, {"d", "delete"},
				{"⏎", "done"}, {"esc", "cancel"},
			}
		case *enumEditor:
			r.content.hints = []keyHint{
				{"↑↓", "select"}, {"⏎", "confirm"}, {"esc", "cancel"},
			}
		default:
			r.content.hints = []keyHint{
				{"⏎", "confirm"}, {"esc", "cancel"}, {"tab", "switch mode"},
			}
		}
		r.status.hints = r.content.hints
		return
	}

	switch r.focus {
	case paneSidebar:
		if r.sidebar.searching {
			r.content.hints = []keyHint{
				{"←↑↓", "navigate"},
				{"⏎", "select"},
				{"esc", "clear"},
			}
		} else {
			hints := []keyHint{
				{"←↑↓", "navigate"},
				{"⏎", "open"},
				{"1-9", "jump"},
				{"/", "search"},
				{"^K", "palette"},
			}
			if r.installed["claude"] {
				hints = append(hints, keyHint{"^A", "ask"})
			}
			hints = append(hints, []keyHint{
				{"→", "content"},
				{"t", "theme"},
				{"?", "help"},
				{"q", "quit"},
			}...)
			r.content.hints = hints
		}
	case paneContent:
		if r.content.searching {
			r.content.hints = []keyHint{
				{"←↑↓", "navigate"},
				{"⏎", "lock"},
				{"esc", "clear"},
			}
			r.status.hints = r.content.hints
			return
		}
		hints := []keyHint{
			{"←↑↓", "navigate"},
			{"⏎", "edit"},
			{"⌫", "revert"},
			{"d", "delete"},
			{"JK", "scroll detail"},
			{"/", "search"},
		}
		hints = append(hints, []keyHint{
			{"y", "copy"},
			{"p", "paste"},
		}...)
		if len(r.content.searchMatches) > 0 {
			hints = append(hints, keyHint{"nN", "next/prev match"})
		} else if r.newCounts[r.content.title] > 0 {
			hints = append(hints, keyHint{"w", "what's new"})
		}
		hints = append(hints, keyHint{"f", "filter"})
		if r.content.currentDocURL() != "" {
			hints = append(hints, keyHint{"o", "docs"})
		}
		if r.content.config != nil && r.content.config.Path != "" {
			hints = append(hints, keyHint{"e", "$EDITOR"})
		}
		hints = append(hints, []keyHint{
			{"←", "back"},
			{"t", "theme"},
		}...)
		if r.content.config != nil && r.content.config.Dirty() {
			hints = append(hints, keyHint{"^S", "save"})
		}
		if r.installed["claude"] {
			hints = append(hints, keyHint{"^A", "ask"})
		}
		hints = append(hints, []keyHint{
			{"?", "help"},
			{"q", "quit"},
		}...)
		r.content.hints = hints
	}
	r.status.hints = r.content.hints
}

func (r *root) cycleTheme() {
	palettes := theme.Palettes
	current := r.app.Theme.Palette.Name
	nextIdx := 0
	for i := range palettes {
		if palettes[i].Name == current {
			nextIdx = (i + 1) % len(palettes)
			break
		}
	}
	r.app.Theme.SetPalette(&palettes[nextIdx])
}

// buildPaletteItems creates the command palette entries from current state.
func (r *root) buildPaletteItems() []PaletteItem {
	var items []PaletteItem

	// app entries
	for i, k := range r.allKonfables {
		idx := i
		items = append(items, PaletteItem{
			Label:      k.Name(),
			Category:   "app",
			MatchTerms: k.Name(),
			Action:     SelectAppMsg{Index: idx},
		})
	}

	// actions
	items = append(items, PaletteItem{
		Label:      "Save",
		Shortcut:   "ctrl+s",
		Category:   "action",
		MatchTerms: "save write",
		Action:     SaveMsg{},
	})
	items = append(items, PaletteItem{
		Label:      "Undo",
		Shortcut:   "ctrl+z",
		Category:   "action",
		MatchTerms: "undo revert",
		Action:     UndoMsg{},
	})
	items = append(items, PaletteItem{
		Label:      "Redo",
		Shortcut:   "ctrl+y",
		Category:   "action",
		MatchTerms: "redo repeat",
		Action:     RedoMsg{},
	})
	items = append(items, PaletteItem{
		Label:      "Toggle Configured Filter",
		Shortcut:   "f",
		Category:   "action",
		MatchTerms: "filter configured only",
		Action:     ToggleFilterMsg{},
	})
	items = append(items, PaletteItem{
		Label:      "Cycle Theme",
		Shortcut:   "t",
		Category:   "action",
		MatchTerms: "theme cycle color palette",
		Action:     CycleThemeMsg{},
	})
	items = append(items, PaletteItem{
		Label:      "What's New",
		Shortcut:   "w",
		Category:   "action",
		MatchTerms: "new version fields",
		Action:     ToggleNewMsg{},
	})
	items = append(items, PaletteItem{
		Label:      "Open in $EDITOR",
		Shortcut:   "e",
		Category:   "action",
		MatchTerms: "editor vim nvim external",
		Action:     OpenEditorMsg{},
	})
	items = append(items, PaletteItem{
		Label:      "Help",
		Shortcut:   "?",
		Category:   "action",
		MatchTerms: "help keybindings shortcuts",
		Action:     ToggleHelpMsg{},
	})

	if r.installed["claude"] {
		items = append(items, PaletteItem{
			Label:      "Ask AI",
			Shortcut:   "ctrl+a",
			Category:   "action",
			MatchTerms: "ask ai claude natural language search find",
			Action:     AskOpenMsg{},
		})
	}

	return items
}

// buildFieldItems creates palette entries for the current app's schema fields.
func (r *root) buildFieldItems() []PaletteItem {
	if len(r.content.fields) == 0 {
		return nil
	}
	app := ""
	if r.content.konfable != nil {
		app = r.content.konfable.Name()
	}
	items := make([]PaletteItem, 0, len(r.content.fields))
	for i, f := range r.content.fields {
		section := ""
		if r.content.schema != nil && i < len(r.content.fieldSection) {
			si := r.content.fieldSection[i]
			if si < len(r.content.schema.Sections) {
				section = r.content.schema.Sections[si].Name
			}
		}
		desc := f.Type
		if f.Description != "" {
			desc = f.Description
		}
		terms := f.Key + " " + section + " " + f.Type
		if f.Description != "" {
			terms += " " + f.Description
		}
		cat := app
		if section != "" {
			cat = section
		}
		items = append(items, PaletteItem{
			Label:       f.Key,
			Description: desc,
			Category:    cat,
			MatchTerms:  terms,
			Action:      JumpToFieldMsg{FieldIdx: i},
		})
	}
	return items
}

// pushNav records the current position in the navigation history.
func (r *root) pushNav() {
	entry := navEntry{
		appIndex: r.sidebar.appIndex(),
	}
	// truncate forward history when pushing
	if r.navHistoryPos < len(r.navHistory) {
		r.navHistory = r.navHistory[:r.navHistoryPos]
	}
	r.navHistory = append(r.navHistory, entry)
	r.navHistoryPos = len(r.navHistory)
}

// navBack navigates to the previous position in history.
func (r *root) navBack() tea.Cmd {
	if r.navHistoryPos <= 0 || len(r.navHistory) == 0 {
		return nil
	}
	// save current position at the tip so forward can return here
	if r.navHistoryPos >= len(r.navHistory) {
		r.navHistory = append(r.navHistory, navEntry{appIndex: r.sidebar.appIndex()})
	}
	r.navHistoryPos--
	entry := r.navHistory[r.navHistoryPos]
	r.sidebar.setCursorToApp(entry.appIndex)
	idx := entry.appIndex
	return func() tea.Msg {
		return AppSelectedMsg{Index: idx, Confirmed: true}
	}
}

// navForward navigates to the next position in history.
func (r *root) navForward() tea.Cmd {
	if r.navHistoryPos >= len(r.navHistory)-1 {
		return nil
	}
	r.navHistoryPos++
	entry := r.navHistory[r.navHistoryPos]
	r.sidebar.setCursorToApp(entry.appIndex)
	idx := entry.appIndex
	return func() tea.Msg {
		return AppSelectedMsg{Index: idx, Confirmed: true}
	}
}

// renderDiffPreview renders a centered card with the diff view and confirmation hints.
func (r *root) renderDiffPreview() string {
	th := r.app.Theme

	cardW := r.width * 60 / 100
	if cardW < 40 {
		cardW = 40
	}
	if cardW > r.width-4 {
		cardW = r.width - 4
	}
	cardH := r.height * 70 / 100
	if cardH < 10 {
		cardH = 10
	}
	if cardH > r.height-4 {
		cardH = r.height - 4
	}

	innerW := cardW - 4
	innerH := cardH - 6 // border + padding + header + footer

	r.content.diffView.SetSize(innerW, innerH)

	var b strings.Builder
	b.WriteString(th.Primary.Bold(true).Render("  save changes?"))
	b.WriteString("\n\n")
	b.WriteString(r.content.diffView.View())
	b.WriteString("\n\n")
	b.WriteString(th.Muted.Italic(true).Render("  enter save  esc cancel"))

	card := helpCardStyle.
		BorderForeground(th.Palette.Primary).
		Width(cardW).
		Height(cardH).
		Render(b.String())

	return lipgloss.Place(r.width, r.height,
		lipgloss.Center, lipgloss.Center,
		card,
		lipgloss.WithWhitespaceChars(" "),
	)
}

// computeNewCounts counts fields with a `since` matching the detected version per app.
// also populates a schema cache to avoid re-parsing in loadApp.
func computeNewCounts(allK []konfables.Konfable, versions map[string]string) (map[string]int, map[string]*pkg.Schema) {
	counts := make(map[string]int)
	cache := make(map[string]*pkg.Schema)
	for _, k := range allK {
		schemaData, err := k.Schema()
		if err != nil || schemaData == nil {
			continue
		}
		s, err := pkg.LoadSchema(schemaData)
		if err != nil {
			continue
		}
		cache[k.Name()] = s

		ver, ok := versions[k.Name()]
		if !ok || ver == "" {
			continue
		}
		nv := pkg.NormalizeSemver(ver)
		if nv == "" {
			continue
		}
		count := 0
		for si := range s.Sections {
			for fi := range s.Sections[si].Fields {
				ns := pkg.NormalizeSemver(s.Sections[si].Fields[fi].Since)
				if ns == "" {
					continue
				}
				if semver.MajorMinor(ns) == semver.MajorMinor(nv) {
					count++
				}
			}
		}
		if count > 0 {
			counts[k.Name()] = count
		}
	}
	return counts, cache
}

// yankField copies the current field's value to the system clipboard.
func (r *root) yankField() tea.Cmd {
	f := r.content.currentField()
	if f == nil {
		return nil
	}
	val, ok := r.content.values[f.Key]
	if !ok || val == "" {
		val = f.Default
	}
	if val == "" {
		return nil
	}
	r.status.status = "copied!"
	return tea.Batch(
		tea.SetClipboard(val),
		tea.Tick(2*time.Second, func(time.Time) tea.Msg {
			return statusClearMsg{}
		}),
	)
}

// pasteField initiates a clipboard read for pasting into the focused field.
func (r *root) pasteField() tea.Cmd {
	if r.content.currentField() == nil {
		return nil
	}
	r.clipboardPending = true
	return tea.ReadClipboard
}

// applyPaste writes clipboard content to the focused field with undo support.
func (r *root) applyPaste(value string) {
	f := r.content.currentField()
	if f == nil || r.content.konfable == nil || r.content.config == nil {
		return
	}
	p := r.content.konfable.Parser()
	if p == nil {
		return
	}
	oldVal := r.content.values[f.Key]
	if value == oldVal {
		return
	}
	data := r.content.config.Content()

	// list fields need SetValues for repeated-key formats
	if f.Type == "list" {
		if mvp, ok := p.(konfables.MultiValueParser); ok {
			var vals []string
			if value != "" {
				vals = strings.Split(value, "\n")
			}
			newData, err := mvp.SetValues(data, f.Key, vals)
			if err != nil {
				r.status.status = "paste failed: " + err.Error()
				return
			}
			r.content.config.SetContent(newData)
			r.content.undoStack.Push(EditOp{FieldKey: f.Key, OldValue: oldVal, NewValue: value})
			r.content.refreshValues()
			return
		}
	}

	serialized := formatValue(value, f.Type, r.content.konfable.Info().Format)
	newData, err := p.SetValue(data, f.Key, serialized)
	if err != nil {
		r.status.status = "paste failed: " + err.Error()
		return
	}
	r.content.config.SetContent(newData)
	r.content.undoStack.Push(EditOp{FieldKey: f.Key, OldValue: oldVal, NewValue: value})
	r.content.refreshValues()
}

// toggleNewFilter toggles the "what's new" field filter on content.
func (r *root) toggleNewFilter() {
	r.content.showNewOnly = !r.content.showNewOnly
	r.content.refilter()
	r.content.syncDetail()
	if r.content.showNewOnly {
		name := ""
		if r.content.konfable != nil {
			name = r.content.konfable.Name()
		}
		count := r.newCounts[name]
		r.status.status = fmt.Sprintf("showing %d new fields", count)
	} else {
		r.status.status = ""
	}
}

// openInEditor launches $EDITOR (or $VISUAL, fallback vim) on the config file.
func (r *root) openInEditor() tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		return func() tea.Msg {
			return StatusMsg{Text: "$EDITOR not set"}
		}
	}

	path := r.content.config.Path
	args := []string{path}

	// pass +LINE for editors that support it (vim, nvim, nano, emacs, etc.)
	if r.content.detail.previewFound && r.content.detail.previewLine >= 0 {
		args = []string{fmt.Sprintf("+%d", r.content.detail.previewLine+1), path}
	}

	cmd := exec.Command(editor, args...)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return EditorExitMsg{Err: err}
	})
}

// statusClearMsg clears the transient status after a delay.
type statusClearMsg struct{}

// currentAppInfo returns the AppInfo for the currently loaded konfable.
func (r *root) currentAppInfo() konfables.AppInfo {
	if r.content.konfable != nil {
		return r.content.konfable.Info()
	}
	return konfables.AppInfo{}
}

// toggleBookmark adds or removes a bookmark for the focused field.
func (r *root) toggleBookmark() tea.Cmd {
	f := r.content.currentField()
	if f == nil || r.content.konfable == nil {
		return nil
	}
	key := r.content.konfable.Name() + "/" + f.Key
	if r.content.bookmarks[key] {
		delete(r.content.bookmarks, key)
		r.status.status = "bookmark removed"
	} else {
		r.content.bookmarks[key] = true
		r.status.status = "bookmark added"
	}
	// persist
	bm := make([]string, 0, len(r.content.bookmarks))
	for k := range r.content.bookmarks {
		bm = append(bm, k)
	}
	r.app.Config.Bookmarks = bm
	cfg := r.app.Config
	return func() tea.Msg {
		_ = setup.SaveConfig(cfg)
		return statusClearMsg{}
	}
}
