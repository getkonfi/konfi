package hyprland

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func testdataPath(name string) string {
	return filepath.Join("testdata", name)
}

func mustReadTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(testdataPath(name))
	if err != nil {
		t.Fatalf("failed to read testdata/%s: %v", name, err)
	}
	return data
}

// -- FindValue tests --

func TestFindValueFlat(t *testing.T) {
	p := newParser()
	data := mustReadTestdata(t, "config.txt")

	v, ok := p.FindValue(data, "monitor")
	if !ok {
		t.Fatal("expected to find 'monitor'")
	}
	if v != ", preferred, auto, 1" {
		t.Fatalf("got %q, want %q", v, ", preferred, auto, 1")
	}
}

func TestFindValueNested(t *testing.T) {
	p := newParser()
	data := mustReadTestdata(t, "config.txt")

	v, ok := p.FindValue(data, "decoration.rounding")
	if !ok {
		t.Fatal("expected to find 'decoration.rounding'")
	}
	if v != "10" {
		t.Fatalf("got %q, want %q", v, "10")
	}
}

func TestFindValueVariable(t *testing.T) {
	p := newParser()
	data := mustReadTestdata(t, "config.txt")

	v, ok := p.FindValue(data, "$mainMod")
	if !ok {
		t.Fatal("expected to find '$mainMod'")
	}
	if v != "SUPER" {
		t.Fatalf("got %q, want %q", v, "SUPER")
	}
}

func TestFindValueMissing(t *testing.T) {
	p := newParser()
	data := mustReadTestdata(t, "config.txt")

	_, ok := p.FindValue(data, "nonexistent")
	if ok {
		t.Fatal("expected not to find 'nonexistent'")
	}
}

func TestFindValueNestedDeep(t *testing.T) {
	p := newParser()
	data := mustReadTestdata(t, "config.txt")

	// general.border_size is inside general block
	v, ok := p.FindValue(data, "general.border_size")
	if !ok {
		t.Fatal("expected to find 'general.border_size'")
	}
	if v != "2" {
		t.Fatalf("got %q, want %q", v, "2")
	}
}

// -- FindLine tests --

func TestFindLineFlat(t *testing.T) {
	p := newParser()
	data := mustReadTestdata(t, "config.txt")

	tests := []struct {
		key    string
		want   int
		wantOK bool
	}{
		{"$mainMod", 2, true},
		{"monitor", 4, true},
		{"windowrule", 31, true},
		{"nonexistent", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, ok := p.FindLine(data, tt.key)
			if ok != tt.wantOK {
				t.Fatalf("FindLine(%q) ok = %v, want %v", tt.key, ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("FindLine(%q) = %d, want %d", tt.key, got, tt.want)
			}
		})
	}
}

func TestFindLineNested(t *testing.T) {
	p := newParser()
	data := mustReadTestdata(t, "config.txt")

	tests := []struct {
		key    string
		want   int
		wantOK bool
	}{
		{"general.border_size", 7, true},
		{"general.gaps_out", 9, true},
		{"decoration.rounding", 14, true},
		{"input.sensitivity", 22, true},
		{"animations.enabled", 27, true},
		{"general.missing", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, ok := p.FindLine(data, tt.key)
			if ok != tt.wantOK {
				t.Fatalf("FindLine(%q) ok = %v, want %v", tt.key, ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("FindLine(%q) = %d, want %d", tt.key, got, tt.want)
			}
		})
	}
}

func TestFindLineDepth2(t *testing.T) {
	p := newParser()
	data := mustReadTestdata(t, "config.txt")

	tests := []struct {
		key    string
		want   int
		wantOK bool
	}{
		{"decoration.blur.enabled", 16, true},
		{"decoration.blur.size", 17, true},
		{"decoration.blur.missing", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, ok := p.FindLine(data, tt.key)
			if ok != tt.wantOK {
				t.Fatalf("FindLine(%q) ok = %v, want %v", tt.key, ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("FindLine(%q) = %d, want %d", tt.key, got, tt.want)
			}
		})
	}
}

// -- SetValue tests --

