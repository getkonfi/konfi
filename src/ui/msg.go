package ui

import "github.com/emin/konfigurator/theme"

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
