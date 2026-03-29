package kitty

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func loadTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read testdata/%s: %v", name, err)
	}
	return data
}

func TestFindValue(t *testing.T) {
	p := newParser()
	data := loadTestdata(t, "config.conf")

	tests := []struct {
		key    string
		want   string
		wantOK bool
	}{
		{"font_family", "JetBrains Mono", true},
		{"font_size", "14.0", true},
		{"bold_font", "auto", true},
		{"background", "#282828", true},
		{"hide_window_decorations", "yes", true},
		{"nonexistent", "", false},
		// "kitty" appears in a comment, not as a key
		{"kitty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, ok := p.FindValue(data, tt.key)
			if ok != tt.wantOK {
				t.Fatalf("FindValue(%q) ok = %v, want %v", tt.key, ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("FindValue(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestFindLine(t *testing.T) {
	p := newParser()
	data := loadTestdata(t, "config.conf")

	tests := []struct {
		key    string
		want   int
		wantOK bool
	}{
		{"font_family", 1, true},
		{"font_size", 2, true},
		{"bold_font", 3, true},
		{"background", 6, true},
		{"shell_integration", 14, true},
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

func TestSetValue(t *testing.T) {
	p := newParser()
	data := loadTestdata(t, "config.conf")

	t.Run("replace existing", func(t *testing.T) {
		got, err := p.SetValue(data, "font_size", "16.0")
		if err != nil {
			t.Fatal(err)
		}
		want := loadTestdata(t, "set_existing.conf")
		if !bytes.Equal(got, want) {
			t.Errorf("mismatch.\ngot:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("append new", func(t *testing.T) {
		got, err := p.SetValue(data, "cursor_shape", "block")
		if err != nil {
			t.Fatal(err)
		}
		want := loadTestdata(t, "set_new.conf")
		if !bytes.Equal(got, want) {
			t.Errorf("mismatch.\ngot:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("preserves comments", func(t *testing.T) {
		got, err := p.SetValue(data, "font_size", "14.0")
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, data) {
			t.Error("setting same value should preserve file exactly")
		}
	})
}

func TestDeleteKey(t *testing.T) {
	p := newParser()
	data := loadTestdata(t, "config.conf")

	t.Run("delete existing", func(t *testing.T) {
		got, err := p.DeleteKey(data, "hide_window_decorations")
		if err != nil {
			t.Fatal(err)
		}
		want := loadTestdata(t, "delete.conf")
		if !bytes.Equal(got, want) {
			t.Errorf("mismatch.\ngot:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("delete missing is noop", func(t *testing.T) {
		got, err := p.DeleteKey(data, "nonexistent")
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, data) {
			t.Error("deleting missing key should return data unchanged")
		}
	})
}

func TestListKeys(t *testing.T) {
	p := newParser()
	data := loadTestdata(t, "config.conf")
	keys := p.ListKeys(data)

	if len(keys) == 0 {
		t.Fatal("expected at least one key")
	}

	found := make(map[string]bool)
	for _, k := range keys {
		found[k] = true
	}
	for _, want := range []string{
		"font_family", "font_size", "bold_font",
		"background", "foreground",
		"window_padding_width", "hide_window_decorations",
		"shell_integration",
	} {
		if !found[want] {
			t.Errorf("missing key %q in ListKeys output", want)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	p := newParser()
	data := loadTestdata(t, "config.conf")

	// add new key
	updated, err := p.SetValue(data, "cursor_shape", "beam")
	if err != nil {
		t.Fatal(err)
	}
	val, ok := p.FindValue(updated, "cursor_shape")
	if !ok {
		t.Fatal("expected to find cursor_shape after set")
	}
	if val != "beam" {
		t.Errorf("got %q, want %q", val, "beam")
	}

	// original keys still intact
	val, ok = p.FindValue(updated, "font_size")
	if !ok {
		t.Fatal("expected font_size to survive round-trip")
	}
	if val != "14.0" {
		t.Errorf("got %q, want %q", val, "14.0")
	}
}

func TestRoundTripGolden(t *testing.T) {
	p := newParser()

	src := []byte(`# kitty terminal config
font_family JetBrains Mono
font_size 14.0

# colors
background #282828
foreground #ebdbb2

# window
window_padding_width 8
hide_window_decorations yes
`)

	// step 1: modify existing
	out, err := p.SetValue(src, "font_size", "16.0")
	if err != nil {
		t.Fatal(err)
	}
	v, ok := p.FindValue(out, "font_size")
	if !ok || v != "16.0" {
		t.Fatalf("SetValue font_size: got %q ok=%v", v, ok)
	}

	// step 2: add new key
	out, err = p.SetValue(out, "cursor_shape", "beam")
	if err != nil {
		t.Fatal(err)
	}
	v, ok = p.FindValue(out, "cursor_shape")
	if !ok || v != "beam" {
		t.Fatalf("SetValue cursor_shape: got %q ok=%v", v, ok)
	}

	// step 3: comments survived
	if !bytes.Contains(out, []byte("# kitty terminal config")) {
		t.Error("comment line lost during round-trip")
	}
	if !bytes.Contains(out, []byte("# colors")) {
		t.Error("second comment line lost during round-trip")
	}

	// step 4: all original keys preserved
	for _, key := range []string{"font_family", "background", "foreground", "window_padding_width", "hide_window_decorations"} {
		if _, ok := p.FindValue(out, key); !ok {
			t.Errorf("key %q lost during round-trip", key)
		}
	}

	// step 5: empty lines preserved
	if bytes.Count(out, []byte("\n\n")) < 2 {
		t.Error("empty lines collapsed during round-trip")
	}

	// step 6: ListKeys includes both old and new
	keys := p.ListKeys(out)
	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}
	if !keySet["cursor_shape"] {
		t.Error("ListKeys missing newly added cursor_shape")
	}
	if !keySet["font_size"] {
		t.Error("ListKeys missing modified font_size")
	}
}

func FuzzParser(f *testing.F) {
	f.Add([]byte("font_size 14.0\n"), "font_size")
	f.Add([]byte("font_family JetBrains Mono\nfont_size 14.0\n"), "font_family")
	f.Add([]byte("# comment\nbackground #282828\n\nforeground #ebdbb2\n"), "background")
	f.Add([]byte(""), "missing")
	f.Add([]byte("no-space-here\n"), "no-space-here")
	f.Add([]byte("key = val\n"), "key")

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
