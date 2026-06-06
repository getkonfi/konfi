package konfi

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
		_, ok := p.FindValue(data, "konfi")
		if ok {
			t.Fatal("should not match inside comments")
		}
	})
}

func TestFindLine(t *testing.T) {
	p := newParser()
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
	p := newParser()
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
	p := newParser()
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
	p := newParser()
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
	p := newParser()
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

func TestRoundTripGolden(t *testing.T) {
	p := newParser()

	src := []byte(`# konfi settings
theme: catppuccin
log_level: info

# editor preferences
auto_save: true
backup: enabled
`)

	// step 1: modify an existing value
	out, err := p.SetValue(src, "theme", "tokyonight")
	if err != nil {
		t.Fatal(err)
	}
	v, ok := p.FindValue(out, "theme")
	if !ok || v != "tokyonight" {
		t.Fatalf("SetValue theme: got %q ok=%v", v, ok)
	}

	// step 2: add a new key
	out, err = p.SetValue(out, "sidebar_width", "30")
	if err != nil {
		t.Fatal(err)
	}
	v, ok = p.FindValue(out, "sidebar_width")
	if !ok || v != "30" {
		t.Fatalf("SetValue sidebar_width: got %q ok=%v", v, ok)
	}

	// step 3: verify comments survived
	if !bytes.Contains(out, []byte("# konfi settings")) {
		t.Error("comment line lost during round-trip")
	}
	if !bytes.Contains(out, []byte("# editor preferences")) {
		t.Error("second comment line lost during round-trip")
	}

	// step 4: verify untouched keys preserved
	for _, key := range []string{"log_level", "auto_save", "backup"} {
		if _, ok := p.FindValue(out, key); !ok {
			t.Errorf("key %q lost during round-trip", key)
		}
	}

	// step 5: verify empty lines preserved
	if !bytes.Contains(out, []byte("\n\n")) {
		t.Error("empty line lost during round-trip")
	}

	// step 6: ListKeys covers everything
	keys := p.ListKeys(out)
	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}
	if !keySet["sidebar_width"] {
		t.Error("ListKeys missing newly added sidebar_width")
	}
	if !keySet["theme"] {
		t.Error("ListKeys missing modified theme")
	}
}

func FuzzParser(f *testing.F) {
	f.Add([]byte("theme: catppuccin\nlog_level: info\n"), "theme")
	f.Add([]byte("# comment\nauto_save: true\n"), "auto_save")
	f.Add([]byte(""), "missing")
	f.Add([]byte("key: value with spaces\n"), "key")
	f.Add([]byte("a: b\nc: d\ne: f\n"), "c")
	f.Add([]byte("no-colon-here\n"), "no-colon-here")

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
