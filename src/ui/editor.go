package ui

import (
	"github.com/emin/konfigurator/pkg"
	"github.com/emin/konfigurator/theme"

	tea "github.com/charmbracelet/bubbletea"
)

// FieldEditor is the interface for type-aware inline config editors.
type FieldEditor interface {
	Init(field pkg.Field, currentValue string, th *theme.Theme) tea.Cmd
	Update(msg tea.Msg) (cmd tea.Cmd, done bool, canceled bool)
	View(width int) string
	Value() string
	Height() int
}

// editorForField returns the appropriate editor for a field type.
// falls back to string editor for unknown types.
func editorForField(f pkg.Field) FieldEditor {
	switch f.Type {
	case "number":
		return &numberEditor{}
	case "enum":
		if len(f.Options) == 0 {
			return &stringEditor{}
		}
		return &enumEditor{}
	case "color":
		return &colorEditor{}
	default:
		return &stringEditor{}
	}
}

// formatValue serializes a value for writing back to a config file.
// TOML string/color/enum values need quoting; ghostty/hyprland values are raw.
func formatValue(value, fieldType, configFormat string) string {
	if configFormat == "toml" {
		switch fieldType {
		case "string", "color", "enum":
			return `"` + value + `"`
		}
	}
	return value
}
