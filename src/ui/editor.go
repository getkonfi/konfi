package ui

import (
	"github.com/eminert/konfi/pkg"
	"github.com/eminert/konfi/theme"

	"charm.land/bubbles/v2/textinput"
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

// offsetEditor is optionally implemented by multi-line editors that track an
// internal cursor offset, so scroll positioning can follow the active line.
type offsetEditor interface {
	cursorOffset() int
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

// newFieldInput returns a textinput configured with the shared inline-editor
// prompt and focus styling. callers set their own value, validator, placeholder.
func newFieldInput(th *theme.Theme) textinput.Model {
	ti := textinput.New()
	ti.Prompt = "┊ "
	s := textinput.DefaultDarkStyles()
	s.Focused.Prompt = th.Muted
	s.Focused.Text = th.Text
	ti.SetStyles(s)
	return ti
}
