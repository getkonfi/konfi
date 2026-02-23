package ui

import (
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

const railWidth = 5

type root struct {
	app     *setup.App
	sidebar sidebar
	content content
	status  statusbar
	focus   pane
	width   int
	height  int
	ready   bool

	// detected konfables for lookup by sidebar index
	detected []konfables.Konfable

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

	// build sidebar items and konfable lookup from detected apps
	var items []sidebarItem
	var detected []konfables.Konfable
	for _, d := range app.Detected {
		if k, ok := d.(konfables.Konfable); ok {
			info := k.Info()
			icon := info.NerdIcon
			if icon == "" {
				icon = info.Icon
			}
			items = append(items, sidebarItem{icon: icon, name: k.Name()})
			detected = append(detected, k)
		} else {
			app.Logger.Warn().Str("app", d.Name()).Msg("detected app does not satisfy full Konfable interface")
		}
	}

	sb := newSidebar(items, th)
	ct := newContent(th)
	ct.versions = app.Versions
	st := newStatusbar(th)

	r := &root{
		app:      app,
		sidebar:  sb,
		content:  ct,
		status:   st,
		focus:    paneSidebar,
		detected: detected,
	}
	r.updateHints()
	return r
}

func (r *root) Init() tea.Cmd {
	// auto-select first app if any detected
	if len(r.detected) > 0 {
		k := r.detected[0]
		r.status.appVersion = r.app.Versions[k.Name()]
		return r.content.loadApp(k)
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
		// sync dirty state after edit commits
		if r.content.config != nil {
			r.status.dirty = r.content.config.Dirty()
		}
		r.updateHints()
		return r, tea.Batch(cmds...)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		r.width = msg.Width
		r.height = msg.Height
		r.ready = true
		r.layout()
		return r, nil

	case tea.KeyMsg:
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
					r.status.dirty = false
					r.status.status = "saved"
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
		if msg.Index >= 0 && msg.Index < len(r.detected) {
			k := r.detected[msg.Index]
			cmd := r.content.loadApp(k)
			if r.content.config != nil {
				r.status.filePath = r.content.config.Path
				r.status.dirty = r.content.config.Dirty()
			} else {
				r.status.filePath = k.ConfigPath()
			}
			r.status.appVersion = r.app.Versions[k.Name()]
			cmds = append(cmds, cmd)

			if msg.Confirmed {
				r.focusPane(paneContent)
			}
		}
		return r, tea.Batch(cmds...)

	case StatusMsg:
		r.status.status = ""
		r.status.filePath = msg.Text
		return r, nil

	case ErrorMsg:
		r.status.status = "error: " + msg.Err.Error()
		return r, nil

	case ExternalChangeMsg:
		var cmd tea.Cmd
		r.content, cmd = r.content.Update(msg)
		cmds = append(cmds, cmd)
		return r, tea.Batch(cmds...)

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
		// sync dirty after potential bool toggle
		if r.content.config != nil {
			r.status.dirty = r.content.config.Dirty()
		}
	}
	cmds = append(cmds, cmd)

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

	// sidebar: narrow icon rail, full height minus statusbar
	bodyH := r.height - 1
	if bodyH < 3 {
		bodyH = 3
	}

	r.sidebar.width = railWidth
	r.sidebar.height = bodyH

	// content: fill remaining width
	r.content.width = r.width - railWidth
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
		r.status.hints = []keyHint{
			{"↑↓", "navigate"},
			{"⏎", "open"},
			{"→", "content"},
			{"t", "theme"},
			{"q", "quit"},
		}
	case paneContent:
		hints := []keyHint{
			{"↑↓", "navigate"},
			{"⏎", "edit"},
			{"←", "back"},
			{"t", "theme"},
		}
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
