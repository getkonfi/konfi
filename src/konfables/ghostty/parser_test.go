package ghostty

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
	p := &parser{}
	data := loadTestdata(t, "config.txt")

	t.Run("existing key", func(t *testing.T) {
		val, ok := p.FindValue(data, "font-size")
		if !ok {
			t.Fatal("expected to find font-size")
		}
		if val != "14" {
			t.Errorf("got %q, want %q", val, "14")
		}
	})

	t.Run("missing key", func(t *testing.T) {
		_, ok := p.FindValue(data, "cursor-style")
		if ok {
			t.Fatal("expected not to find cursor-style")
		}
	})

	t.Run("skips comments", func(t *testing.T) {
		// "ghostty" appears in the comment but not as a key
		_, ok := p.FindValue(data, "ghostty")
		if ok {
			t.Fatal("should not match inside comments")
		}
	})

	t.Run("value with spaces", func(t *testing.T) {
		val, ok := p.FindValue(data, "font-family")
		if !ok {
			t.Fatal("expected to find font-family")
		}
		if val != "JetBrains Mono" {
			t.Errorf("got %q, want %q", val, "JetBrains Mono")
		}
	})
}

func TestFindLine(t *testing.T) {
	p := &parser{}
	data := loadTestdata(t, "config.txt")

	tests := []struct {
		key    string
		want   int
		wantOK bool
	}{
		{"font-family", 1, true},
		{"font-size", 2, true},
		{"window-decoration", 7, true},
		{"shell-integration", 12, true},
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
	p := &parser{}
	data := loadTestdata(t, "config.txt")

	t.Run("replace existing", func(t *testing.T) {
		got, err := p.SetValue(data, "font-size", "16")
		if err != nil {
			t.Fatal(err)
		}
		want := loadTestdata(t, "set_existing.txt")
		if !bytes.Equal(got, want) {
			t.Errorf("mismatch.\ngot:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("append new", func(t *testing.T) {
		got, err := p.SetValue(data, "cursor-style", "block")
		if err != nil {
			t.Fatal(err)
		}
		want := loadTestdata(t, "set_new.txt")
		if !bytes.Equal(got, want) {
			t.Errorf("mismatch.\ngot:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("preserves comments", func(t *testing.T) {
		got, err := p.SetValue(data, "font-size", "14")
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, data) {
			t.Error("setting same value should preserve file exactly")
		}
	})
}

func TestDeleteKey(t *testing.T) {
	p := &parser{}
	data := loadTestdata(t, "config.txt")

	t.Run("delete existing", func(t *testing.T) {
		got, err := p.DeleteKey(data, "window-decoration")
		if err != nil {
			t.Fatal(err)
		}
		want := loadTestdata(t, "delete.txt")
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

func TestFindValues(t *testing.T) {
	p := &parser{}

	data := []byte(`font-family = JetBrains Mono
keybind = ctrl+c=copy
keybind = ctrl+v=paste
keybind = ctrl+shift+v=paste_from_clipboard
font-size = 14
`)

	t.Run("multi-value key", func(t *testing.T) {
		vals, ok := p.FindValues(data, "keybind")
		if !ok {
			t.Fatal("expected to find keybind values")
		}
		if len(vals) != 3 {
			t.Fatalf("got %d values, want 3", len(vals))
		}
		if vals[0] != "ctrl+c=copy" {
			t.Errorf("vals[0] = %q, want %q", vals[0], "ctrl+c=copy")
		}
	})

	t.Run("single-value key", func(t *testing.T) {
		vals, ok := p.FindValues(data, "font-family")
		if !ok {
			t.Fatal("expected to find font-family")
		}
		if len(vals) != 1 {
			t.Fatalf("got %d values, want 1", len(vals))
		}
	})

	t.Run("missing key", func(t *testing.T) {
		_, ok := p.FindValues(data, "nonexistent")
		if ok {
			t.Fatal("expected not to find nonexistent key")
		}
	})
}

func TestSetValues(t *testing.T) {
	p := &parser{}

	data := []byte(`font-family = JetBrains Mono
keybind = ctrl+c=copy
keybind = ctrl+v=paste
font-size = 14
`)

	t.Run("replace multi-value", func(t *testing.T) {
		newVals := []string{"ctrl+a=select_all", "ctrl+c=copy"}
		got, err := p.SetValues(data, "keybind", newVals)
		if err != nil {
			t.Fatal(err)
		}

		// verify round-trip
		vals, ok := p.FindValues(got, "keybind")
		if !ok {
			t.Fatal("expected to find keybind after set")
		}
		if len(vals) != 2 {
			t.Fatalf("got %d values, want 2", len(vals))
		}
		if vals[0] != "ctrl+a=select_all" {
			t.Errorf("vals[0] = %q, want %q", vals[0], "ctrl+a=select_all")
		}

		// other keys preserved
		v, ok := p.FindValue(got, "font-family")
		if !ok || v != "JetBrains Mono" {
			t.Error("other keys should be preserved")
		}
	})

	t.Run("set empty removes all", func(t *testing.T) {
		got, err := p.SetValues(data, "keybind", nil)
		if err != nil {
			t.Fatal(err)
		}
		_, ok := p.FindValues(got, "keybind")
		if ok {
			t.Error("expected no keybind values after setting empty")
		}
	})
}

func TestListKeys(t *testing.T) {
	p := &parser{}
	data := loadTestdata(t, "config.txt")
	keys := p.ListKeys(data)

	if len(keys) == 0 {
		t.Fatal("expected at least one key")
	}

	// verify known keys are present
	found := make(map[string]bool)
	for _, k := range keys {
		found[k] = true
	}
	for _, want := range []string{"font-family", "font-size", "window-decoration", "shell-integration"} {
		if !found[want] {
			t.Errorf("missing key %q in ListKeys output", want)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	p := &parser{}
	data := loadTestdata(t, "config.txt")

	updated, err := p.SetValue(data, "cursor-style", "beam")
	if err != nil {
		t.Fatal(err)
	}

	val, ok := p.FindValue(updated, "cursor-style")
	if !ok {
		t.Fatal("expected to find cursor-style after set")
	}
	if val != "beam" {
		t.Errorf("got %q, want %q", val, "beam")
	}

	// original keys still intact
	val, ok = p.FindValue(updated, "font-size")
	if !ok {
		t.Fatal("expected font-size to survive round-trip")
	}
	if val != "14" {
		t.Errorf("got %q, want %q", val, "14")
	}
}
