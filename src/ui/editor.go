package ui

import (
	"strconv"

	"github.com/emin/konfigurator/pkg"
	"github.com/emin/konfigurator/theme"

	tea "charm.land/bubbletea/v2"
)

// FieldEditor is the interface for type-aware inline config editors.
type FieldEditor interface {
	Init(field pkg.Field, currentValue string, th *theme.Theme) tea.Cmd
	Update(msg tea.Msg) (cmd tea.Cmd, done bool, canceled bool)
	View(width int) string
	Value() string
	Height() int
}

// InlineEditor is optionally implemented by editors that render on the
// same row as the field label instead of on a separate line below.
type InlineEditor interface {
	FieldEditor
	InlineView(width int) string
}

// editorForField returns the appropriate editor for a field type.
// widget hint takes priority over type dispatch.
func editorForField(f pkg.Field) FieldEditor {
	switch f.Widget {
	case "font":
		return &fontEditor{}
	case "slider":
		return &sliderEditor{}
	case "path":
		return &pathEditor{}
	case "stylestring":
		if len(f.Options) > 0 && len(f.AltOptions) > 0 {
			return &stylestringEditor{}
		}
		return &stringEditor{}
	case "hook":
		return &hookEditor{}
	case "togglemap":
		return &toggleMapEditor{}
	case "structlist":
		return &structListEditor{}
	case "patternlist":
		return &listEditor{}
	}
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
	case "list":
		return &listEditor{}
	case "multi":
		return &multiEditor{}
	default:
		return &stringEditor{}
	}
}

// formatValue serializes a value for writing back to a config file.
// TOML string/color/enum values need quoting; ghostty/hyprland values are raw.
func formatValue(value, fieldType, configFormat string) string {
	if configFormat == "toml" {
		switch fieldType {
		case "string", "color", "enum", "multi":
			return strconv.Quote(value)
		}
	}
	return value
}
