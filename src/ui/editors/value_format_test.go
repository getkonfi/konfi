package editors

import (
	"strings"
	"testing"

	"github.com/getkonfi/konfi/konfables"
	"github.com/getkonfi/konfi/pkg"
	"github.com/getkonfi/konfi/pkg/parser"
)

// a ghostty-style repeated key (keybind) must read and write every occurrence,
// not just the first — mirrors the openEditor/commitEdit dispatch for
// type:list + structlist + MultiValueParser fields.
func TestKeybindRepeatedKeyRoundTrip(t *testing.T) {
	p := &parser.FlatParser{Split: parser.SplitEquals, Format: parser.FormatEquals}
	cfg := []byte("keybind = ctrl+a=new_tab\n" +
		"keybind = ctrl+b=close_surface\n" +
		"keybind = ctrl+c=copy_to_clipboard\n" +
		"font-size = 14\n")

	// read path
	vals, ok := p.FindValues(cfg, "keybind")
	if !ok || len(vals) != 3 {
		t.Fatalf("FindValues = %v (ok=%v), want 3", vals, ok)
	}

	// editor round-trip through structListEditor
	field := pkg.Field{
		Key: "keybind", Type: "list", Widget: "structlist", Separator: "=",
		ItemSchema: []pkg.FieldPart{{Name: "keys"}, {Name: "action"}},
	}
	e := &structListEditor{}
	e.Init(field, strings.Join(vals, "\n"), testTheme())

	// write path
	newData, err := p.SetValues(cfg, "keybind", konfables.SplitListValue(e.Value()))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := p.FindValues(newData, "keybind")
	if len(got) != 3 {
		t.Fatalf("after round-trip got %d keybinds, want 3:\n%s", len(got), newData)
	}
	for i, want := range vals {
		if got[i] != want {
			t.Fatalf("keybind[%d] = %q, want %q", i, got[i], want)
		}
	}
	if v, _ := p.FindValue(newData, "font-size"); v != "14" {
		t.Fatalf("unrelated key not preserved: font-size = %q", v)
	}
}

func TestColorEditorSelectionDoesNotUseSquareBrackets(t *testing.T) {
	var e colorEditor
	_ = e.Init(pkg.Field{Palette: []string{"#112233"}}, "#112233", testTheme())

	got := stripANSI(e.View(80))
	firstLine := strings.Split(got, "\n")[0]
	if strings.ContainsAny(firstLine, "[]") {
		t.Fatalf("selected color cell should not use square brackets: %q", firstLine)
	}
	if !strings.Contains(firstLine, "#112233") {
		t.Fatalf("selected color cell should show the hex code: %q", firstLine)
	}
}