func TestSetValueFlat(t *testing.T) {
	p := newParser()
	data := mustReadTestdata(t, "config.txt")
	want := mustReadTestdata(t, "set_flat.txt")

	got, err := p.SetValue(data, "monitor", "DP-1, 1920x1080, 0x0, 1")
	if err != nil {
		t.Fatalf("SetValue error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("mismatch.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestSetValueNested(t *testing.T) {
	p := newParser()
	data := mustReadTestdata(t, "config.txt")
	want := mustReadTestdata(t, "set_nested.txt")

	got, err := p.SetValue(data, "decoration.rounding", "15")
	if err != nil {
		t.Fatalf("SetValue error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("mismatch.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestSetValueNewNested(t *testing.T) {
	p := newParser()
	data := mustReadTestdata(t, "config.txt")
	want := mustReadTestdata(t, "set_new_nested.txt")

	got, err := p.SetValue(data, "general.layout", "dwindle")
	if err != nil {
		t.Fatalf("SetValue error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("mismatch.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestSetValueCreateBlock(t *testing.T) {
	p := newParser()
	data := mustReadTestdata(t, "config.txt")

	got, err := p.SetValue(data, "misc.disable_hyprland_logo", "true")
	if err != nil {
		t.Fatalf("SetValue error: %v", err)
	}

	// verify the value was set
	v, ok := p.FindValue(got, "misc.disable_hyprland_logo")
	if !ok {
		t.Fatal("expected to find newly created 'misc.disable_hyprland_logo'")
	}
	if v != "true" {
		t.Fatalf("got %q, want %q", v, "true")
	}
}

func TestSetValueVariable(t *testing.T) {
	p := newParser()
	data := mustReadTestdata(t, "config.txt")

	got, err := p.SetValue(data, "$mainMod", "ALT")
	if err != nil {
		t.Fatalf("SetValue error: %v", err)
	}

	v, ok := p.FindValue(got, "$mainMod")
	if !ok {
		t.Fatal("expected to find '$mainMod' after set")
	}
	if v != "ALT" {
		t.Fatalf("got %q, want %q", v, "ALT")
	}
}

// -- DeleteKey tests --

func TestDeleteFlat(t *testing.T) {
	p := newParser()
	data := mustReadTestdata(t, "config.txt")
	want := mustReadTestdata(t, "delete_flat.txt")

	got, err := p.DeleteKey(data, "monitor")
	if err != nil {
		t.Fatalf("DeleteKey error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("mismatch.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestDeleteNested(t *testing.T) {
	p := newParser()
	data := mustReadTestdata(t, "config.txt")
	want := mustReadTestdata(t, "delete_nested.txt")

	got, err := p.DeleteKey(data, "decoration.rounding")
	if err != nil {
		t.Fatalf("DeleteKey error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("mismatch.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestDeleteMissing(t *testing.T) {
	p := newParser()
	data := mustReadTestdata(t, "config.txt")

	got, err := p.DeleteKey(data, "nonexistent")
	if err != nil {
		t.Fatalf("DeleteKey error: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("expected data to be unchanged when deleting missing key")
	}
}

// -- depth-2 nested tests --

func TestFindValueDepth2(t *testing.T) {
	p := newParser()
	data := mustReadTestdata(t, "config.txt")

	v, ok := p.FindValue(data, "decoration.blur.enabled")
	if !ok {
		t.Fatal("expected to find 'decoration.blur.enabled'")
	}
	if v != "true" {
		t.Fatalf("got %q, want %q", v, "true")
	}

	v, ok = p.FindValue(data, "decoration.blur.size")
	if !ok {
		t.Fatal("expected to find 'decoration.blur.size'")
	}
	if v != "3" {
		t.Fatalf("got %q, want %q", v, "3")
	}
}

func TestSetValueDepth2(t *testing.T) {
	p := newParser()
	data := mustReadTestdata(t, "config.txt")

	got, err := p.SetValue(data, "decoration.blur.size", "5")
	if err != nil {
		t.Fatalf("SetValue error: %v", err)
	}
	v, ok := p.FindValue(got, "decoration.blur.size")
	if !ok {
		t.Fatal("expected to find 'decoration.blur.size' after set")
	}
	if v != "5" {
		t.Fatalf("got %q, want %q", v, "5")
	}

	// original value at depth-1 should be unaffected
	v, ok = p.FindValue(got, "decoration.rounding")
	if !ok {
		t.Fatal("expected 'decoration.rounding' to still exist")
	}
	if v != "10" {
		t.Fatalf("got %q, want %q", v, "10")
	}
}

func TestDeleteValueDepth2(t *testing.T) {
	p := newParser()
	data := mustReadTestdata(t, "config.txt")

	got, err := p.DeleteKey(data, "decoration.blur.enabled")
	if err != nil {
		t.Fatalf("DeleteKey error: %v", err)
	}
	_, ok := p.FindValue(got, "decoration.blur.enabled")
	if ok {
		t.Fatal("expected 'decoration.blur.enabled' to be deleted")
	}

	// sibling should still exist
	v, ok := p.FindValue(got, "decoration.blur.size")
	if !ok {
		t.Fatal("expected 'decoration.blur.size' to still exist")
	}
	if v != "3" {
		t.Fatalf("got %q, want %q", v, "3")
	}
}

// -- round-trip --

func TestRoundTrip(t *testing.T) {
	p := newParser()
	data := mustReadTestdata(t, "config.txt")

	// set a value then find it
	modified, err := p.SetValue(data, "decoration.rounding", "20")
	if err != nil {
		t.Fatalf("SetValue error: %v", err)
	}
	v, ok := p.FindValue(modified, "decoration.rounding")
	if !ok {
		t.Fatal("expected to find 'decoration.rounding' after set")
	}
	if v != "20" {
		t.Fatalf("round-trip: got %q, want %q", v, "20")
	}
}

func TestRoundTripGolden(t *testing.T) {
	p := newParser()

	src := []byte(`# hyprland config

$mainMod = SUPER

monitor = , preferred, auto, 1

general {
    border_size = 2
    gaps_in = 5
    gaps_out = 10
}

decoration {
    rounding = 10
    blur {
        enabled = true
        size = 3
    }
}

input {
    sensitivity = 0.0
    kb_layout = us
}
`)

	// step 1: modify a flat variable
	out, err := p.SetValue(src, "$mainMod", "ALT")
	if err != nil {
		t.Fatal(err)
	}
	v, ok := p.FindValue(out, "$mainMod")
	if !ok || v != "ALT" {
		t.Fatalf("SetValue $mainMod: got %q ok=%v", v, ok)
	}

	// step 2: modify a nested key
	out, err = p.SetValue(out, "general.border_size", "3")
	if err != nil {
		t.Fatal(err)
	}
	v, ok = p.FindValue(out, "general.border_size")
	if !ok || v != "3" {
		t.Fatalf("SetValue general.border_size: got %q ok=%v", v, ok)
	}

	// step 3: modify a depth-2 key
	out, err = p.SetValue(out, "decoration.blur.size", "5")
	if err != nil {
		t.Fatal(err)
	}
	v, ok = p.FindValue(out, "decoration.blur.size")
	if !ok || v != "5" {
		t.Fatalf("SetValue decoration.blur.size: got %q ok=%v", v, ok)
	}

	// step 4: add a new key in existing block
	out, err = p.SetValue(out, "input.follow_mouse", "1")
	if err != nil {
		t.Fatal(err)
	}
	v, ok = p.FindValue(out, "input.follow_mouse")
	if !ok || v != "1" {
		t.Fatalf("SetValue input.follow_mouse: got %q ok=%v", v, ok)
	}

	// step 5: verify comments survived
	if !bytes.Contains(out, []byte("# hyprland config")) {
		t.Error("comment line lost during round-trip")
	}

	// step 6: verify untouched keys preserved
	for _, key := range []string{"monitor", "general.gaps_in", "general.gaps_out", "decoration.rounding", "decoration.blur.enabled", "input.sensitivity", "input.kb_layout"} {
		if _, ok := p.FindValue(out, key); !ok {
			t.Errorf("key %q lost during round-trip", key)
		}
	}

	// step 7: verify block structure preserved
	for _, block := range []string{"general {", "decoration {", "input {", "blur {"} {
		if !bytes.Contains(out, []byte(block)) {
			t.Errorf("block %q structure lost", block)
		}
	}

	// step 8: ListKeys covers everything
	keys := p.ListKeys(out)
	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}
	if !keySet["input.follow_mouse"] {
		t.Error("ListKeys missing newly added input.follow_mouse")
	}
}

func FuzzParser(f *testing.F) {
	f.Add([]byte("$mainMod = SUPER\nmonitor = , preferred, auto, 1\n"), "monitor")
	f.Add([]byte("general {\n    border_size = 2\n}\n"), "general.border_size")
	f.Add([]byte("decoration {\n    blur {\n        enabled = true\n    }\n}\n"), "decoration.blur.enabled")
	f.Add([]byte("# comment\n\ninput {\n    sensitivity = 0.0\n}\n"), "input.sensitivity")
	f.Add([]byte(""), "missing")
	f.Add([]byte("key = value\n"), "key")
	f.Add([]byte("block {\n}\n"), "block.inner")

	p := newParser()
	f.Fuzz(func(t *testing.T, data []byte, key string) {
		p.FindValue(data, key)
		p.FindLine(data, key)
		p.ListKeys(data)
		if out, err := p.SetValue(data, key, "fuzzval"); err == nil {
			p.FindValue(out, key)
			p.ListKeys(out)
		}
		p.DeleteKey(data, key)
	})
}
