package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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

const sidebarWidth = 20

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

	// home item at top of sidebar
	items := []sidebarItem{{
		icon:      "\uf015", // nf-fa-home
		name:      "home",
		installed: true,
		home:      true,
	}}
	var allK []konfables.Konfable
	for _, ki := range setup.AllKonfablesWithInfo() {
		if k, ok := ki.Konfable.(konfables.Konfable); ok {
			info := k.Info()
			icon := info.NerdIcon
			if icon == "" {
				icon = info.Icon
			}
			items = append(items, sidebarItem{
				icon:      icon,
				name:      k.Name(),
				installed: installed[k.Name()],
				system:    ki.System,
			})
			allK = append(allK, k)
		} else {
			app.Logger.Warn().Str("app", ki.Konfable.Name()).Msg("registered app does not satisfy full Konfable interface")
		}
	}

	// compute per-app "what's new" field counts
	newCounts := computeNewCounts(allK, app.Versions)

	sb := newSidebar(items, th)
	sb.newCounts = newCounts
	ct := newContent(th)
	ct.versions = app.Versions
	ct.appVersion = app.AppVersion

	// populate dashboard data (skip home item at index 0)
	for _, k := range allK {
		info := k.Info()
		icon := info.NerdIcon
		if icon == "" {
			icon = info.Icon
		}
		da := dashboardApp{
			icon:      icon,
			name:      k.Name(),
			installed: installed[k.Name()],
		}
		if v, ok := app.Versions[k.Name()]; ok {
			da.version = v
		}
		ct.dashboardApps = append(ct.dashboardApps, da)
	}
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
				if err := r.content.config.Save(context.Background()); err != nil {
					r.status.status = "save failed: " + err.Error()
				} else {
					r.content.fileState = "saved"
					r.content.snapshotOrigValues()
					r.status.status = "saved"
					return r, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
						return fileStateClearMsg{}
					})
				}
			}
			return r, nil

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
			if len(r.content.searchMatches) > 0 {
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
			return r, tea.Batch(cmds...)

		case "?":
			r.showHelp = true
			r.updateHints()
			return r, nil

		case "ctrl+k":
			items := r.buildPaletteItems()
			cmd := r.palette.Open(PaletteModeCommands, items)
			r.palette.width = r.width
			r.palette.height = r.height
			return r, cmd

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			idx := int(msg.String()[0]-'0') - 1
			if idx < len(r.allKonfables) {
				r.pushNav()
				r.sidebar.cursor = idx
				return r, func() tea.Msg {
					return AppSelectedMsg{Index: idx, Confirmed: true}
				}
			}
			return r, nil

		case "ctrl+n":
			cur := r.sidebar.cursor
			if cur < len(r.allKonfables)-1 {
				r.pushNav()
				r.sidebar.cursor = cur + 1
				idx := r.sidebar.cursor
				return r, func() tea.Msg {
					return AppSelectedMsg{Index: idx, Confirmed: true}
				}
			}
			return r, nil

		case "ctrl+p":
			cur := r.sidebar.cursor
			if cur > 0 {
				r.pushNav()
				r.sidebar.cursor = cur - 1
				idx := r.sidebar.cursor
				return r, func() tea.Msg {
					return AppSelectedMsg{Index: idx, Confirmed: true}
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

	case SelectAppMsg:
		if msg.Index >= 0 && msg.Index < len(r.allKonfables) {
			r.pushNav()
			r.sidebar.cursor = msg.Index
			return r, func() tea.Msg {
				return AppSelectedMsg{Index: msg.Index, Confirmed: true}
			}
		}
		return r, nil

	case SaveMsg:
		if r.content.config != nil && r.content.config.Dirty() {
			if err := r.content.config.Save(context.Background()); err != nil {
				r.status.status = "save failed: " + err.Error()
			} else {
				r.content.fileState = "saved"
				r.content.snapshotOrigValues()
				r.status.status = "saved"
				return r, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
					return fileStateClearMsg{}
				})
			}
		}
		return r, nil

	case AppSelectedMsg:
		if msg.Index == -1 {
			if msg.Confirmed {
				r.content.showDashboard()
				r.status.status = ""
			}
			return r, nil
		}
		if msg.Index >= 0 && msg.Index < len(r.allKonfables) {
			if !msg.Confirmed {
				return r, nil // browse only — don't load
			}
			k := r.allKonfables[msg.Index]
			r.status.status = ""
			cmd := r.content.loadApp(k)
			cmds = append(cmds, cmd)
			r.focusPane(paneContent)
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
			r.status.status = "pasted"
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
			r.app.Config.Theme = msg.Value
		case "log_level":
			r.app.Config.LogLevel = msg.Value
		}
		return r, tea.Batch(cmds...)

	case EditorExitMsg:
		// reload config after external editor exits
		if r.content.config != nil {
			if err := r.content.config.Reload(context.Background()); err == nil {
				r.content.refreshValues()
				r.content.snapshotOrigValues()
				r.content.fileState = ""
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

	if r.palette.Visible() {
		r.palette.width = r.width
		r.palette.height = r.height
		v.Content = r.palette.View()
		return v
	}

	v.Content = content
	return v
}

func (r *root) layout() {
	// statusbar: 1 line at bottom
	r.status.width = r.width

	// sidebar: wider panel, full height minus statusbar
	bodyH := r.height - 1
	if bodyH < 3 {
		bodyH = 3
	}

	r.sidebar.width = sidebarWidth
	r.sidebar.height = bodyH

	// content: fill remaining width
	r.content.width = r.width - sidebarWidth
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
		r.content.hints = []keyHint{
			{"⏎", "confirm"},
			{"esc", "cancel"},
			{"tab", "switch mode"},
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
			r.content.hints = []keyHint{
				{"←↑↓", "navigate"},
				{"⏎", "open"},
				{"1-9", "jump"},
				{"/", "search"},
				{"^K", "palette"},
				{"→", "content"},
				{"t", "theme"},
				{"?", "help"},
				{"q", "quit"},
			}
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

	return items
}

// pushNav records the current position in the navigation history.
func (r *root) pushNav() {
	entry := navEntry{
		appIndex: r.sidebar.cursor,
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
	// save current position for forward nav before moving back
	r.pushNav()
	// move back two: one past the push we just did, one to the actual target
	r.navHistoryPos -= 2
	if r.navHistoryPos < 0 {
		r.navHistoryPos = 0
	}
	entry := r.navHistory[r.navHistoryPos]
	r.sidebar.cursor = entry.appIndex
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
	r.sidebar.cursor = entry.appIndex
	idx := entry.appIndex
	return func() tea.Msg {
		return AppSelectedMsg{Index: idx, Confirmed: true}
	}
}

// computeNewCounts counts fields with a `since` matching the detected version per app.
// returns nil map entries for apps without version info — degrades silently.
func computeNewCounts(allK []konfables.Konfable, versions map[string]string) map[string]int {
	counts := make(map[string]int)
	for _, k := range allK {
		ver, ok := versions[k.Name()]
		if !ok || ver == "" {
			continue
		}
		nv := pkg.NormalizeSemver(ver)
		if nv == "" {
			continue
		}
		schemaData, err := k.Schema()
		if err != nil || schemaData == nil {
			continue
		}
		s, err := pkg.LoadSchema(schemaData)
		if err != nil {
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
	return counts
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
	serialized := formatValue(value, f.Type, r.content.konfable.Info().Format)
	newData, err := p.SetValue(data, f.Key, serialized)
	if err != nil {
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
