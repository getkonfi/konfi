package ui

import (
	"github.com/emin/konfigurator/pkg"
	"github.com/emin/konfigurator/theme"
)

// shared tea.Msg types for the UI layer

// ThemeChangedMsg is sent when the active theme palette changes.
type ThemeChangedMsg struct {
	Theme *theme.Theme
}

// AppSelectedMsg is sent when a konfable is selected in the sidebar.
// Confirmed is true when the user pressed enter/space (explicit open),
// false when browsing with j/k/arrows (preview only).
type AppSelectedMsg struct {
	Index     int
	Name      string
	Confirmed bool
}

// ErrorMsg wraps an error for status bar display.
type ErrorMsg struct {
	Err error
}

// StatusMsg sets a transient status bar message.
type StatusMsg struct {
	Text string
}

// ExternalChangeMsg is sent when an external process modifies the config file.
type ExternalChangeMsg struct {
	Path string
}

// insightTickMsg fires every ~5s to cycle the insight line in the header.
type insightTickMsg struct{ gen int }

// splitFlapTickMsg fires every ~60ms to advance the split-flap animation.
type splitFlapTickMsg struct{ gen int }

// logoAnimTickMsg fires every ~60ms to advance the logo animation.
type logoAnimTickMsg struct{ gen int }

// DocOpenedMsg is sent after attempting to open a doc URL in the browser.
type DocOpenedMsg struct{ URL string }

// KonfSettingChangedMsg is sent when a konfigurator setting is edited.
type KonfSettingChangedMsg struct {
	Key   string
	Value string
}

// fileStateClearMsg resets the transient file state after a delay.
type fileStateClearMsg struct{}

// FontsLoadedMsg delivers system font families from fc-list.
type FontsLoadedMsg struct {
	Fonts []string // sorted, deduplicated family names
	Err   error    // nil on success, non-nil triggers freetext fallback
}

// SelectAppMsg requests navigating to a specific app by index.
type SelectAppMsg struct {
	Index int
}

// SaveMsg requests saving the current config.
type SaveMsg struct{}

// UndoMsg requests undoing the last edit.
type UndoMsg struct{}

// RedoMsg requests redoing the last undone edit.
type RedoMsg struct{}

// EditorExitMsg is sent when the external $EDITOR process exits.
type EditorExitMsg struct{ Err error }

// confirmQuitClearMsg auto-clears the quit confirmation after a timeout.
type confirmQuitClearMsg struct{}

// confirmSwitchClearMsg auto-clears the app-switch confirmation after a timeout.
type confirmSwitchClearMsg struct{}

// ToggleFilterMsg toggles the "configured only" filter from the palette.
type ToggleFilterMsg struct{}

// CycleThemeMsg cycles to the next theme from the palette.
type CycleThemeMsg struct{}

// ToggleNewMsg toggles the "what's new" filter from the palette.
type ToggleNewMsg struct{}

// OpenEditorMsg opens the config in $EDITOR from the palette.
type OpenEditorMsg struct{}

// ToggleHelpMsg toggles the help overlay from the palette.
type ToggleHelpMsg struct{}

// JumpToFieldMsg requests navigating to a specific field by index.
type JumpToFieldMsg struct{ FieldIdx int }

// saveResultMsg is sent when an async config save completes.
type saveResultMsg struct{ err error }

// reloadResultMsg is sent when an async config reload completes.
type reloadResultMsg struct {
	source string // "external", "editor", "save-reload"
}

// appLoadedMsg is sent when async config loading completes.
type appLoadedMsg struct {
	config  *pkg.ConfigFile
	path    string
	isNew   bool
	err     error
	appName string // guards against stale loads after app switch
}
