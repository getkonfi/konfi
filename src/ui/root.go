package ui

import (
	"time"

	"github.com/emin/konfigurator/konfables"
	"github.com/emin/konfigurator/setup"
	"github.com/emin/konfigurator/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type pane int

const (
	paneSidebar pane = iota
	paneContent
)

const sidebarWidth = 24

type root struct {
	app     *setup.App
	sidebar sidebar
	content content
	status  statusbar
	focus   pane
	width   int
	height  int
	ready   bool

	// all konfables (indexed by sidebar item order)
	allKonfables []konfables.Konfable
	installed    map[string]bool

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

	// build sidebar items and konfable lookup from all registered apps
	var items []sidebarItem
	var allK []konfables.Konfable
	for _, d := range setup.AllKonfables() {
		if k, ok := d.(konfables.Konfable); ok {
			info := k.Info()
			icon := info.NerdIcon
			if icon == "" {
				icon = info.Icon
			}
			items = append(items, sidebarItem{
				icon:      icon,
				name:      k.Name(),
				installed: installed[k.Name()],
			})
			allK = append(allK, k)
		} else {
			app.Logger.Warn().Str("app", d.Name()).Msg("registered app does not satisfy full Konfable interface")
		}
	}

	sb := newSidebar(items, th)
	ct := newContent(th)
	ct.versions = app.Versions
	st := newStatusbar(th)

	r := &root{
		app:          app,
		sidebar:      sb,
		content:      ct,
		status:       st,
		focus:        paneSidebar,
		allKonfables: allK,
		installed:    installed,
	}
	r.updateHints()
	return r
}

func (r *root) Init() tea.Cmd {
	// auto-select first app
	if len(r.allKonfables) > 0 {
		k := r.allKonfables[0]
		if r.installed[k.Name()] {
			return r.content.loadApp(k)
		}
		r.content.showNotInstalled(k)
		return nil
	}
	return nil
}

func (r *root) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// when content is in edit mode, only ctrl+c stays at root level —
	// everything else passes through so esc/blink/keys reach the editor.
	if r.content.editing {
		if km, ok := msg.(tea.KeyMsg); ok && km.String() == "ctrl+c" {
			if r.content.config != nil {
				r.content.config.StopWatching()
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
		if km, ok := msg.(tea.KeyMsg); ok {
			if km.String() == "ctrl+c" {
				if r.content.config != nil {
					r.content.config.StopWatching()
				}
				return r, tea.Quit
			}
			var cmd tea.Cmd
			r.content, cmd = r.content.Update(msg)
			r.updateHints()
			return r, cmd
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		r.width = msg.Width
		r.height = msg.Height
		r.ready = true
		r.layout()
		return r, nil

	case tea.KeyMsg:
		// when sidebar is searching, don't intercept keys that should reach the textinput
		if r.sidebar.searching {
			switch msg.String() {
			case "ctrl+c":
				if r.content.config != nil {
					r.content.config.StopWatching()
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

		switch msg.String() {
		case "ctrl+c", "q":
			if r.content.config != nil {
				r.content.config.StopWatching()
			}
			return r, tea.Quit

		case "ctrl+s":
			if r.content.config != nil && r.content.config.Dirty() {
				if err := r.content.config.Save(); err != nil {
					r.status.status = "save failed: " + err.Error()
				} else {
					r.content.fileState = "saved"
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
		}

	case AppSelectedMsg:
		if msg.Index >= 0 && msg.Index < len(r.allKonfables) {
			k := r.allKonfables[msg.Index]

			if !r.installed[k.Name()] {
				r.content.showNotInstalled(k)
			} else {
				cmd := r.content.loadApp(k)
				cmds = append(cmds, cmd)

				if msg.Confirmed {
					r.focusPane(paneContent)
				}
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

	case fileStateClearMsg:
		if r.content.config != nil && r.content.config.Dirty() {
			r.content.fileState = "unsaved"
		} else {
			r.content.fileState = ""
		}
		return r, nil

	case DocOpenedMsg:
		if msg.URL != "" {
			r.status.status = "opened docs"
		} else {
			r.status.status = "no docs available"
		}
		r.updateHints()
		return r, nil

	case insightTickMsg, splitFlapTickMsg:
		var cmd tea.Cmd
		r.content, cmd = r.content.Update(msg)
		return r, cmd
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

func (r *root) View() string {
	if !r.ready {
		return "loading..."
	}

	r.layout()

	sidebarView := r.sidebar.View()
	contentView := r.content.View()
	statusView := r.status.View()

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, contentView)
	return lipgloss.JoinVertical(lipgloss.Left, body, statusView)
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
	if r.content.editing {
		r.status.hints = []keyHint{
			{"⏎", "confirm"},
			{"esc", "cancel"},
		}
		return
	}

	switch r.focus {
	case paneSidebar:
		if r.sidebar.searching {
			r.status.hints = []keyHint{
				{"↑↓", "navigate"},
				{"⏎", "select"},
				{"esc", "clear"},
			}
		} else {
			r.status.hints = []keyHint{
				{"↑↓", "navigate"},
				{"⏎", "open"},
				{"/", "search"},
				{"→", "content"},
				{"t", "theme"},
				{"q", "quit"},
			}
		}
	case paneContent:
		if r.content.searching {
			r.status.hints = []keyHint{
				{"↑↓", "navigate"},
				{"⏎", "lock"},
				{"esc", "clear"},
			}
			return
		}
		hints := []keyHint{
			{"↑↓", "navigate"},
			{"⏎", "edit"},
			{"/", "search"},
			{"z", "fold"},
			{"f", "filter"},
		}
		if r.content.currentDocURL() != "" {
			hints = append(hints, keyHint{"o", "docs"})
		}
		hints = append(hints, []keyHint{
			{"←", "back"},
			{"t", "theme"},
		}...)
		if r.content.config != nil && r.content.config.Dirty() {
			hints = append(hints, keyHint{"^S", "save"})
		}
		hints = append(hints, keyHint{"q", "quit"})
		r.status.hints = hints
	}
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
