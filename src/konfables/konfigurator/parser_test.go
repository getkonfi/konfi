package konfigurator

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
		val, ok := p.FindValue(data, "theme")
		if !ok {
			t.Fatal("expected to find theme")
		}
		if val != "catppuccin" {
			t.Errorf("got %q, want %q", val, "catppuccin")
		}
	})

	t.Run("missing key", func(t *testing.T) {
		_, ok := p.FindValue(data, "nonexistent")
		if ok {
			t.Fatal("expected not to find nonexistent")
		}
	})

	t.Run("skips comments", func(t *testing.T) {
		_, ok := p.FindValue(data, "konfigurator")
		if ok {
			t.Fatal("should not match inside comments")
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
		{"theme", 1, true},
		{"log_level", 2, true},
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
		got, err := p.SetValue(data, "theme", "tokyonight")
		if err != nil {
			t.Fatal(err)
		}
		want := loadTestdata(t, "set_existing.txt")
		if !bytes.Equal(got, want) {
			t.Errorf("mismatch.\ngot:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("append new", func(t *testing.T) {
		got, err := p.SetValue(data, "some_key", "value")
		if err != nil {
			t.Fatal(err)
		}
		want := loadTestdata(t, "set_new.txt")
		if !bytes.Equal(got, want) {
			t.Errorf("mismatch.\ngot:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("preserves comments", func(t *testing.T) {
		got, err := p.SetValue(data, "theme", "catppuccin")
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
		got, err := p.DeleteKey(data, "log_level")
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

func TestListKeys(t *testing.T) {
	p := &parser{}
	data := loadTestdata(t, "config.txt")
	keys := p.ListKeys(data)

	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}

	found := make(map[string]bool)
	for _, k := range keys {
		found[k] = true
	}
	for _, want := range []string{"theme", "log_level"} {
		if !found[want] {
			t.Errorf("missing key %q in ListKeys output", want)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	p := &parser{}
	data := loadTestdata(t, "config.txt")

	updated, err := p.SetValue(data, "some_key", "test")
	if err != nil {
		t.Fatal(err)
	}

	val, ok := p.FindValue(updated, "some_key")
	if !ok {
		t.Fatal("expected to find some_key after set")
	}
	if val != "test" {
		t.Errorf("got %q, want %q", val, "test")
	}

	// original keys still intact
	val, ok = p.FindValue(updated, "theme")
	if !ok {
		t.Fatal("expected theme to survive round-trip")
	}
	if val != "catppuccin" {
		t.Errorf("got %q, want %q", val, "catppuccin")
	}
}
