package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/eminert/konfi/konfables"
	"github.com/eminert/konfi/pkg"
	"github.com/eminert/konfi/setup"
	"github.com/eminert/konfi/theme"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

type root struct {
	app         *setup.App
	sidebar     sidebar
	content     content
	status      statusbar
	palette     palette
	focus       pane
	mode        AppMode
	width       int
	height      int
	ready       bool
	showHelp    bool
	confirmQuit bool

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

	// unsaved per-app working copies kept while browsing other apps
	dirtyConfigs map[string]dirtyConfigState

	// cached parsed schemas — populated by computeNewCounts, reused by loadApp
	schemaCache map[string]*pkg.Schema

	// diff confirmation overlay before saving
	showDiffPreview bool

	// live preview state — file written to disk but not committed
	previewing bool

	// program ref for sending messages from outside the event loop
	program *tea.Program
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

	// parse bookmarks from config
	ct.bookmarks = make(map[string]bool)
	for _, b := range app.Config.Bookmarks {
		ct.bookmarks[b] = true
	}

	ct.dashboardApps = buildDashboardApps(allK, installed, nerdFont, app.Versions, schemaCache, newCounts)

	st := newStatusbar(th)
	pal := newPalette(th)

	r := &root{
		app:          app,
		sidebar:      sb,
		content:      ct,
		status:       st,
		palette:      pal,
		focus:        paneSidebar,
		mode:         ModeNormal,
		allKonfables: allK,
		installed:    installed,
		newCounts:    newCounts,
		dirtyConfigs: make(map[string]dirtyConfigState),
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
	if model, cmd, done := r.interceptModal(msg); done {
		return model, cmd
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		r.width = msg.Width
		r.height = msg.Height
		r.ready = true
		r.layout()
		return r, nil
	case tea.KeyPressMsg:
		return r.updateKey(msg)
	default:
		return r.updateMessage(msg)
	}
}

// interceptModal handles overlay, edit, and search states that capture input
// before the normal keymap runs. it returns handled=true when the event was
// consumed and Update should return immediately.
func (r *root) interceptModal(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	// palette intercepts ALL input when visible
	if r.palette.Visible() {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			p, cmd := r.palette.Update(msg)
			r.palette = p
			return r, cmd, true
		case PaletteSelectedMsg:
			r.palette.Close()
			// re-dispatch the inner action
			m, cmd := r.Update(msg.Action)
			return m, cmd, true
		case PaletteClosedMsg:
			r.palette.Close()
			return r, nil, true
		default:
			p, cmd := r.palette.Update(msg)
			r.palette = p
			return r, cmd, true
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
				}, true
			case "p":
				info := r.currentAppInfo()
				if info.AutoReload {
					r.showDiffPreview = false
					r.previewing = true
					r.status.status = "previewing..."
					cfg := r.content.config
					return r, func() tea.Msg {
						err := cfg.Preview(context.Background())
						return previewResultMsg{err: err}
					}, true
				}
			case "esc", "n":
				r.showDiffPreview = false
				r.status.status = ""
				return r, nil, true
			}
			return r, nil, true
		}
	}

	// preview mode — esc reverts, ctrl+s keeps
	if r.previewing {
		if km, ok := msg.(tea.KeyPressMsg); ok {
			switch km.String() {
			case "ctrl+s":
				r.previewing = false
				r.status.status = "saving..."
				cfg := r.content.config
				return r, func() tea.Msg {
					err := cfg.Save(context.Background())
					return saveResultMsg{err: err}
				}, true
			case "esc":
				r.previewing = false
				r.status.status = "reverting..."
				cfg := r.content.config
				return r, func() tea.Msg {
					err := cfg.RevertPreview(context.Background())
					return revertPreviewResultMsg{err: err}
				}, true
			}
		}
	}

	// when content is in edit mode, only ctrl+c stays at root level —
	// everything else passes through so esc/blink/keys reach the editor.
	if r.content.Editing() {
		if km, ok := msg.(tea.KeyPressMsg); ok && km.String() == "ctrl+c" {
			if r.content.config != nil {
				r.content.stopWatching()
			}
			return r, tea.Quit, true
		}
		var cmd tea.Cmd
		r.content, cmd = r.content.Update(msg)
		// sync file state after edit commits
		if r.content.config != nil && r.content.config.Dirty() {
			r.content.fileState = "unsaved"
		}
		r.updateHints()
		return r, tea.Batch(cmd), true
	}

	// when content is searching, forward keys to content (same pattern as sidebar search)
	if r.content.searching {
		if km, ok := msg.(tea.KeyPressMsg); ok {
			if km.String() == "ctrl+c" {
				if r.content.config != nil {
					r.content.stopWatching()
				}
				return r, tea.Quit, true
			}
			var cmd tea.Cmd
			r.content, cmd = r.content.Update(msg)
			r.updateHints()
			return r, cmd, true
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
			return r, nil, true
		}
	}

	return r, nil, false
}

