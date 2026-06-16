package editors

import (
	"github.com/getkonfi/konfi/pkg"
	"github.com/getkonfi/konfi/theme"

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

// OffsetEditor is optionally implemented by multi-line editors that track an
// internal cursor offset, so scroll positioning can follow the active line.
type OffsetEditor interface {
	CursorOffset() int
}

// Previewer is implemented by editors that expose a live preview of the
// currently composed value (color swatch, styled string).
type Previewer interface {
	FieldEditor
	PreviewValue() string
}

// InteractionKind classifies an editor's key UX so callers can render the
// matching hint set without knowing concrete editor types.
type InteractionKind int

const (
	InteractionSingle    InteractionKind = iota // single value: confirm/cancel/switch
	InteractionList                             // add/delete/edit list rows
	InteractionToggleMap                        // toggle/add/delete map entries
	InteractionEnum                             // select from options
	InteractionMulti                            // toggle multiple options, then accept
)

// Interactor is optionally implemented by editors with a non-default key UX.
// editors that don't implement it are treated as InteractionSingle.
type Interactor interface {
	Interaction() InteractionKind
}

// MultiValueEditor is optionally implemented by editors that consume every
// occurrence of a repeated key, joined by newlines (list, structlist).
type MultiValueEditor interface {
	AcceptsMultiValue() bool
}

// ForField returns the appropriate editor for a field type.
// widget hint takes priority over type dispatch.
func ForField(f pkg.Field) FieldEditor {
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
	case "blocklist":
		return &blockEditor{}
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
