package editors

import (
	"testing"

	"github.com/eminert/konfi/pkg"
	"github.com/eminert/konfi/theme"
)

// TestStructListEditorInitWithJSONArrayDefault verifies that initializing a
// structListEditor with "[]" (the default for missing JSON widget values in
// content.go) doesn't create a phantom item. "[]" is a JSON array literal
// but structlist uses separator-delimited text, not JSON.
func TestStructListEditorInitWithJSONArrayDefault(t *testing.T) {
	th := theme.NewTheme(theme.PaletteByName("catppuccin-mocha"))
	field := pkg.Field{
		Key:    "keybind",
		Label:  "Key Bindings",
		Type:   "list",
		Widget: "structlist",
		ItemSchema: []pkg.FieldPart{
			{Name: "keys", Type: "string", Required: true},
			{Name: "action", Type: "string", Required: true},
		},
		Separator: "=",
	}

	e := &structListEditor{}
	e.Init(field, "[]", th)

	// "[]" is not a valid structlist item — editor should have 0 items
	if len(e.items) != 0 {
		t.Errorf("Init with %q created %d phantom items, want 0", "[]", len(e.items))
		for i, item := range e.items {
			t.Logf("  item[%d] = %v", i, item)
		}
	}

	// value should be empty string (no items)
	if v := e.Value(); v != "" {
		t.Errorf("Value() = %q, want empty string", v)
	}
}
