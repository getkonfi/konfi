package ui

import (
	"strings"
	"testing"

	"github.com/eminert/konfi/konfables"
	"github.com/eminert/konfi/pkg"
	"github.com/eminert/konfi/pkg/parser"
)

func TestStripKeyPrefix(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
		key   string
		want  string
	}{
		{"copied form", "format = $all", "format", "$all"},
		{"no spaces", "format=$all", "format", "$all"},
		{"plain value untouched", "$all", "format", "$all"},
		{"non-matching key untouched", "other = x", "format", "other = x"},
		{"key as substring not stripped", "formatted", "format", "formatted"},
		{"value with equals preserved", "ctrl+a = goto_tab:1", "keybind", "ctrl+a = goto_tab:1"},
		{"empty key untouched", "a = b", "", "a = b"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := stripKeyPrefix(tc.value, tc.key); got != tc.want {
				t.Fatalf("stripKeyPrefix(%q, %q) = %q, want %q", tc.value, tc.key, got, tc.want)
			}
		})
	}
}

func TestSingleLine(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"$os$username", "$os$username"}, // clean value untouched
		{"a\nb", "a\\nb"},                // real newline escaped
		{"a\tb", "a\\tb"},                // tab escaped
		{"a\r\nb", "a\\nb"},              // crlf collapses to one escape
	} {
		if got := singleLine(tc.in); got != tc.want {
			t.Fatalf("singleLine(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestLowContrast(t *testing.T) {
	const bg = "#1e1e2e" // catppuccin base
	if !lowContrast(bg, bg) {
		t.Fatal("identical color/background should be low contrast")
	}
	if !lowContrast("#222232", bg) {
		t.Fatal("near-background color should be low contrast")
	}
	if lowContrast("#ffffff", bg) {
		t.Fatal("white on dark base should not be low contrast")
	}
}

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

// colorValue adds a background backdrop only when the tint is too close to bg.
func TestColorValueContrastBackdrop(t *testing.T) {
	const bg = "#1e1e2e"

	nearBg := colorValue("#1f1f30", bg)
	if !strings.Contains(nearBg, "48;2") {
		t.Fatalf("near-background color should get a contrast backdrop, got %q", nearBg)
	}

	readable := colorValue("#ffffff", bg)
	if strings.Contains(readable, "48;2") {
		t.Fatalf("readable color should not get a backdrop, got %q", readable)
	}

	// no ## marker in either case
	if strings.Contains(stripANSI(nearBg), "##") || strings.Contains(stripANSI(readable), "##") {
		t.Fatal("color value should not contain a ## marker")
	}
}