func (r *root) updateKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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

	switch msg.String() {
	case "ctrl+c":
		if r.previewing {
			r.previewing = false
			if r.content.config != nil {
				_ = r.content.config.RevertPreview(context.Background())
			}
		}
		if r.content.config != nil {
			r.content.stopWatching()
		}
		return r, tea.Quit

	case "q":
		if r.content.config != nil && r.content.config.Dirty() && !r.confirmQuit {
			r.confirmQuit = true
			if r.previewing {
				r.status.status = "previewing — q to quit and revert, ctrl+s to save"
			} else {
				r.status.status = "unsaved changes — q to quit, ctrl+s to save"
			}
			return r, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
				return confirmQuitClearMsg{}
			})
		}
		if r.previewing {
			r.previewing = false
			if r.content.config != nil {
				_ = r.content.config.RevertPreview(context.Background())
			}
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

	case ".":
		r.toggleConfiguredFilter()
		return r, nil

	case "tab":
		if r.focus == paneContent && !r.content.detailFocused {
			r.toggleChangedFilter()
			return r, nil
		}

	case "left":
		if r.focus == paneContent && r.content.detailFocused {
			break
		}
		if r.focus != paneSidebar {
			r.focusPane(paneSidebar)
		}
		return r, nil

	case "right":
		if r.focus == paneSidebar {
			var cmd tea.Cmd
			r.sidebar, cmd = r.sidebar.selectCurrent(true)
			return r, cmd
		}

	case "esc":
		if r.focus == paneContent && r.content.detailFocused {
			break
		}
		// layered esc: clear filters before jumping to sidebar
		if cleared, clearStatus := r.content.clearTopFilter(); cleared {
			if clearStatus {
				r.status.status = ""
			}
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
		return r, tea.Batch(r.applyThemeChange()...)

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

	case "c":
		if r.focus == paneContent && !r.content.detailFocused {
			cmd := r.yankField()
			if cmd != nil {
				r.updateHints()
				return r, cmd
			}
		}

	case "p":
		if r.focus == paneContent && !r.content.detailFocused {
			cmd := r.pasteField()
			if cmd != nil {
				return r, cmd
			}
		}

	case "w":
		// only toggle "what's new" filter when content focused with no active search matches
		if r.focus == paneContent && !r.content.detailFocused && len(r.content.searchMatches) == 0 {
			r.toggleNewFilter()
			r.updateHints()
			return r, nil
		}

	case "e":
		if r.focus == paneContent && r.content.config != nil && r.content.config.Path != "" {
			cmd := r.openInEditor()
			return r, cmd
		}

	case "m":
		if r.focus == paneContent && !r.content.detailFocused && r.content.konfable != nil {
			cmd := r.toggleBookmark()
			if cmd != nil {
				r.updateHints()
				return r, cmd
			}
		}

	case "b":
		if r.focus == paneContent && !r.content.detailFocused && r.content.schema != nil {
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
	return r.fanOut(msg)
}

func (r *root) updateMessage(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case ToggleFilterMsg:
		r.toggleConfiguredFilter()
		return r, nil

	case CycleThemeMsg:
		r.cycleTheme()
		return r, tea.Batch(r.applyThemeChange()...)

	case ToggleNewMsg:
		if r.focus == paneContent && len(r.content.searchMatches) == 0 {
			r.toggleNewFilter()
			r.updateHints()
		}
		return r, nil

	case OpenEditorMsg:
		if r.focus == paneContent && r.content.config != nil && r.content.config.Path != "" {
			cmd := r.openInEditor()
			return r, cmd
		}
		return r, nil

	case ToggleHelpMsg:
		r.showHelp = !r.showHelp
		r.updateHints()
		return r, nil

	case JumpToFieldMsg:
		if len(r.content.fields) == 0 {
			return r, nil
		}
		// find the visible row matching this field index
		for vi, row := range r.content.visible {
			if row.isSection || row.fieldIdx != msg.FieldIdx {
				continue
			}
			r.content.cursor = vi
			r.content.syncDetail()
			r.focusPane(paneContent)
			r.updateHints()
			break
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
				stashedName, stashed := r.stashActiveDirtyConfig()
				r.content.showDashboard()
				if stashed {
					r.status.status = "kept unsaved changes for " + stashedName
				} else {
					r.status.status = ""
				}
			}
			return r, nil
		}
		if msg.Index >= 0 && msg.Index < len(r.allKonfables) {
			if !msg.Confirmed && !r.app.Config.BrowseLoadsApp {
				return r, nil // browse only — don't load
			}
			k := r.allKonfables[msg.Index]
			if r.content.konfable != nil && r.content.konfable.Name() == k.Name() {
				if msg.Confirmed {
					r.focusPane(paneContent)
				}
				return r, nil
			}
			stashedName, stashed := r.stashActiveDirtyConfig()
			if r.previewing {
				r.previewing = false
				if r.content.config != nil {
					_ = r.content.config.RevertPreview(context.Background())
				}
			}
			if stashed {
				r.status.status = "kept unsaved changes for " + stashedName
			} else {
				r.status.status = ""
			}
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
		restoredState, restoredDirty := r.takeDirtyConfig(msg.appName, msg.path)
		if restoredDirty {
			r.content.config = restoredState.config
			if restoredState.fileState != "" {
				r.content.fileState = restoredState.fileState
			} else {
				r.content.fileState = "unsaved"
			}
			if restoredState.undoStack != nil {
				r.content.undoStack = restoredState.undoStack.Clone()
			}
			r.status.status = "restored unsaved changes"
		} else if msg.err == nil && msg.config != nil {
			r.content.config = msg.config
			r.content.config.Path = msg.path
			if msg.isNew {
				r.content.fileState = "new"
			}
		}
		r.content.refreshValues()
		if restoredDirty {
			r.content.origValues = cloneValues(restoredState.origValues)
			r.content.cachedChangesDirty = true
			r.content.syncDiffView()
		} else {
			r.content.snapshotOrigValues()
		}
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
		r.updateHints()
		return r, nil

	case reloadResultMsg:
		if r.content.config != nil {
			wasEditorDirty := msg.source == "editor-dirty"
			r.content.refreshValues()
			r.content.snapshotOrigValues()
			applied, skipped := 0, 0
			if wasEditorDirty {
				applied, skipped = r.content.reapplyUnchangedFieldEdits(msg.fieldEdits)
			}
			r.content.syncDiffView()
			if r.content.konfable != nil && !r.content.config.Dirty() {
				r.clearDirtyConfig(r.content.konfable.Name())
			}
			switch msg.source {
			case "editor-dirty":
				if r.content.config.Dirty() {
					r.content.fileState = "unsaved"
				} else {
					r.content.fileState = "reloaded from $EDITOR"
				}
				r.status.status = editorReloadStatus(applied, skipped)
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
			if r.content.konfable != nil {
				r.clearDirtyConfig(r.content.konfable.Name())
			}
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

	case previewResultMsg:
		if msg.err != nil {
			r.previewing = false
			r.status.status = "preview failed: " + msg.err.Error()
		} else {
			r.content.fileState = "previewing"
			r.status.status = "previewing — ctrl+s keep, esc revert"
		}
		return r, nil

	case revertPreviewResultMsg:
		if msg.err != nil {
			r.status.status = "revert failed: " + msg.err.Error()
		} else {
			r.content.fileState = "unsaved"
			r.status.status = "reverted"
			return r, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
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
		switch {
		case r.previewing:
			r.content.fileState = "previewing"
		case r.content.config != nil && r.content.config.Dirty():
			r.content.fileState = "unsaved"
		default:
			r.content.fileState = ""
		}
		return r, nil

	case confirmQuitClearMsg:
		r.confirmQuit = false
		if r.status.status == "unsaved changes — q to quit, ctrl+s to save" {
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
			cmds = append(cmds, r.applyThemeChange()...)
			r.app.Config.Theme = msg.Value
		case "log_level":
			r.app.Config.LogLevel = msg.Value
		case "browse_loads_app":
			r.app.Config.BrowseLoadsApp = msg.Value == "true"
		case "nerd_font":
			// flip the cached flag everywhere it gates icon rendering. sidebar
			// and detail re-read on each frame, so the change is visible
			// immediately. dashboard tiles are baked at startup and stay on
			// their original icon until the next launch.
			on := msg.Value == "true"
			r.app.Config.NerdFont = on
			r.sidebar.nerdFont = on
			r.content.nerdFont = on
			r.content.detail.nerdFont = on
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
			fieldEdits := r.content.pendingFieldEdits()
			cfg := r.content.config
			r.updateHints()
			return r, func() tea.Msg {
				_ = cfg.Reload(context.Background())
				source := "editor"
				if wasDirty {
					source = "editor-dirty"
				}
				return reloadResultMsg{source: source, fieldEdits: fieldEdits}
			}
		}
		r.updateHints()
		return r, nil

	case insightTickMsg, splitFlapTickMsg, logoAnimTickMsg:
		var cmd tea.Cmd
		r.content, cmd = r.content.Update(msg)
		return r, cmd
	default:
		return r.fanOut(msg)
	}
}

func (r *root) fanOut(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
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
	r.sidebar.dirtyApps = make(map[string]bool, len(r.dirtyConfigs)+1)
	for name, state := range r.dirtyConfigs {
		if state.config != nil && state.config.Dirty() {
			r.sidebar.dirtyApps[name] = true
		}
	}
	if r.content.konfable != nil {
		name := r.content.konfable.Name()
		if len(changes) > 0 {
			r.sidebar.dirtyApps[name] = true
		}
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
		r.content.detailFocused = false
		r.content.syncDetail()
		r.sidebar.focused = true
	case paneContent:
		r.content.focused = true
		r.content.syncDetail()
	}
	r.updateHints()
}

// applyThemeChange propagates the active theme to the child models and refreshes
// the statusbar. returns the child update cmds for the caller to batch.
func (r *root) applyThemeChange() []tea.Cmd {
	themeMsg := ThemeChangedMsg{Theme: r.app.Theme}
	var cmds []tea.Cmd
	var cmd tea.Cmd
	r.sidebar, cmd = r.sidebar.Update(themeMsg)
	cmds = append(cmds, cmd)
	r.content, cmd = r.content.Update(themeMsg)
	cmds = append(cmds, cmd)
	r.status.theme = r.app.Theme
	r.status.themeName = r.app.Theme.Palette.Name
	r.status.refreshStyles()
	return cmds
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

// computeNewCounts counts fields with a `since` matching the detected version per app.
// also populates a schema cache to avoid re-parsing in loadApp.
func computeNewCounts(allK []konfables.Konfable, versions map[string]string) (counts map[string]int, cache map[string]*pkg.Schema) {
	counts = make(map[string]int)
	cache = make(map[string]*pkg.Schema)
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
		if pkg.NormalizeSemver(ver) == "" {
			continue
		}
		count := 0
		for si := range s.Sections {
			for fi := range s.Sections[si].Fields {
				if pkg.FieldIsNewIn(s.Sections[si].Fields[fi], ver) {
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
		tea.SetClipboard(f.Key+" = "+val),
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

// stripKeyPrefix removes a leading "<key> =" / "<key>=" assignment prefix from
// a pasted snippet, so a value copied as "key = value" pastes as just the value.
// returns value unchanged when the prefix does not match this field's key.
func stripKeyPrefix(value, key string) string {
	trimmed := strings.TrimLeft(value, " \t")
	if key == "" || !strings.HasPrefix(trimmed, key) {
		return value
	}
	rest := strings.TrimLeft(trimmed[len(key):], " \t")
	if !strings.HasPrefix(rest, "=") {
		return value
	}
	return strings.TrimLeft(rest[1:], " \t")
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
	// tolerate "key = value" snippets (the form yankField copies) so an in-app
	// copy → paste round-trips to just the value.
	value = stripKeyPrefix(value, f.Key)
	oldVal := r.content.values[f.Key]
	if value == oldVal {
		return
	}
	data := r.content.config.Content()
	newData, err := konfables.WriteField(p, data, *f, value, r.content.konfable.Info().Format)
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

func (r *root) toggleConfiguredFilter() {
	if r.content.schema == nil {
		return
	}
	r.content.configuredOnly = !r.content.configuredOnly
	r.content.refilter()
	r.content.syncDetail()
	r.updateHints()
}

func (r *root) toggleChangedFilter() {
	if r.content.schema == nil {
		return
	}
	r.content.changedOnly = !r.content.changedOnly
	r.content.refilter()
	r.content.syncDetail()
	if r.content.changedOnly {
		r.status.status = fmt.Sprintf("showing %d changed fields", len(r.content.pendingChanges()))
	} else {
		r.status.status = ""
	}
	r.updateHints()
}

// openInEditor launches $EDITOR (or $VISUAL, fallback vim) on the config file.
// honors multi-token EDITOR values like "code --wait" or "nvim --noplugin".
// quoted arguments containing whitespace are not supported — wrap in a script if you need them.
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

	tokens := strings.Fields(editor)
	if len(tokens) == 0 {
		return func() tea.Msg {
			return StatusMsg{Text: "$EDITOR is blank"}
		}
	}
	bin, editorArgs := tokens[0], tokens[1:]

	path := r.content.config.Path
	args := make([]string, 0, len(editorArgs)+2)
	args = append(args, editorArgs...)

	// pass +LINE for editors that support it (vim, nvim, nano, emacs, etc.)
	if r.content.detail.previewFound && r.content.detail.previewLine >= 0 {
		args = append(args, fmt.Sprintf("+%d", r.content.detail.previewLine+1))
	}
	args = append(args, path)

	cmd := exec.Command(bin, args...)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return EditorExitMsg{Err: err}
	})
}

func editorReloadStatus(applied, skipped int) string {
	switch {
	case applied > 0 && skipped > 0:
		return fmt.Sprintf("reloaded from $EDITOR — kept %d in-TUI edits, skipped %d changed in $EDITOR", applied, skipped)
	case applied > 0:
		return fmt.Sprintf("reloaded from $EDITOR — kept %d in-TUI edits", applied)
	case skipped > 0:
		return fmt.Sprintf("reloaded from $EDITOR — skipped %d in-TUI edits changed in $EDITOR", skipped)
	default:
		return "reloaded from $EDITOR"
	}
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
